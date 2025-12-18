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
		if helpFlag.Set() {
			if contains(f.Args, "test") {
				fmt.Print(usageTest)
			} else {
				fmt.Print(usage)
			}
			return
		}
		zli.F(err)
	}

	switch cmd {
	case "help":
		if contains(f.Args, "test") {
			fmt.Print(usageTest)
		} else {
			fmt.Print(usage)
		}
	case "version":
		v := f.Bool(false, "v")
		zli.F(f.Parse())
		zli.PrintVersion(v.Bool())
	case "copy", "cp":
		cmdCopy(f)
	case "test":
		if helpFlag.Set() || contains(f.Args, "help") {
			fmt.Print(usageTest)
			return
		}
		cmdTest(f)
	}
}

func contains[S ~[]E, E comparable](s S, v E) bool {
	for i := range s {
		if v == s[i] {
			return true
		}
	}
	return false
}
