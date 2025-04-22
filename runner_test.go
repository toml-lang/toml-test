package tomltest

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"
)

func notInList(t *testing.T, list []string, str string) {
	t.Helper()
	for _, item := range list {
		if item == str {
			t.Fatalf("error: %q in list", str)
		}
	}
}

func TestVersion(t *testing.T) {
	_, err := Runner{Version: "0.9", Files: os.DirFS("./tests")}.Run()
	if err == nil {
		t.Fatal("expected an error for version 0.9")
	}

	r := Runner{Version: "1.0.0", Files: os.DirFS("./tests")}
	ls, err := r.List()
	if err != nil {
		t.Fatal()
	}
	notInList(t, ls, "valid/string/escape-esc")

	r = Runner{Version: "1.0.0", Files: os.DirFS("./tests")}
	ls, err = r.List()
	if err != nil {
		t.Fatal()
	}
	notInList(t, ls, "valid/string/escape-esc")
}

type testParser struct{}

func (t *testParser) Encode(ctx context.Context, input string) (output string, outputIsError bool, err error) {
	switch input {
	case `a=1`:
		return `{"a": {"type":"integer","value":"1"}}`, false, nil
	case `a=`, `c=`:
		return `oh noes: error one`, true, nil
	case `b=`:
		return `error two`, true, nil
	default:
		panic(fmt.Sprintf("unreachable: %q", input))
	}
}

func (t testParser) Decode(ctx context.Context, input string) (string, bool, error) {
	return t.Encode(ctx, input)
}

func TestErrors(t *testing.T) {
	r := Runner{
		Parser: &testParser{},
		Files: fstest.MapFS{
			"valid/a.toml":       &fstest.MapFile{Data: []byte(`a=1`)},
			"valid/a.json":       &fstest.MapFile{Data: []byte(`{"a": {"type":"integer","value":"1"}}`)},
			"invalid/a.toml":     &fstest.MapFile{Data: []byte(`a=`)},
			"invalid/b.toml":     &fstest.MapFile{Data: []byte(`b=`)},
			"invalid/dir/c.toml": &fstest.MapFile{Data: []byte(`c=`)},
		},
		Errors: map[string]string{
			"invalid/a":  "oh noes",
			"invalid/b":  "don't match",
			"dir/c.toml": "oh noes",
		},
	}
	tt, err := r.Run()
	if err != nil {
		t.Error(err)
	}
	for _, test := range tt.Tests {
		if test.Path == "invalid/b" {
			if !test.Failed() {
				t.Errorf("expected failure for %q, but got none", test.Path)
			}
			continue
		}

		if test.Failed() {
			t.Errorf("\n%s: %s", test.Path, test.Failure)
		}
	}

	t.Run("non-existent", func(t *testing.T) {
		r := Runner{
			Parser: &testParser{},
			Files:  fstest.MapFS{},
			Errors: map[string]string{
				"file/doesn/exist": "oh noes",
			},
		}
		_, err := r.Run()
		if err == nil {
			t.Fatal("error is nil")
		}
		if !strings.Contains(err.Error(), "didn't match anything") {
			t.Fatalf("wrong error: %s", err)
		}
	})
}

func TestSkip(t *testing.T) {
	r := Runner{
		Parser:    &testParser{},
		SkipTests: []string{"valid/a"},
		Files: fstest.MapFS{
			"valid/a.toml": &fstest.MapFile{Data: []byte(`a=`)},
		},
	}
	tests, err := r.Run()
	if err != nil {
		t.Fatal(err)
	}
	if tests.FailedValid != 0 || tests.Skipped != 1 {
		t.Fatalf("FailedValid=%d; Skipped=%d", tests.FailedValid, tests.Skipped)
	}
}

func TestSkipMustError(t *testing.T) {
	r := Runner{
		Parser:        &testParser{},
		SkipMustError: true,
		SkipTests:     []string{"valid/a"},
		Files: fstest.MapFS{
			"valid/a.toml": &fstest.MapFile{Data: []byte(`a=1`)},
			"valid/a.json": &fstest.MapFile{Data: []byte(`{"a": {"type":"integer","value":"1"}}`)},
		},
	}
	tests, err := r.Run()
	if err != nil {
		t.Fatal(err)
	}
	if tests.FailedValid != 1 || tests.Skipped != 0 {
		t.Fatalf("FailedValid=%d; Skipped=%d", tests.FailedValid, tests.Skipped)
	}
	if tests.Tests[0].Failure != "Test skipped with -skip but didn't fail" {
		t.Errorf("wrong failure message: %q", tests.Tests[0].Failure)
	}
}
