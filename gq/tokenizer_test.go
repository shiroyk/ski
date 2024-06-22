package gq

import (
	"testing"
)

func TestParseFuncArguments(t *testing.T) {
	t.Parallel()
	rules := []string{
		`-> text(`, `-> text(")`,
		`-> text("')`, `-> text('")`, `-> text(' ", ")`,
		`-> text("\")`, `-> text('\')`, `-> text(" ", ')`,
	}
	for _, rule := range rules {
		if _, _, err := parseFuncArguments(rule); err == nil {
			t.Fatalf("Unexpected function and argument parse %s", rule)
		}
	}
}
