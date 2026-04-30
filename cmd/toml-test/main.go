package main

import (
	"errors"
	"fmt"
	"os"

	"zgo.at/zli"
)

func main() {
	f := zli.NewFlags(os.Args)
	helpFlag := f.Bool(false, "h", "help")
	zli.F(f.Parse(zli.AllowUnknown()))
	cmd, err := f.ShiftCommand("help", "version", "test", "list", "ls", "copy", "cp")
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
	case "list", "ls":
		cmdList(f)
	case "copy", "cp":
		cmdCopy(f)
	case "test":
		cmdTest(f)
	}
}
