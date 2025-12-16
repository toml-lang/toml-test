package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tomltest "github.com/toml-lang/toml-test/v2"
	"zgo.at/zli"
)

func cmdCopy(f zli.Flags) {
	var (
		tomlVersion = f.String(tomltest.DefaultVersion, "toml")
	)
	zli.F(f.Parse())
	if len(f.Args) != 1 {
		zli.Fatalf("need exactly one destination directory")
	}

	files := getList(tomltest.NewRunner(tomltest.Runner{Version: tomlVersion.String()}))
	files = append(files, ".gitattributes")

	d := f.Args[0]
	err := os.MkdirAll(d, 0o777)
	zli.F(err)

	fsys := tomltest.TestCases()
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
