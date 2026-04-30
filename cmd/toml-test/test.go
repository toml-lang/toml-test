package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/BurntSushi/toml"
	tomltest "github.com/toml-lang/toml-test/v2"
	"zgo.at/jfmt"
	"zgo.at/zli"
)

var hlErr = zli.Color256(224).Bg() | zli.Color256(0) | zli.Bold

//go:embed script.gotxt
var script []byte

func shellQuote(s string) string {
	if strings.Contains(s, "'") {
		s = strings.ReplaceAll(s, `'`, `'"'"'`) // cmd'foo → 'cmd'"'"'foo'
	}
	return "'" + s + "'"
}

var scriptTemplate = template.Must(template.New("").
	Option("missingkey=error").
	Funcs(template.FuncMap{"join": strings.Join, "quote": shellQuote}).
	Parse(string(script)))

func cmdTest(f zli.Flags) {
	runner, verbose, script, asJSON, setup := parseTestFlags(f)

	for _, s := range setup {
		f := strings.Fields(s)
		if verbose > 0 {
			fmt.Printf("SETUP %v\n", s)
		}
		out, err := exec.Command(f[0], f[1:]...).CombinedOutput()
		if err != nil {
			zli.Fatalf("error running -setup=%q: %s: command output:\n%s", s, err, out)
		}
	}

	tests, err := runner.Run()
	zli.F(err)

	if script {
		var failedValid, failedEncoder, failedInvalid []string
		for _, f := range tests.Tests {
			if f.Failed() {
				if f.Encoder() {
					failedEncoder = append(failedEncoder, "encoder/"+f.Path[6:])
				} else if f.Invalid() {
					failedInvalid = append(failedInvalid, f.Path)
				} else {
					failedValid = append(failedValid, f.Path)
				}
			}
		}
		var enc []string
		if runner.Encoder != nil {
			enc = runner.Encoder.Cmd()
		}
		v, _, _ := strings.Cut(zli.Version(), "/") // "v2.2.0" or "e4e12cad/2026-04-30"
		err := scriptTemplate.Execute(os.Stdout, struct {
			Decoder       []string
			Encoder       []string
			Setup         []string
			TOML          string
			Version       string
			FailedValid   []string
			FailedEncoder []string
			FailedInvalid []string
		}{runner.Decoder.Cmd(), enc, setup, runner.Version, v, failedValid, failedEncoder, failedInvalid})
		zli.F(err)
		return
	}

	if asJSON {
		printJSON(runner, tests, verbose)
	} else {
		printText(runner, tests, verbose)
	}

	if tests.FailedValid > 0 || tests.FailedEncoder > 0 || tests.FailedInvalid > 0 {
		zli.Exit(1)
	}
	zli.Exit(0)
}

func parseTestFlags(f zli.Flags) (tomltest.Runner, int, bool, bool, []string) {
	var (
		decoder       = f.String("", "decoder")
		encoder       = f.String("", "encoder")
		setup         = f.StringList(nil, "setup")
		tomlVersion   = f.String(tomltest.DefaultVersion, "toml")
		verbose       = f.IntCounter(0, "v")
		color         = f.String("always", "color")
		skip          = f.StringList(nil, "skip")
		run           = f.StringList(nil, "run")
		parallel      = f.Int(runtime.NumCPU(), "parallel")
		script        = f.Bool(false, "script")
		intAsFloat    = f.Bool(false, "int-as-float")
		errors        = f.String("", "errors")
		timeout       = f.String("1s", "timeout")
		skipMustError = f.Bool(false, "skip-must-err", "skip-must-error")
		asJSON        = f.Bool(false, "json")
	)
	zli.F(f.Parse())
	if script.Bool() && asJSON.Bool() {
		zli.Fatalf("-script does not support -json")
	}
	if decoder.String() == "" {
		zli.Fatalf("must have -decoder command")
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

	if len(f.Args) > 0 {
		zli.Fatalf("no positional arguments allowed")
	}

	var enc tomltest.Parser
	if encoder.String() != "" {
		enc = tomltest.NewCommandParser(strings.Fields(encoder.String()))
	}

	runner := tomltest.NewRunner(tomltest.Runner{
		Decoder:       tomltest.NewCommandParser(strings.Fields(decoder.String())),
		Encoder:       enc,
		RunTests:      run.StringsSplit(","),
		SkipTests:     skip.StringsSplit(","),
		Version:       tomlVersion.String(),
		Parallel:      parallel.Int(),
		Timeout:       dur,
		IntAsFloat:    intAsFloat.Bool(),
		SkipMustError: skipMustError.Bool(),
		Errors:        errs,
	})
	if intAsFloat.Bool() {
		runner.SkipTests = append(runner.SkipTests, "valid/integer/long")
	}

	// TODO: -run='valid/*' doesn't really work as expected, as it uses filepath
	// glob matching where '*' doesn't match a '/'
	for _, runner := range runner.RunTests {
		_, err := filepath.Match(runner, "")
		if err != nil {
			zli.Fatalf("invalid glob pattern %q in -run: %s", runner, err)
		}
	}
	for _, runner := range runner.SkipTests {
		_, err := filepath.Match(runner, "")
		if err != nil {
			zli.Fatalf("invalid glob pattern %q in -skip: %s", runner, err)
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

	return runner, verbose.Int(), script.Bool(), asJSON.Bool(), setup.Strings()
}

func newEnc() *json.Encoder {
	j := json.NewEncoder(os.Stdout)
	j.SetEscapeHTML(false)
	j.SetIndent("", "    ")
	return j
}

func printJSON(runner tomltest.Runner, tests tomltest.Tests, verbose int) {
	var enc []string
	if runner.Encoder != nil {
		enc = runner.Encoder.Cmd()
	}

	out := struct {
		Version       string          `json:"version"`
		TOML          string          `json:"toml"`
		Flags         []string        `json:"flags"`
		Decoder       []string        `json:"decoder"`
		Encoder       []string        `json:"encoder"`
		PassedValid   int             `json:"passed_valid"`
		PassedEncoder int             `json:"passed_encoder"`
		PassedInvalid int             `json:"passed_invalid"`
		FailedValid   int             `json:"failed_valid"`
		FailedEncoder int             `json:"failed_encoder"`
		FailedInvalid int             `json:"failed_invalid"`
		Skipped       int             `json:"skipped"`
		Tests         []tomltest.Test `json:"tests"`
	}{
		fmt.Sprintf("toml-test %s", zli.Version()),
		runner.Version, os.Args, runner.Decoder.Cmd(), enc,
		tests.PassedValid, tests.PassedEncoder, tests.PassedInvalid,
		tests.FailedValid, tests.FailedEncoder, tests.FailedInvalid,
		tests.Skipped, []tomltest.Test{},
	}
	for _, t := range tests.Tests {
		if t.Failed() || verbose >= 1 {
			out.Tests = append(out.Tests, t)
		}
	}
	newEnc().Encode(out)
}

func printText(runner tomltest.Runner, tests tomltest.Tests, verbose int) {
	for _, t := range tests.Tests {
		if t.Failed() || verbose > 1 {
			fmt.Print(detailed(runner, t))
		} else if verbose == 1 {
			fmt.Print(short(runner, t))
		}
	}

	enc := "[no encoder]"
	if runner.Encoder != nil {
		enc = fmt.Sprintf("%s", runner.Encoder.Cmd())
	}
	fmt.Printf("toml-test %s %s %s\n", zli.Version(), runner.Decoder.Cmd(), enc)
	if tests.Skipped > 0 {
		fmt.Printf("skipped tests: %d\n", tests.Skipped)
	}
	fmt.Printf("  valid tests: %3d passed, %2d failed\n", tests.PassedValid, tests.FailedValid)
	if runner.Encoder == nil {
		fmt.Println("encoder tests: no encoder command given")
	} else {
		fmt.Printf("encoder tests: %3d passed, %2d failed\n", tests.PassedEncoder, tests.FailedEncoder)
	}
	fmt.Printf("invalid tests: %3d passed, %2d failed\n", tests.PassedInvalid, tests.FailedInvalid)
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
		if t.Encoder() {
			b.WriteString(" (encoder)")
		}
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
			indent(t.Failure, 4, false),
			zli.Colorize(" ", hlErr)))
		b.WriteByte('\n')
	}
	showStream(b, fmt.Sprintf("input sent to parser-cmd (PID %d)", t.PID), t.Input)

	out, err := jfmt.NewFormatter(0, "", "  ").FormatString(t.Output)
	if err == nil {
		t.Output = out
	}

	if t.OutputFromStderr {
		showStream(b, fmt.Sprintf("output from parser-cmd (PID %d) (stderr)", t.PID), t.Output)
	} else {
		showStream(b, fmt.Sprintf("output from parser-cmd (PID %d) (stdout)", t.PID), t.Output)
	}
	if t.Invalid() {
		showStream(b, "want", "Exit code 1")
	} else {
		showStream(b, "want", t.Want)
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
	fmt.Fprintln(b, indent(s, 7, s != "Exit code 1"))
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
