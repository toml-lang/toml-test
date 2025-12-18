package tomltest

import (
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func forVersion(t *testing.T, f func(string, *testing.T)) {
	os.Unsetenv("BURNTSUSHI_TOML_110") // Just to be sure.
	t.Run("toml 1.0", func(t *testing.T) { f("1.0.0", t) })

	os.Setenv("BURNTSUSHI_TOML_110", "")
	defer func() { os.Unsetenv("BURNTSUSHI_TOML_110") }()
	t.Run("toml 1.1", func(t *testing.T) { f("1.1.0", t) })
}

func TestCompareTOML(t *testing.T) {
	forVersion(t, func(v string, t *testing.T) {
		t.Run("self", func(t *testing.T) {
			files, err := Runner{Version: v, Files: TestCases()}.List()
			if err != nil {
				t.Fatal(err)
			}

			for _, f := range files {
				if !strings.HasPrefix(f, "valid/") {
					continue
				}
				t.Run(f, func(t *testing.T) {
					d, err := os.ReadFile("tests/" + f + ".toml")
					if err != nil {
						t.Fatal(err)
					}

					var wantT, haveT any
					_, err = toml.Decode(string(d), &wantT)
					if err != nil {
						t.Fatal(err)
					}
					_, err = toml.Decode(string(d), &haveT)
					if err != nil {
						t.Fatal(err)
					}

					t.Run("identical", func(t *testing.T) {
						r := Test{}
						r = r.CompareTOML(wantT, haveT)
						if r.Failure != "" {
							t.Fatal(r.Failure)
						}
					})
					t.Run("differ", func(t *testing.T) {
						var haveT any
						_, err = toml.Decode(string(d)+"\n\nnewkey123=1\n", &haveT)
						if err != nil {
							t.Fatal(err)
						}

						r := Test{}
						r = r.CompareTOML(wantT, haveT)
						if r.Failure == "" {
							t.Fatal("wanted failure")
						}
					})
				})
			}
		})
	})

	tests := []struct {
		a, b string
		eq   bool
	}{
		{``, ``, true},
		{`a=[{}]`, `[[a]]`, true},
		{`a=[{k=1}]`, "[[a]]\nk=1", true},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			var wantT, haveT any
			_, err := toml.Decode(tt.a, &wantT)
			if err != nil {
				t.Fatal(err)
			}
			_, err = toml.Decode(tt.b, &haveT)
			if err != nil {
				t.Fatal(err)
			}

			{
				r := Test{}
				r = r.CompareTOML(wantT, haveT)
				if tt.eq && r.Failure != "" {
					t.Fatal(r.Failure)
				}
				if !tt.eq && r.Failure == "" {
					t.Fatal("wanted failure")
				}
			}
			{ // Want + have reversed
				r := Test{}
				r = r.CompareTOML(haveT, wantT)
				if tt.eq && r.Failure != "" {
					t.Fatal(r.Failure)
				}
				if !tt.eq && r.Failure == "" {
					t.Fatal("wanted failure")
				}
			}
		})
	}
}

func TestSize(t *testing.T) {
	err := fs.WalkDir(TestCases(), "valid", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		inf, err := d.Info()
		if err != nil {
			return err
		}
		if inf.IsDir() {
			return nil
		}
		if strings.Contains(path, "/spec-") || strings.Contains(path, "/spec/") {
			return nil
		}
		if path == "valid/comment/tricky.json" { // Exception
			return nil
		}

		if inf.Size() > 2048 {
			t.Errorf("larger than 2K: %s (%fK)", path, float64(inf.Size())/1024)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
