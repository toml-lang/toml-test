package main

import (
	"strings"
	"testing"
)

func TestUsage(t *testing.T) {
	if strings.Contains(usage, "\t") {
		t.Error("usage contains tabs")
	}
	for i, line := range strings.Split(usage, "\n") {
		if l := len(line); l > 79 {
			t.Errorf("line %d longer than 79 cols: %d", i, l)
		}
	}
}
