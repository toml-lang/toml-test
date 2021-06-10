package main

import (
	"flag"
	"fmt"
	"go/build"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
)

var (
	flagTestdir = ""
	flagShowAll = false
	flagEncoder = false
	flagSkip    = ""
)

var (
	parserCmd  string
	dirInvalid string
	dirValid   string
	invalidExt = "toml" // set to "json" when testing encoders
)

func init() {
	log.SetFlags(0)

	// If no test directory was specified, let's look for it automatically.
	// Assumes `toml-test` was installed with the Go tool.
	if len(flagTestdir) == 0 {
		imp := path.Join("github.com", "BurntSushi", "toml-test", "tests")
		for _, dir := range build.Default.SrcDirs() {
			if readable(path.Join(dir, imp)) {
				flagTestdir = path.Join(dir, imp)
				break
			}
		}
	}

	// Nada, just use 'tests'.
	if len(flagTestdir) == 0 {
		flagTestdir = "tests"
	}

	flag.StringVar(&flagTestdir, "testdir", flagTestdir,
		"The path to the test directory.")
	flag.BoolVar(&flagShowAll, "all", flagShowAll,
		"When set, all tests will be shown.")
	flag.BoolVar(&flagEncoder, "encoder", flagEncoder,
		"When set, the given executable will be tested as a TOML encoder.")
	flag.StringVar(&flagSkip, "skip", flagSkip,
		"Tests to skip, comma-separated and as e.g. 'invalid/test-name'")

	flag.Usage = usage
	flag.Parse()

	dirValid = path.Join(flagTestdir, "valid")
	if flagEncoder {
		dirInvalid = path.Join(flagTestdir, "invalid-encoder")
		invalidExt = "json"
	} else {
		dirInvalid = path.Join(flagTestdir, "invalid")
	}
}

func usage() {
	log.Printf("Usage: %s parser-cmd [ test-name ... ]\n",
		path.Base(os.Args[0]))
	log.Println(`
parser-cmd should be a program that accepts TOML data on stdin until EOF,
and outputs the corresponding JSON encoding on stdout. Please see 'README.md'
for details on how to satisfy the interface expected by 'toml-test' with your
own parser.

The 'testdir' directory should have two sub-directories: 'invalid' and 'valid'.

The 'invalid' directory should contain 'toml' files,
where test names are the file names not including the '.toml' suffix.

The 'valid' directory should contain 'toml' files and a 'json' file for each
'toml' file, that contains the expected output of 'parser-cmd'. Test names
are the file names not including the '.toml' or '.json' suffix.

Test names must be globally unique. Behavior is undefined if there is a
failure test with the same name as a valid test.

Note that toml-test can also test TOML encoders with the "encoder" flag set.
In particular, the binary will be given JSON on stdin and expect TOML on
stdout. The JSON will be in the same format as specified in the toml-test
README. Note that this depends on the correctness of my TOML parser!
(For encoders, the same directory scheme above is used, except the
'invalid-encoder' directory is used instead of the 'invalid' directory.)

Flags:`)

	flag.PrintDefaults()

	os.Exit(1)
}

func main() {
	if flag.NArg() < 1 {
		flag.Usage()
	}
	parserCmd = flag.Arg(0)

	var skip []string
	if fs := strings.TrimSpace(flagSkip); fs != "" {
		skip = strings.Split(fs, ",")
		for i := range skip {
			skip[i] = strings.TrimSpace(skip[i])
		}
	}

	var results []result

	// Run all tests.
	if flag.NArg() == 1 {
		results = runAllTests(skip)
	} else { // just a few
		results = make([]result, 0, flag.NArg()-1)
		for _, testName := range flag.Args()[1:] {
			results = append(results, runTestByName(testName))
		}
	}

	out := make([]string, 0, len(results))
	var passed, failed, skipped int
	for _, r := range results {
		if flagShowAll || r.failed() {
			out = append(out, r.String())
		}
		if r.failed() {
			failed++
		} else if r.skipped {
			skipped++
		} else {
			passed++
		}
	}
	if len(out) > 0 {
		fmt.Println(strings.Join(out, "\n"+strings.Repeat("-", 79)+"\n"))
		fmt.Println("")
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
		if !strings.HasSuffix(f.Name(), fmt.Sprintf(".%s", invalidExt)) {
			continue
		}
		tname := stripSuffix(f.Name())
		if r, skipped := hasSkip("invalid/"+tname, skip); skipped {
			results = append(results, r)
			continue
		}
		results = append(results, runInvalidTest(tname))
	}
	for _, f := range validTests {
		if !strings.HasSuffix(f.Name(), ".toml") {
			continue
		}
		tname := stripSuffix(f.Name())
		if r, skipped := hasSkip("valid/"+tname, skip); skipped {
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
	if readable(invPath("%s.%s", name, invalidExt)) {
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
