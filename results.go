package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/BurntSushi/toml"

	"os"
	"os/exec"
)

type result struct {
	testName string
	err      error
	valid    bool
	failure  string
	key      string
}

func (r result) errorf(format string, v ...interface{}) result {
	r.err = fmt.Errorf(format, v...)
	return r
}

func (r result) failedf(format string, v ...interface{}) result {
	r.failure = fmt.Sprintf(format, v...)
	return r
}

func (r result) mismatch(expected string, got interface{}) result {
	return r.failedf("Type mismatch for key '%s'. Expected %s but got %T.",
		r.key, expected, got)
}

func (r result) valMismatch(expected string, got string) result {
	return r.failedf("Type mismatch for key '%s'. Expected %s but got %s.",
		r.key, expected, got)
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
		return vPath("%s.%s", r.testName, ext)
	}
	return invPath("%s.%s", r.testName, ext)
}

func (r result) pathGold() string {
	if !r.valid {
		panic("Invalid tests do not have a 'correct' version.")
	}
	if flagEncoder {
		return vPath("%s.toml", r.testName)
	}
	return vPath("%s.json", r.testName)
}

func runInvalidTest(name string) result {
	r := result{
		testName: name,
		valid:    false,
	}

	_, stderr, err := runParser(r.pathTest())
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
	r := result{
		testName: name,
		valid:    true,
	}

	stdout, stderr, err := runParser(r.pathTest())
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
		return r.errorf("Parser does not satisfy interface. stdout is " +
			"empty, but the process exited successfully.")
	}

	if flagEncoder {
		tomlExpected, err := loadToml(r.pathGold())
		if err != nil {
			return r.errorf(err.Error())
		}
		var tomlTest interface{}
		if _, err := toml.DecodeReader(stdout, &tomlTest); err != nil {
			return r.errorf(
				"Could not decode TOML output from encoder: %s", err)
		}
		return r.cmpToml(tomlExpected, tomlTest)
	} else {
		jsonExpected, err := loadJson(r.pathGold())
		if err != nil {
			return r.errorf(err.Error())
		}
		var jsonTest interface{}
		if err := json.NewDecoder(stdout).Decode(&jsonTest); err != nil {
			return r.errorf(
				"Could not decode JSON output from parser: %s", err)
		}
		return r.cmpJson(jsonExpected, jsonTest)
	}
}

func runParser(testFile string) (*bytes.Buffer, *bytes.Buffer, error) {
	f, err := os.Open(testFile)
	if err != nil {
		return nil, nil, err
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	c := exec.Command(parserCmd)
	c.Stdin = f
	c.Stdout = stdout
	c.Stderr = stderr

	if err := c.Run(); err != nil {
		return stdout, stderr, err
	}
	return stdout, nil, nil
}

func loadJson(fp string) (interface{}, error) {
	fjson, err := os.Open(fp)
	if err != nil {
		return nil, fmt.Errorf(
			"Could not find expected JSON output at %s.", fp)
	}

	var vjson interface{}
	if err := json.NewDecoder(fjson).Decode(&vjson); err != nil {
		return nil, fmt.Errorf(
			"Could not decode expected JSON output at %s: %s", fp, err)
	}
	return vjson, nil
}

func loadToml(fp string) (interface{}, error) {
	var vtoml interface{}
	if _, err := toml.DecodeFile(fp, &vtoml); err != nil {
		return nil, fmt.Errorf(
			"Could not decode expected TOML output at %s: %s", fp, err)
	}
	return vtoml, nil
}

func (r result) String() string {
	buf := new(bytes.Buffer)
	p := func(s string, v ...interface{}) { fmt.Fprintf(buf, s, v...) }

	validStr := "invalid"
	if r.valid {
		validStr = "valid"
	}
	p("Test: %s (%s)\n\n", r.testName, validStr)

	if r.err != nil {
		p("Error running test: %s", r.err)
		return buf.String()
	}
	if len(r.failure) > 0 {
		p(r.failure)
		return buf.String()
	}

	p("PASSED.")
	return buf.String()
}
