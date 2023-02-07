package parser

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestActions(t *testing.T) {
	t.Parallel()
	var actions Actions
	var err error
	if _, ok := GetParser("act"); !ok {
		Register("act", new(testParser))
	}
	actions = []Action{NewAction(NewStep("act", "1"), NewStep("act", "2")), NewActionOp(OperatorAnd), NewAction(NewStep("act", "3"))}
	ctx := NewContext(Options{Config: Config{Timeout: time.Second}})

	_, err = actions.GetString(ctx, "action")
	if err != nil {
		t.Error(err)
	}

	_, err = actions.GetStrings(ctx, "action")
	if err != nil {
		t.Error(err)
	}

	_, err = actions.GetElement(ctx, "action")
	if err != nil {
		t.Error(err)
	}

	_, err = actions.GetElements(ctx, "action")
	if err != nil {
		t.Error(err)
	}

	bytes, err := yaml.Marshal(actions)
	if err != nil {
		t.Error(err)
	}

	y := `- - act: "1"
  - act: "2"
- and
- act: "3"
`
	if string(bytes) != y {
		t.Errorf("want %q, got %q", y, string(bytes))
	}
}
