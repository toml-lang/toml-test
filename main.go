package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func usage() {
	log.Printf(`Usage: %[1]s parser-cmd [ test-name ... ]

toml-test is a tool to verify the correctness of TOML parsers.
https://github.com/BurntSushi/toml-test

The first positional argument (parser-cmd) should be a program that accepts TOML
data on stdin until EOF, and outputs the corresponding JSON encoding on stdout.
Please see 'README.md' for details on how to satisfy the interface expected by
'toml-test' with your own parser.

Any other positional arguments are tests to run. For example:

   $ %[1]s my-parser valid/test1 invalid/oh-noes

When omitted it will run all tests.

Note that flags *must* be placed before parser-cmd.

Flags:

    -encoder    The given executable will be tested as a TOML encoder.

                The binary will be sent JSON on stdin and write TOML to stdout.
                The JSON will be in the same format as specified in the
                toml-test README. Note that this depends on the correctness of
                my TOML parser! (For encoders, the same directory scheme above
                is used, except the 'invalid-encoder' directory is used instead
                of the 'invalid' directory.)

    -all        Show detailed input/output for all tests.

    -skip       Tests to skip, can add multiple by separating them with commas.
                Example:

                     $ %[1]s -skip valid/test1,invalid/oh-noes my-parser

    -no-bold    Don't output bold text in test failure messages.

    -testdir    Location of the tests; the default is to use the tests compiled
                in the binary; this is only useful if you want to work on
                writing tests.

                This should have two sub-directories: 'invalid' and 'valid'. The
                'invalid' directory contains 'toml' files, where test names are
                the file names not including the '.toml' suffix.

                The 'valid' directory contains 'toml' files and a 'json' file
                for each 'toml' file, which contains the expected output of
                'parser-cmd'. Test names are the file names not including the
                '.toml' or '.json' suffix.
`, path.Base(os.Args[0]))

	os.Exit(1)
}

var (
	flagEncoder = false
	flagNoBold  = false
	parserCmd   string
)

//go:embed tests/*
var packed embed.FS

var files fs.FS

var (
	dirValid   = "valid"
	dirInvalid = "invalid"
	invalidExt = "toml" // set to "json" when testing encoders
)

// TODO: Go's flag package is kinda crap; it *requires* flags to come before
// positional arguments. We can work around that though (many go commands do,
// like go test).
func parseFlags() (showAll bool, skip []string) {
	flagSkip := ""
	flagTestdir := ""

	log.SetFlags(0)
	flag.StringVar(&flagTestdir, "testdir", flagTestdir, "")
	flag.BoolVar(&showAll, "all", showAll, "")
	flag.BoolVar(&flagEncoder, "encoder", flagEncoder, "")
	flag.BoolVar(&flagNoBold, "no-bold", flagNoBold, "")
	flag.StringVar(&flagSkip, "skip", flagSkip, "")
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
	}

	parserCmd = flag.Arg(0)

	if fs := strings.TrimSpace(flagSkip); fs != "" {
		skip = strings.Split(fs, ",")
		for i := range skip {
			skip[i] = strings.TrimSpace(skip[i])
		}
	}

	if flagEncoder {
		dirInvalid = "invalid-encoder"
		invalidExt = "json"
	}

	var err error
	if flagTestdir != "" {
		files = os.DirFS(flagTestdir)
	} else {
		files, err = fs.Sub(packed, "tests")
	}
	if err != nil {
		log.Fatalln(err)
	}

	return showAll, skip
}

func main() {
	showAll, skip := parseFlags()

	var results []result
	if flag.NArg() == 1 {
		results = runAllTests(skip)
	} else {
		results = make([]result, 0, flag.NArg()-1)
		for _, n := range flag.Args()[1:] {
			results = append(results, runTestByName(n))
		}
	}

	var passed, failed, skipped int
	for _, r := range results {
		if showAll || r.failed() {
			fmt.Println(strings.TrimLeft(indent(r.String(), 4), " "))
			fmt.Println()
		}

		if r.failed() {
			failed++
		} else if r.skipped {
			skipped++
		} else {
			passed++
		}
	}

	fmt.Printf("toml-test %s: %3d passed, %2d failed", parserCmd, passed, failed)
	if skipped > 0 {
		fmt.Printf(", %2d skipped", skipped)
	}
	fmt.Println()

	if failed > 0 {
		os.Exit(1)
	}
}

func runAllTests(skip []string) []result {
	invalidTests, err := fs.ReadDir(files, dirInvalid)
	if err != nil {
		log.Fatalf("Cannot read invalid directory %q: %s", dirInvalid, err)
	}

	validTests, err := fs.ReadDir(files, dirValid)
	if err != nil {
		log.Fatalf("Cannot read valid directory %q: %s", dirValid, err)
	}

	results := make([]result, 0, len(invalidTests)+len(validTests))
	for _, f := range invalidTests {
		if !strings.HasSuffix(f.Name(), fmt.Sprintf(".%s", invalidExt)) {
			continue
		}
		tname := filepath.Join(dirInvalid, stripSuffix(f.Name()))
		if r, skipped := hasSkip(tname, skip); skipped {
			results = append(results, r)
			continue
		}
		results = append(results, runInvalidTest(tname))
	}
	for _, f := range validTests {
		if !strings.HasSuffix(f.Name(), ".toml") {
			continue
		}
		tname := filepath.Join(dirValid, stripSuffix(f.Name()))
		if r, skipped := hasSkip(tname, skip); skipped {
			results = append(results, r)
			continue
		}
		results = append(results, runValidTest(tname))
	}
	return results
}

func hasSkip(test string, skip []string) (result, bool) {
	for _, s := range skip {
		if s == test {
			return result{testName: test, skipped: true}, true
		}
	}
	return result{}, false
}

func runTestByName(name string) result {
	invalid := strings.Contains(name, dirInvalid+"/")
	if invalid {
		return runInvalidTest(name)
	}
	return runValidTest(name)
}

func readable(fp string) bool {
	_, err := os.Stat(fp)
	return err == nil
}

func stripSuffix(fname string) string {
	for _, suf := range []string{".toml", ".json"} {
		if ind := strings.LastIndex(fname, suf); ind > -1 {
			return fname[0:ind]
		}
	}
	return fname
}

func indent(s string, n int) string {
	sp := strings.Repeat(" ", n)
	return sp + strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "\n"+sp)
}
