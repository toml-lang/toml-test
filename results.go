package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os/exec"
	"strings"

	"github.com/BurntSushi/toml"
)

type result struct {
	testName string
	err      error
	valid    bool
	skipped  bool
	failure  string
	key      string

	// Input is the test, output is whatever the tested tool returned.
	input, output, want string
	fromStderr          bool
}

func (r result) errorf(format string, v ...interface{}) result {
	r.err = fmt.Errorf(format, v...)
	return r
}

func (r result) bugf(format string, v ...interface{}) result {
	return r.failedf("BUG IN TEST CASE: "+format, v...)
}

func (r result) failedf(format string, v ...interface{}) result {
	r.failure = fmt.Sprintf(format, v...)
	return r
}

func (r result) mismatch(wantType string, want, have interface{}) result {
	return r.failedf("Key '%s' is not an %s but %[4]T:\n"+
		"  Expected:     %#[3]v\n"+
		"  Your encoder: %#[4]v",
		r.key, wantType, want, have)
}

func (r result) valMismatch(wantType, haveType string, want, have interface{}) result {
	return r.failedf("Key '%s' is not an %s but %s:\n"+
		"  Expected:     %#[3]v\n"+
		"  Your encoder: %#[4]v",
		r.key, wantType, want, have)
}

func (r result) kjoin(key string) result {
	if len(r.key) == 0 {
		r.key = key
	} else {
		r.key += "." + key
	}
	return r
}

func (r result) failed() bool {
	return r.err != nil || len(r.failure) > 0
}

func (r result) pathTest() string {
	ext := "toml"
	if flagEncoder {
		ext = "json"
	}
	if r.valid {
		return fmt.Sprintf("%s.%s", r.testName, ext)
	}
	return fmt.Sprintf("%s.%s", r.testName, ext)
}

func (r result) pathGold() string {
	if !r.valid {
		panic("Invalid tests do not have a 'correct' version.")
	}
	if flagEncoder {
		return fmt.Sprintf("%s.toml", r.testName)
	}
	return fmt.Sprintf("%s.json", r.testName)
}

func runInvalidTest(name string) result {
	r := result{
		testName: name,
		valid:    false,
	}

	_, stderr, err := runParser(&r)
	if err != nil {
		// Errors here are OK if it's just an exit error.
		if _, ok := err.(*exec.ExitError); ok {
			return r
		}

		// Otherwise, something has gone horribly wrong.
		return r.errorf(err.Error())
	}
	if stderr != nil { // test has passed!
		return r
	}
	return r.failedf("Expected an error, but no error was reported.")
}

func runValidTest(name string) result {
	r := result{testName: name, valid: true}

	stdout, stderr, err := runParser(&r)
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			switch {
			case stderr != nil && stderr.Len() > 0:
				return r.failedf(stderr.String())
			case stdout != nil && stdout.Len() > 0:
				return r.failedf(stdout.String())
			}
		}
		return r.errorf(err.Error())
	}

	if stdout == nil {
		return r.errorf("stdout is empty but %q exited successfully", parserCmd)
	}

	wantB, err := fs.ReadFile(files, r.pathGold())
	if err != nil {
		return r.bugf(err.Error())
	}
	r.want = string(wantB)

	if flagEncoder {
		want, err := loadTOML(r)
		if err != nil {
			return r.bugf(err.Error())
		}

		var have interface{}
		if _, err := toml.Decode(r.output, &have); err != nil {
			return r.errorf("decode TOML from encoder %q:\n  %s", parserCmd, err)
		}
		return r.cmpTOML(want, have)
	}

	want, err := loadJSON(r)
	if err != nil {
		return r.errorf(err.Error())
	}

	var have interface{}
	if err := json.Unmarshal([]byte(r.output), &have); err != nil {
		return r.errorf("decode JSON output from parser:\n  %s", err)
	}

	return r.cmpJSON(want, have)
}

func runParser(r *result) (*bytes.Buffer, *bytes.Buffer, error) {
	data, err := fs.ReadFile(files, r.pathTest())
	if err != nil {
		return nil, nil, err
	}

	r.input = string(data)

	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	c := exec.Command(parserCmd)
	c.Stdin, c.Stdout, c.Stderr = bytes.NewReader(data), stdout, stderr

	err = c.Run() // Error checked later

	if stderr.Len() > 0 {
		r.output = strings.TrimSpace(stderr.String()) + "\n"
		r.fromStderr = true
	} else {
		r.output = strings.TrimSpace(stdout.String()) + "\n"
	}

	if err != nil {
		return stdout, stderr, err
	}
	return stdout, nil, nil
}

func loadJSON(r result) (interface{}, error) {
	var vjson interface{}
	if err := json.Unmarshal([]byte(r.want), &vjson); err != nil {
		return nil, fmt.Errorf("Could not decode JSON file %q:\n  %s", r.pathGold(), err)
	}
	return vjson, nil
}

func loadTOML(r result) (interface{}, error) {
	var vtoml interface{}
	_, err := toml.Decode(r.want, &vtoml)
	if err != nil {
		return nil, fmt.Errorf("Could not decode TOML file %q:\n  %s", r.pathGold(), err)
	}
	return vtoml, nil
}

func (r result) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(bold(fmt.Sprintf("Test: %s", r.testName)))
	buf.WriteString(fmt.Sprintf("  (%s < %s)\n", parserCmd, r.pathTest()))

	if r.failure == "" && r.err == nil {
		buf.WriteString("PASSED")
		return buf.String()
	}

	msg := r.failure
	if r.err != nil {
		msg = r.err.Error()
	}

	buf.WriteString(msg)
	showStream(buf, "input sent to "+parserCmd, r.input)
	if r.fromStderr {
		showStream(buf, "output from "+parserCmd+" (stderr)", r.output)
	} else {
		showStream(buf, "output from "+parserCmd+" (stdout)", r.output)
	}
	showStream(buf, "want", r.want)
	buf.WriteByte('\n')
	return buf.String()
}

func showStream(buf *bytes.Buffer, name, s string) {
	buf.WriteByte('\n')

	fmt.Fprintln(buf, bold("    "+name+":"))
	if s == "" {
		fmt.Fprintln(buf, "        <empty>")
		return
	}

	fmt.Fprintln(buf, indent(s, 8))
}

func bold(s string) string {
	if flagNoBold {
		return s
	}
	return "\x1b[1m" + s + "\x1b[0m"
}
