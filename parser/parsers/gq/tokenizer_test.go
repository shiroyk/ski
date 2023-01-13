package gq

import "testing"

func TestBuildInFunc(t *testing.T) {
	rules := []string{
		`-> -> unknown`, `-> text(`, `-> text(")`,
		`-> text("')`, `-> text('")`, `-> text(' ", ")`,
		`-> text("\")`, `-> text('\')`, `-> text(" ", ')`,
	}

	for _, rule := range rules {
		if _, err := gq.GetString(ctx, content, rule); err == nil {
			t.Fatalf("Unexpected function and argument parse %s", rule)
		}
	}
}
