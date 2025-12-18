package main

import (
	"fmt"
	"sort"
	"strings"

	tomltest "github.com/toml-lang/toml-test/v2"
	"zgo.at/zli"
)

func cmdList(f zli.Flags) {
	var (
		tomlVersion = f.String(tomltest.DefaultVersion, "toml")
		asJSON      = f.Bool(false, "json")
	)
	zli.F(f.Parse())

	l := getList(tomltest.NewRunner(tomltest.Runner{Version: tomlVersion.String()}))
	if asJSON.Bool() {
		newEnc().Encode(l)
	} else {
		for _, ll := range l {
			fmt.Println(ll)
		}
	}
}

func getList(r tomltest.Runner) []string {
	l, err := r.List()
	zli.F(err)

	sort.Strings(l)
	n := make([]string, 0, len(l)*2)
	for _, ll := range l {
		if strings.HasPrefix(ll, "encoder/") {
			continue
		}

		if strings.HasPrefix(ll, "valid/") {
			n = append(n, ll+".json")
		}
		n = append(n, ll+".toml")
	}
	return n
}
