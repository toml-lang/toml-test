package tomltest

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type testType uint8

const (
	TypeValid testType = iota
	TypeInvalid
)

//go:embed tests/*
var embeddedTests embed.FS

// EmbeddedTests are the tests embedded in toml-test, rooted to the "test/"
// directory.
func EmbeddedTests() fs.FS {
	f, err := fs.Sub(embeddedTests, "tests")
	if err != nil {
		panic(err)
	}
	return f
}

// Runner runs a set of tests.
//
// The validity of the parameters is not checked extensively; the caller should
// verify this if need be. See ./cmd/toml-test for an example.
type Runner struct {
	Files     fs.FS    // Test files.
	Encoder   bool     // Are we testing an encoder?
	ParserCmd []string // The parser command.
	RunTests  []string // Tests to run; run all if blank.
	SkipTests []string // Tests to skip.
}

// Tests are tests to run.
type Tests struct {
	Tests []Test

	// Set when test are run.

	Skipped, Passed, Failed int
}

// Result is the result of a single test.
type Test struct {
	Path string // Path of test, e.g. "valid/string-test"

	// Set when a test is run.

	Skipped          bool   // Skipped this test?
	Failure          string // Failure message.
	Key              string // TOML key the failure occured on; may be blank.
	Encoder          bool   // Encoder test?
	Input            string // The test case that we sent to the external program.
	Output           string // Output from the external program.
	Want             string // The output we want.
	OutputFromStderr bool   // The Output came from stderr, not stdout.
}

// List all tests in Files.
func (r Runner) List() ([]string, error) {
	ls := make([]string, 0, 256)
	if err := r.findTOML("valid", &ls); err != nil {
		return nil, fmt.Errorf("reading 'valid/' dir: %w", err)
	}

	d := "invalid" + map[bool]string{true: "-encoder", false: ""}[r.Encoder]
	if err := r.findTOML(d, &ls); err != nil {
		return nil, fmt.Errorf("reading 'invalid/' dir: %w", err)
	}
	return ls, nil
}

// Run all tests listed in t.RunTests.
func (r Runner) Run() (Tests, error) {
	skipped, err := r.findTests()
	if err != nil {
		return Tests{}, fmt.Errorf("tomltest.Runner.Run: %w", err)
	}

	tests := Tests{Tests: make([]Test, 0, len(r.RunTests)), Skipped: skipped}
	for _, p := range r.RunTests {
		if r.hasSkip(p) {
			tests.Skipped++
			tests.Tests = append(tests.Tests, Test{Path: p, Skipped: true, Encoder: r.Encoder})
			continue
		}

		t := Test{Path: p, Encoder: r.Encoder}.Run(r.Files, r.ParserCmd)
		tests.Tests = append(tests.Tests, t)

		if t.Failed() {
			tests.Failed++
		} else {
			tests.Passed++
		}
	}

	return tests, nil
}

// find all TOML files in 'path' relative to the test directory.
func (r Runner) findTOML(path string, appendTo *[]string) error {
	return fs.WalkDir(r.Files, path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".toml") {
			return nil
		}

		*appendTo = append(*appendTo, strings.TrimSuffix(path, ".toml"))
		return nil
	})
}

// Expand RunTest glob patterns, or return all tests if RunTests if empty.
func (r *Runner) findTests() (int, error) {
	ls, err := r.List()
	if err != nil {
		return 0, err
	}

	if len(r.RunTests) == 0 {
		r.RunTests = ls
		return 0, nil
	}

	run := make([]string, 0, len(r.RunTests))
	for _, l := range ls {
		for _, r := range r.RunTests {
			if m, _ := filepath.Match(r, l); m {
				run = append(run, l)
				break
			}
		}
	}
	r.RunTests = run
	return len(ls) - len(run), nil
}

func (r Runner) hasSkip(path string) bool {
	for _, s := range r.SkipTests {
		if m, _ := filepath.Match(s, path); m {
			return true
		}
	}
	return false
}

// Run this test.
func (t Test) Run(fsys fs.FS, cmd []string) Test {
	if t.Type() == TypeInvalid {
		return t.runInvalid(fsys, cmd)
	}
	return t.runValid(fsys, cmd)
}

func (t Test) runInvalid(fsys fs.FS, cmd []string) Test {
	_, stderr, err := t.runParser(fsys, cmd)
	if err != nil {
		// We expect an exit of >0, so this is good.
		if _, ok := err.(*exec.ExitError); ok {
			return t
		}
		return t.fail(err.Error())
	}
	if stderr != nil { // We expect some error.
		return t
	}
	return t.fail("Expected an error, but no error was reported.")
}

func (t Test) runValid(fsys fs.FS, cmd []string) Test {
	stdout, stderr, err := t.runParser(fsys, cmd)
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			switch {
			case stderr != nil && stderr.Len() > 0:
				return t.fail(stderr.String())
			case stdout != nil && stdout.Len() > 0:
				return t.fail(stdout.String())
			}
		}
		return t.fail(err.Error())
	}
	if stdout == nil {
		return t.fail("stdout is empty but %q exited successfully", cmd)
	}

	// Compare for encoder test
	if t.Encoder {
		want, err := t.ReadWantTOML(fsys)
		if err != nil {
			return t.bug(err.Error())
		}
		var have interface{}
		if _, err := toml.Decode(t.Output, &have); err != nil {
			return t.fail("decode TOML from encoder %q:\n  %s", cmd, err)
		}
		return t.cmpTOML(want, have)
	}

	// Compare for decoder test
	want, err := t.ReadWantJSON(fsys)
	if err != nil {
		return t.fail(err.Error())
	}

	var have interface{}
	if err := json.Unmarshal([]byte(t.Output), &have); err != nil {
		return t.fail("decode JSON output from parser:\n  %s", err)
	}

	return t.cmpJSON(want, have)
}

// Run the parser (e.g. toml-test-decode or toml-test-encode), set Input and
// Output on Test, and return std{out,err}.
func (t *Test) runParser(fsys fs.FS, cmd []string) (*bytes.Buffer, *bytes.Buffer, error) {
	var err error
	_, t.Input, err = t.ReadInput(fsys)
	if err != nil {
		return nil, nil, err
	}

	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	c := exec.Command(cmd[0])
	c.Args = cmd
	c.Stdin, c.Stdout, c.Stderr = strings.NewReader(t.Input), stdout, stderr

	err = c.Run() // Error checked later

	if stderr.Len() > 0 {
		t.Output = strings.TrimSpace(stderr.String()) + "\n"
		t.OutputFromStderr = true
	} else {
		t.Output = strings.TrimSpace(stdout.String()) + "\n"
	}

	if err != nil {
		return stdout, stderr, err
	}
	return stdout, nil, nil
}

// ReadInput reads the file sent to the encoder.
func (t Test) ReadInput(fsys fs.FS) (path, data string, err error) {
	path = t.Path + map[bool]string{true: ".json", false: ".toml"}[t.Encoder]
	d, err := fs.ReadFile(fsys, path)
	if err != nil {
		return path, "", err
	}
	return path, string(d), nil
}

func (t Test) ReadWant(fsys fs.FS) (path, data string, err error) {
	if t.Type() == TypeInvalid {
		panic("testoml.Test.ReadWant: invalid tests do not have a 'correct' version")
	}

	path = t.Path + map[bool]string{true: ".toml", false: ".json"}[t.Encoder]
	d, err := fs.ReadFile(fsys, path)
	if err != nil {
		return path, "", err
	}
	return path, string(d), nil
}

func (t *Test) ReadWantJSON(fsys fs.FS) (v interface{}, err error) {
	var path string
	path, t.Want, err = t.ReadWant(fsys)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(t.Want), &v); err != nil {
		return nil, fmt.Errorf("decode JSON file %q:\n  %s", path, err)
	}
	return v, nil
}
func (t *Test) ReadWantTOML(fsys fs.FS) (v interface{}, err error) {
	var path string
	path, t.Want, err = t.ReadWant(fsys)
	if err != nil {
		return nil, err
	}
	_, err = toml.Decode(t.Want, &v)
	if err != nil {
		return nil, fmt.Errorf("Could not decode TOML file %q:\n  %s", path, err)
	}
	return v, nil
}

// Test type: "valid", "invalid"
func (t Test) Type() testType {
	if strings.HasPrefix(t.Path, "invalid") {
		return TypeInvalid
	}
	return TypeValid
}

func (t Test) fail(format string, v ...interface{}) Test {
	t.Failure = fmt.Sprintf(format, v...)
	return t
}
func (t Test) bug(format string, v ...interface{}) Test {
	return t.fail("BUG IN TEST CASE: "+format, v...)
}

func (t Test) Failed() bool { return t.Failure != "" }
