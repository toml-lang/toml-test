package main

import (
	"bytes"
	_ "embed"
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
	"text/template"
	"time"

	"github.com/BurntSushi/toml"
	tomltest "github.com/toml-lang/toml-test"
	"zgo.at/jfmt"
	"zgo.at/zli"
)

var hlErr = zli.Color256(224).Bg() | zli.Color256(0) | zli.Bold

//go:embed script.gotxt
var script []byte

var scriptTemplate = template.Must(template.New("").Option("missingkey=error").Parse(string(script)))

func parseFlags() (tomltest.Runner, []string, int, bool, bool, bool, bool) {
	f := zli.NewFlags(os.Args)
	var (
		help          = f.Bool(false, "help", "h")
		versionFlag   = f.IntCounter(0, "version", "V")
		tomlVersion   = f.String(tomltest.DefaultVersion, "toml")
		encoder       = f.Bool(false, "encoder")
		showAll       = f.IntCounter(0, "v")
		color         = f.String("always", "color")
		skip          = f.StringList(nil, "skip")
		run           = f.StringList(nil, "run")
		listFiles     = f.Bool(false, "list-files")
		cat           = f.Int(0, "cat")
		copyFiles     = f.Bool(false, "copy")
		parallel      = f.Int(runtime.NumCPU(), "parallel")
		script        = f.Bool(false, "script", "print-skip") // -print-skip is the old name
		intAsFloat    = f.Bool(false, "int-as-float")
		errors        = f.String("", "errors")
		timeout       = f.String("1s", "timeout")
		noNumber      = f.Bool(false, "no-number", "no_number")
		skipMustError = f.Bool(false, "skip-must-err", "skip-must-error")
		asJSON        = f.Bool(false, "json")
	)
	zli.F(f.Parse())
	if help.Bool() {
		fmt.Printf(usage, filepath.Base(os.Args[0]))
		zli.Exit(0)
	}
	if script.Bool() && encoder.Bool() {
		zli.Fatalf("cannot use -script and -encoder; generate a script without -encoder and fill in the encoder binary")
	}
	if script.Bool() && asJSON.Bool() {
		zli.Fatalf("-script does not support -json")
	}
	if tomlVersion.String() == "latest" {
		*tomlVersion.Pointer() = tomltest.DefaultVersion
	}
	if versionFlag.Int() > 0 {
		zli.PrintVersion(versionFlag.Int() > 1)
		zli.Exit(0)
	}
	if cat.Set() {
		doCat(tomlVersion.String(), cat.Int(), run.Strings(), skip.Strings())
		zli.Exit(0)
	}
	if copyFiles.Set() {
		doCopy(tomlVersion.String(), f.Args)
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
		Encoder:       encoder.Bool(),
		RunTests:      run.StringsSplit(","),
		SkipTests:     skip.StringsSplit(","),
		Version:       tomlVersion.String(),
		Parallel:      parallel.Int(),
		Files:         tomltest.TestCases(),
		Parser:        tomltest.NewCommandParser(f.Args),
		Timeout:       dur,
		IntAsFloat:    intAsFloat.Bool(),
		SkipMustError: skipMustError.Bool(),
		Errors:        errs,
	}
	if intAsFloat.Bool() {
		r.SkipTests = append(r.SkipTests, "valid/integer/long")
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

	return r, f.Args, showAll.Int(), listFiles.Bool(),
		script.Bool(), noNumber.Bool(), asJSON.Bool()
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

func newEnc() *json.Encoder {
	j := json.NewEncoder(os.Stdout)
	j.SetEscapeHTML(false)
	j.SetIndent("", "    ")
	return j
}

func main() {
	runner, cmd, showAll, listFiles, script, noNumber, asJSON := parseFlags()

	if listFiles {
		l := getList(runner)
		if asJSON {
			newEnc().Encode(l)
		} else {
			for _, ll := range l {
				fmt.Println(ll)
			}
		}
		return
	}

	tests, err := runner.Run()
	zli.F(err)

	if script {
		var failedValid, failedInvalid []string
		for _, f := range tests.Tests {
			if f.Failed() {
				if f.Type() == tomltest.TypeValid {
					failedValid = append(failedValid, f.Path)
				} else {
					failedInvalid = append(failedInvalid, f.Path)
				}
			}
		}
		err := scriptTemplate.Execute(os.Stdout, struct {
			Decoder       string
			TOML          string
			FailedValid   []string
			FailedInvalid []string
		}{strings.Join(cmd, " "), runner.Version, failedValid, failedInvalid})
		zli.F(err)
		return
	}

	if asJSON {
		printJSON(runner, tests, cmd, showAll)
	} else {
		printText(runner, tests, cmd, showAll, noNumber)
	}

	if tests.FailedValid > 0 || tests.FailedEncoder > 0 || tests.FailedInvalid > 0 {
		zli.Exit(1)
	}
	zli.Exit(0)
}

func printJSON(runner tomltest.Runner, tests tomltest.Tests, cmd []string, showAll int) {
	_, _, date := zli.GetVersion()

	out := struct {
		Version       string          `json:"version"`
		TOML          string          `json:"toml"`
		Flags         []string        `json:"flags"`
		Parser        []string        `json:"parser"`
		PassedValid   int             `json:"passed_valid"`
		PassedEncoder int             `json:"passed_encoder"`
		PassedInvalid int             `json:"passed_invalid"`
		FailedValid   int             `json:"failed_valid"`
		FailedEncoder int             `json:"failed_encoder"`
		FailedInvalid int             `json:"failed_invalid"`
		Skipped       int             `json:"skipped"`
		Tests         []tomltest.Test `json:"tests"`
	}{
		fmt.Sprintf("toml-test v%s", date.Format("2006-01-02")),
		runner.Version, os.Args, cmd,
		tests.PassedValid, tests.PassedEncoder, tests.PassedInvalid,
		tests.FailedValid, tests.FailedEncoder, tests.FailedInvalid,
		tests.Skipped, []tomltest.Test{},
	}
	for _, t := range tests.Tests {
		if t.Failed() || showAll >= 1 {
			out.Tests = append(out.Tests, t)
		}
	}
	newEnc().Encode(out)
}

func printText(runner tomltest.Runner, tests tomltest.Tests, cmd []string, showAll int, noNumber bool) {
	for _, t := range tests.Tests {
		if t.Failed() || showAll > 1 {
			fmt.Print(detailed(runner, t, noNumber))
		} else if showAll == 1 {
			fmt.Print(short(runner, t))
		}
	}

	_, _, date := zli.GetVersion()
	fmt.Printf("toml-test v%s %s: ", date.Format("2006-01-02"), cmd)
	if tests.Skipped > 0 {
		fmt.Printf(", %2d skipped", tests.Skipped)
	}

	fmt.Println()
	if runner.Encoder {
		fmt.Printf("encoder tests: %3d passed, %2d failed\n", tests.PassedEncoder, tests.FailedEncoder)
	} else {
		fmt.Printf("  valid tests: %3d passed, %2d failed\n", tests.PassedValid, tests.FailedValid)
		fmt.Printf("invalid tests: %3d passed, %2d failed\n", tests.PassedInvalid, tests.FailedInvalid)
	}
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

func detailed(r tomltest.Runner, t tomltest.Test, noNumber bool) string {
	b := new(strings.Builder)
	b.WriteString(short(r, t))

	if t.Failed() {
		b.WriteString(indentWith(
			indent(t.Failure, 4, false),
			zli.Colorize(" ", hlErr)))
		b.WriteByte('\n')
	}
	showStream(b, fmt.Sprintf("input sent to parser-cmd (PID %d)", t.PID), t.Input, noNumber)

	out, err := jfmt.NewFormatter(0, "", "  ").FormatString(t.Output)
	if err == nil {
		t.Output = out
	}

	if t.OutputFromStderr {
		showStream(b, fmt.Sprintf("output from parser-cmd (PID %d) (stderr)", t.PID), t.Output, noNumber)
	} else {
		showStream(b, fmt.Sprintf("output from parser-cmd (PID %d) (stdout)", t.PID), t.Output, noNumber)
	}
	if t.Type() == tomltest.TypeValid {
		showStream(b, "want", t.Want, noNumber)
	} else {
		showStream(b, "want", "Exit code 1", noNumber)
	}
	b.WriteByte('\n')

	return b.String()
}

func showStream(b *strings.Builder, name, s string, noNumber bool) {
	b.WriteByte('\n')
	fmt.Fprintln(b, zli.Colorize("     "+name+":", zli.Bold))
	if s == "" {
		fmt.Fprintln(b, "          <empty>")
		return
	}
	fmt.Fprintln(b, indent(s, 7, !noNumber && s != "Exit code 1"))
}

func indentWith(s, with string) string {
	return with + strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "\n"+with)
}

func indent(s string, n int, number bool) string {
	sp := strings.Repeat(" ", n)
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
		if number {
			if lines[i] != "" { // No trailing space for empty lines.
				lines[i] = " " + lines[i]
			}
			lines[i] = fmt.Sprintf("%s%s%2d │\x1b[0m%s", zli.Color256(244), sp, i+1, lines[i])
		} else {
			if lines[i] != "" {
				lines[i] = fmt.Sprintf("%s%s", sp, lines[i])
			}
		}
	}
	return strings.Join(lines, "\n")
}

func doCat(tomlVersion string, size int, run, skip []string) {
	fsys := tomltest.TestCases()
	f, err := fs.ReadFile(fsys, "files-toml-"+tomlVersion)
	zli.F(err)

	useFiles := make([]string, 0, 8)
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

func doCopy(tomlVersion string, args []string) {
	fsys := tomltest.TestCases()
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
