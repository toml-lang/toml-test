package main

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/template"

	"zgo.at/zli"
)

var hlErr = zli.Color256(224).Bg() | zli.Color256(0) | zli.Bold

//go:embed script.gotxt
var script []byte

var scriptTemplate = template.Must(template.New("").
	Option("missingkey=error").
	Funcs(template.FuncMap{
		"join": strings.Join,
	}).
	Parse(string(script)))

func main() {
	f := zli.NewFlags(os.Args)
	helpFlag := f.Bool(false, "h", "help")
	zli.F(f.Parse(zli.AllowUnknown()))
	cmd, err := f.ShiftCommand("help", "version", "test", "copy", "cp")
	if errors.Is(err, zli.ErrCommandNoneGiven{}) {
		fmt.Print(usage)
		return
	}
	if err != nil {
		zli.F(err)
	}
	if helpFlag.Set() {
		f.Args, cmd = []string{cmd}, "help"
	}

	switch cmd {
	case "help":
		topic := ""
		if len(f.Args) > 0 {
			topic = f.Args[0]
		}
		u, ok := helpTopics[topic]
		if !ok {
			zli.Fatalf("no help for %q", topic)
		}
		fmt.Print(u)
	case "version":
		v := f.Bool(false, "v")
		zli.F(f.Parse())
		zli.PrintVersion(v.Bool())
	case "copy", "cp":
		cmdCopy(f)
	case "test":
		cmdTest(f)
	}
}
