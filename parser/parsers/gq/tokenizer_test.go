package gq

import (
	"testing"
)

func TestParseRuleFunction(t *testing.T) {
	rules := []string{
		`-> -> unknown`, `-> text(`, `-> text(")`,
		`-> text("')`, `-> text('")`, `-> text(' ", ")`,
		`-> text("\")`, `-> text('\')`, `-> text(" ", ')`,
	}

	for _, rule := range rules {
		if _, _, err := parseRuleFunctions(rule); err == nil {
			t.Fatalf("Unexpected function and argument parse %s", rule)
		}
	}
}
