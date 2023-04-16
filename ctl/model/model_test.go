package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestSourceYaml(t *testing.T) {
	t.Parallel()
	s := `source:
  name: test
  http: |
    http://localhost
    user-agent: cloudcat
  timeout: 60s
`
	model := new(Model)
	err := yaml.Unmarshal([]byte(s), model)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, Source{
		Name:    "test",
		HTTP:    "http://localhost\nuser-agent: cloudcat\n",
		Timeout: time.Minute,
	}, *model.Source)
}
