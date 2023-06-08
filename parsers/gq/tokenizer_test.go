package gq

import (
	"testing"
)

func TestParseRuleFunction(t *testing.T) {
	t.Parallel()
	rules := []string{
		`-> -> unknown`, `-> text(`, `-> text(")`,
		`-> text("')`, `-> text('")`, `-> text(' ", ")`,
		`-> text("\")`, `-> text('\')`, `-> text(" ", ')`,
	}
	funcs := builtins()
	for _, rule := range rules {
		if _, _, err := parseRuleFunctions(funcs, rule); err == nil {
			t.Fatalf("Unexpected function and argument parse %s", rule)
		}
	}
}
