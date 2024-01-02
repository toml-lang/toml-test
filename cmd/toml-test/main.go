package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	tomltest "github.com/toml-lang/toml-test"
	"zgo.at/zli"
)

var hlErr = zli.Color256(224).Bg() | zli.Color256(0) | zli.Bold

func parseFlags() (tomltest.Runner, []string, int, string, bool, bool) {
	f := zli.NewFlags(os.Args)
	var (
		help        = f.Bool(false, "help", "h")
		versionFlag = f.IntCounter(0, "version", "V")
		tomlVersion = f.String("1.0.0", "toml")
		encoder     = f.Bool(false, "encoder")
		testDir     = f.String("", "testdir")
		showAll     = f.IntCounter(0, "v")
		color       = f.String("always", "color")
		skip        = f.StringList(nil, "skip")
		run         = f.StringList(nil, "run")
		listFiles   = f.Bool(false, "list-files")
		cat         = f.Int(0, "cat")
		copyFiles   = f.Bool(false, "copy")
		parallel    = f.Int(runtime.NumCPU(), "parallel")
		printSkip   = f.Bool(false, "print-skip")
		intAsFloat  = f.Bool(false, "int-as-float")
		errors      = f.String("", "errors")
		timeout     = f.String("1s", "timeout")
	)
	zli.F(f.Parse())
	if help.Bool() {
		fmt.Printf(usage, filepath.Base(os.Args[0]))
		zli.Exit(0)
	}
	if versionFlag.Int() > 0 {
		zli.PrintVersion(versionFlag.Int() > 1)
		zli.Exit(0)
	}
	fsys := getFS(testDir.String(), testDir.Set())
	if cat.Set() {
		doCat(fsys, tomlVersion.String(), cat.Int(), run.Strings(), skip.Strings())
		zli.Exit(0)
	}
	if copyFiles.Set() {
		doCopy(fsys, tomlVersion.String(), f.Args)
		zli.Exit(0)
	}

	dur, err := time.ParseDuration(timeout.String())
	zli.F(err)

	var errs map[string]string
	if errors.Set() {
		fp, err := os.Open(errors.String())
		zli.F(err)
		func() {
			defer fp.Close()
			if strings.HasSuffix(errors.String(), ".json") {
				err = json.NewDecoder(fp).Decode(&errs)
			} else {
				_, err = toml.NewDecoder(fp).Decode(&errs)
			}
			zli.F(err)
		}()
	}

	r := tomltest.Runner{
		Encoder:    encoder.Bool(),
		RunTests:   run.StringsSplit(","),
		SkipTests:  skip.StringsSplit(","),
		Version:    tomlVersion.String(),
		Parallel:   parallel.Int(),
		Files:      fsys,
		Parser:     tomltest.NewCommandParser(fsys, f.Args),
		Timeout:    dur,
		IntAsFloat: intAsFloat.Bool(),
		Errors:     errs,
	}

	if len(f.Args) == 0 && !listFiles.Bool() {
		zli.Fatalf("no parser command")
	}
	for _, r := range r.RunTests {
		_, err := filepath.Match(r, "")
		if err != nil {
			zli.Fatalf("invalid glob pattern %q in -run: %s", r, err)
		}
	}
	for _, r := range r.SkipTests {
		_, err := filepath.Match(r, "")
		if err != nil {
			zli.Fatalf("invalid glob pattern %q in -skip: %s", r, err)
		}
	}

	_, ok := os.LookupEnv("NO_COLOR")
	zli.WantColor = !ok
	if color.Set() {
		switch color.String() {
		case "always", "yes":
			zli.WantColor = true
		case "never", "no", "none", "off":
			zli.WantColor = false
		case "bold", "monochrome":
			zli.WantColor = true
			hlErr = zli.Bold
		default:
			zli.Fatalf("invalid value for -color: %q", color)
		}
	}

	return r, f.Args, showAll.Int(), testDir.String(), listFiles.Bool(), printSkip.Bool()
}

func getFS(testDir string, set bool) fs.FS {
	fsys := tomltest.EmbeddedTests()
	if set {
		fsys = os.DirFS(testDir)

		// So I used the path to toml-dir a few times, be forgiving by looking
		// for a "tests" directory and sub-ing to that.
		ls, err := fs.ReadDir(fsys, ".")
		zli.F(err)

		var f, t int
		for _, l := range ls {
			if l.IsDir() && (l.Name() == "valid" || l.Name() == "invalid") {
				f++
			}
			if l.IsDir() && l.Name() == "tests" {
				t++
			}
		}
		if f < 2 {
			if t == 0 {
				zli.Fatalf("%q does not seem to contain any tests (no valid and/or invalid directory)", testDir)
			}
			fsys, err = fs.Sub(fsys, "tests")
			zli.F(err)
		}
	}
	return fsys
}

func getList(r tomltest.Runner) []string {
	l, err := r.List()
	zli.F(err)

	sort.Strings(l)
	n := make([]string, 0, len(l)*2)
	for _, ll := range l {
		if strings.HasPrefix(ll, "valid/") {
			n = append(n, ll+".json")
		}
		n = append(n, ll+".toml")
	}
	return n
}

func main() {
	runner, cmd, showAll, testDir, listFiles, printSkip := parseFlags()

	if listFiles {
		l := getList(runner)
		for _, ll := range l {
			fmt.Println(ll)
		}
		return
	}

	tests, err := runner.Run()
	zli.F(err)

	for _, t := range tests.Tests {
		if t.Failed() || showAll > 1 {
			fmt.Print(detailed(runner, t))
		} else if showAll == 1 {
			fmt.Print(short(runner, t))
		}
	}

	_, _, date := zli.GetVersion()
	fmt.Printf("toml-test v%s %s: ", date.Format("2006-01-02"), cmd)
	if testDir == "" {
		fmt.Print("using embedded tests")
	} else {
		fmt.Printf("tests from %q", testDir)
	}
	if tests.Skipped > 0 {
		fmt.Printf(", %2d skipped", tests.Skipped)
	}
	if printSkip && (tests.FailedValid > 0 || tests.FailedInvalid > 0) {
		fmt.Print("\n\n    #!/usr/bin/env bash\n    skip=(\n")
		for _, f := range tests.Tests {
			if f.Failed() {
				fmt.Printf("        -skip '%s'\n", f.Path)
			}
		}
		fmt.Println("    )")
		fmt.Print("    toml-test $skip[@] " + strings.Join(cmd, " "))
		if runner.Encoder {
			fmt.Print(" -encoder")
		}
		fmt.Println()
	}

	fmt.Println()
	if runner.Encoder {
		fmt.Printf("encoder tests: %3d passed, %2d failed\n", tests.PassedValid, tests.FailedValid)
	} else {
		fmt.Printf("  valid tests: %3d passed, %2d failed\n", tests.PassedValid, tests.FailedValid)
		fmt.Printf("invalid tests: %3d passed, %2d failed\n", tests.PassedInvalid, tests.FailedInvalid)
	}

	if tests.FailedValid > 0 || tests.FailedInvalid > 0 {
		zli.Exit(1)
	}
	zli.Exit(0)
}

func short(r tomltest.Runner, t tomltest.Test) string {
	b := new(strings.Builder)

	switch {
	case t.Failure != "":
		b.WriteString(zli.Colorize("FAIL", hlErr))
		b.WriteByte(' ')
		b.WriteString(zli.Bold.String())
		b.WriteString(t.Path)
		b.WriteString(zli.Reset.String())
	case t.Skipped:
		b.WriteString(hlErr.String())
		b.WriteString("SKIP")
		b.WriteString(zli.Reset.String())
		b.WriteByte(' ')
		b.WriteString(t.Path)
	default:
		b.WriteString("PASS ")
		b.WriteString(t.Path)
	}

	b.WriteByte('\n')
	return b.String()
}

func detailed(r tomltest.Runner, t tomltest.Test) string {
	b := new(strings.Builder)
	b.WriteString(short(r, t))

	if t.Failed() {
		b.WriteString(indentWith(
			indent(t.Failure, 4),
			zli.Colorize(" ", hlErr)))
		b.WriteByte('\n')
	}
	showStream(b, "input sent to parser-cmd", t.Input)

	var j map[string]any
	err := json.Unmarshal([]byte(t.Output), &j)
	if err == nil {
		out, err := json.MarshalIndent(j, "", "  ")
		if err == nil {
			t.Output = string(out)
		}
	}

	if t.OutputFromStderr {
		showStream(b, "output from parser-cmd (stderr)", t.Output)
	} else {
		showStream(b, "output from parser-cmd (stdout)", t.Output)
	}
	if t.Type() == tomltest.TypeValid {
		showStream(b, "want", t.Want)
	} else {
		showStream(b, "want", "Exit code 1")
	}
	b.WriteByte('\n')

	return b.String()
}

func showStream(b *strings.Builder, name, s string) {
	b.WriteByte('\n')
	fmt.Fprintln(b, zli.Colorize("     "+name+":", zli.Bold))
	if s == "" {
		fmt.Fprintln(b, "          <empty>")
		return
	}
	fmt.Fprintln(b, indent(s, 7))
}

func indentWith(s, with string) string {
	return with + strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "\n"+with)
}

func indent(s string, n int) string {
	sp := strings.Repeat(" ", n)
	return sp + strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "\n"+sp)
}

func doCat(fsys fs.FS, tomlVersion string, size int, run, skip []string) {
	f, err := fs.ReadFile(fsys, "files-toml-"+tomlVersion)
	zli.F(err)

	var useFiles = make([]string, 0, 8)
outer1:
	for _, line := range strings.Split(string(f), "\n") {
		if strings.HasPrefix(line, "valid/") && strings.HasSuffix(line, ".toml") {
			for _, s := range skip {
				if m, _ := filepath.Match(s, line); m {
					continue outer1
				}
			}
			if len(run) > 0 {
				found := false
				for _, s := range run {
					if m, _ := filepath.Match(s, line); m {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
			useFiles = append(useFiles, line)
		}
	}
	if len(useFiles) == 0 {
		zli.Fatalf("all files excluded")
	}

	gather := make(map[string]map[string]any) /// file -> decoded
	for _, f := range useFiles {
		var t map[string]any
		_, err := toml.DecodeFS(fsys, f, &t)
		zli.F(err)
		gather[f] = t
	}

	var (
		out   = new(bytes.Buffer)
		wrote int
		keys  []string
		i     int
	)
	for k := range gather {
		keys = append(keys, k)
	}
	sort.Strings(keys)
outer2:
	for {
		for _, line := range keys {
			t := gather[line]
			p := line + "-" + strconv.Itoa(i)

			var prefix func(tbl map[string]any) map[string]any
			prefix = func(tbl map[string]any) map[string]any {
				newTbl := make(map[string]any)
				for k, v := range tbl {
					switch vv := v.(type) {
					case map[string]any:
						k = p + "-" + k
						v = prefix(vv)
					}
					newTbl[k] = v
				}
				return newTbl
			}
			t = prefix(t)

			err = toml.NewEncoder(out).Encode(map[string]any{
				p: t,
			})
			zli.F(err)

			fmt.Println(out.String())
			wrote += out.Len() + 1
			if wrote > size*1024 {
				break outer2
			}
			out.Reset()
		}
		i++
	}
}

func doCopy(fsys fs.FS, tomlVersion string, args []string) {
	if len(args) != 1 {
		zli.Fatalf("need exactly one destination directory")
	}

	files := getList(tomltest.Runner{Version: tomlVersion, Files: fsys})

	d := args[0]
	err := os.MkdirAll(d, 0o777)
	zli.F(err)

	for _, f := range files {
		srcfp, err := fsys.Open(f)
		zli.F(err)

		err = os.MkdirAll(filepath.Dir(filepath.Join(d, f)), 0o777)
		zli.F(err)

		dstfp, err := os.Create(filepath.Join(d, f))
		zli.F(err)

		_, err = io.Copy(dstfp, srcfp)
		zli.F(err)

		err = srcfp.Close()
		zli.F(err)

		err = dstfp.Close()
		zli.F(err)
	}

	v, c, t := zli.GetVersion()

	err = os.WriteFile(filepath.Join(d, "version.toml"), []byte(fmt.Sprintf(`
# Update with:
#     rm -r [this-dir]
#     toml-test -copy [this-dir]
src    = 'https://github.com/toml-lang/toml-test'
tag    = '%s'
commit = '%s'
date   = %s
`[1:], v, c, t.Format("2006-01-02"))), 0o0644)
	zli.F(err)
}
