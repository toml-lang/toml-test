package main

import (
	"strings"
	"testing"
)

func TestUsage(t *testing.T) {
	for k, v := range helpTopics {
		if strings.Contains(v, "\t") {
			t.Errorf("usage for %q contains tabs", k)
		}
		for i, line := range strings.Split(v, "\n") {
			if l := len([]rune(line)); l > 79 {
				t.Errorf("line %d for %q longer than 79 cols (%d): %q", i, k, l, line)
			}
		}
	}

}
