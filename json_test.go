package tomltest

import (
	"testing"
)

func TestCompareDatetime(t *testing.T) {
	t.Skip() // TODO

	tests := []struct {
		kind, want, have string
		wantFail         bool
	}{
		{"datetime", "2006-01-02T15:04:05.123Z", "2006-01-02T15:04:05.123Z", false},
		{"datetime", "2006-01-02T15:04:05.123Z", "2006-01-02T15:04:05", true},
		{"time-local", "15:04:05.123", "15:04:05.123", false},
		{"time-local", "15:04:05.123", "15:04:05", true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			r := Test{}
			r = r.cmpAsDatetimes(tt.kind, tt.want, tt.have)
			if tt.wantFail && r.Failure == "" {
				t.Fatal("wanted fail, but no failure")
			}
			if !tt.wantFail && r.Failure != "" {
				t.Fatalf("unexpected failure:\n%s", r.Failure)
			}
		})
	}
}

func TestCompareNaN(t *testing.T) {
	a := map[string]any{
		"nan": map[string]any{
			"type":  "float",
			"value": "nan",
		},
	}
	b := map[string]any{
		"nan": map[string]any{
			"type":  "float",
			"value": "+nan",
		},
	}
	c := map[string]any{
		"nan": map[string]any{
			"type":  "float",
			"value": "-nan",
		},
	}

	{
		r := Test{}
		r = r.CompareJSON(a, b)
		if r.Failure != "" {
			t.Fatal(r.Failure)
		}
	}

	{
		r := Test{}
		r = r.CompareJSON(b, c)
		if r.Failure != "" {
			t.Fatal(r.Failure)
		}
	}
}
