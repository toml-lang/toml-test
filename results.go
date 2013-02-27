package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

type result struct {
	testName string
	err error
	valid bool
	failure string
	key string
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

func (r result) path() string {
	if r.valid {
		return vPath("%s.toml", r.testName)
	}
	return invPath("%s.toml", r.testName)
}

func (r result) jsonPath() string {
	if !r.valid {
		panic("Cannot call `jsonPath` on invalid test.")
	}
	return vPath("%s.json", r.testName)
}

func runInvalidTest(name string) result {
	r := result{
		testName: name,
		valid: false,
	}

	_, stderr, err := runParser(r.path())
	if err != nil {
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
		valid: true,
	}

	jsonExpected, err := loadJson(r.jsonPath())
	if err != nil {
		return r.errorf(err.Error())
	}

	stdout, stderr, err := runParser(r.path())
	if err != nil {
		return r.errorf(err.Error())
	}
	if stderr != nil { // test has failed :-(
		return r.failedf(stderr.String())
	}
	if stdout == nil {
		panic("BUG: stdout and stderr are both nil.")
	}

	var jsonTest interface{}
	if err := json.NewDecoder(stdout).Decode(&jsonTest); err != nil {
		return r.errorf(
			"Could not decode JSON output from parser: %s", err)
	}

	return r.cmpJson(jsonExpected, jsonTest)
}

func runParser(tomlFile string) (*bytes.Buffer, *bytes.Buffer, error) {
	f, err := os.Open(tomlFile)
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
		return nil, stderr, nil
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
