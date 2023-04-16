package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadConfig(t *testing.T) {
	t.Parallel()
	_, err := ReadConfig("~/.config/cloudcat/config.yml")
	assert.NoError(t, err)
}
