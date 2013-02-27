package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
)

var (
	flagTestdir = "tests"
	flagShowAll = false
)

var (
	parserCmd  string
	dirInvalid string
	dirValid   string
)

func init() {
	log.SetFlags(0)

	flag.StringVar(&flagTestdir, "testdir", flagTestdir,
		"The path to the test directory.")
	flag.BoolVar(&flagShowAll, "all", flagShowAll,
		"When set, all tests will be shown.")

	flag.Usage = usage
	flag.Parse()

	dirInvalid = path.Join(flagTestdir, "invalid")
	dirValid = path.Join(flagTestdir, "valid")
}

func usage() {
	log.Printf("Usage: %s parser-cmd [ test-name ... ]\n\n",
		path.Base(os.Args[0]))
	log.Println(
		`parser-cmd should be a program that accepts TOML data on stdin until EOF,
and outputs the corresponding JSON encoding on stdout.

Date times should be represented as JSON strings, in their full ISO8601 Zulu
form.

The 'testdir' directory should have two sub-directories: 'invalid' and 'valid'.

The 'invalid' directory should contain 'toml' files,
where test names are the file names not including the '.toml' suffix.

The 'valid' directory should contian 'toml' files and a 'json' file for each
'toml' file, that contains the expected output of 'parser-cmd'. Test names
are the file names not including the '.toml' or '.json' suffix.

Test names must be globally unique. Behavior is undefined if there is a
failure test with the same name as a valid test.

Flags:`)

	flag.PrintDefaults()

	os.Exit(1)
}

func main() {
	if flag.NArg() < 1 {
		flag.Usage()
	}
	parserCmd = flag.Arg(0)

	var results []result

	// Run all tests.
	if flag.NArg() == 1 {
		results = runAllTests()
	} else { // just a few
		results = make([]result, 0, flag.NArg()-1)
		for _, testName := range flag.Args()[1:] {
			results = append(results, runTestByName(testName))
		}
	}

	out := make([]string, 0, len(results))
	passed, failed := 0, 0
	for _, r := range results {
		if flagShowAll || r.failed() {
			out = append(out, r.String())
		}
		if r.failed() {
			failed++
		} else {
			passed++
		}
	}
	if len(out) > 0 {
		fmt.Println(strings.Join(out, "\n"+strings.Repeat("-", 79)+"\n"))
		fmt.Println("")
	}
	fmt.Printf("%d passed, %d failed\n", passed, failed)
}

func runAllTests() []result {
	invalidTests, err := ioutil.ReadDir(dirInvalid)
	if err != nil {
		log.Fatalf("Cannot read invalid directory (%s): %s", dirInvalid, err)
	}

	validTests, err := ioutil.ReadDir(dirValid)
	if err != nil {
		log.Fatalf("Cannot read valid directory (%s): %s", dirValid, err)
	}

	results := make([]result, 0, len(invalidTests)+len(validTests))
	for _, f := range invalidTests {
		if !strings.HasSuffix(f.Name(), ".toml") {
			continue
		}
		tname := stripSuffix(f.Name())
		results = append(results, runInvalidTest(tname))
	}
	for _, f := range validTests {
		if !strings.HasSuffix(f.Name(), ".toml") {
			continue
		}
		tname := stripSuffix(f.Name())
		results = append(results, runValidTest(tname))
	}
	return results
}

func runTestByName(name string) result {
	if readable(invPath("%s.toml", name)) {
		return runInvalidTest(name)
	}
	if readable(vPath("%s.toml", name)) && readable(vPath("%s.json", name)) {

		return runValidTest(name)
	}
	return result{testName: name}.errorf(
		"Could not find test in '%s' or '%s'.", dirInvalid, dirValid)
}

func readable(fp string) bool {
	_, err := os.Stat(fp)
	return err == nil
}

func vPath(fname string, v ...interface{}) string {
	return path.Join(dirValid, fmt.Sprintf(fname, v...))
}

func invPath(fname string, v ...interface{}) string {
	return path.Join(dirInvalid, fmt.Sprintf(fname, v...))
}

func stripSuffix(fname string) string {
	for _, suf := range []string{".toml", ".json"} {
		if ind := strings.LastIndex(fname, suf); ind > -1 {
			return fname[0:ind]
		}
	}
	return fname
}
