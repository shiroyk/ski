package buffer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadAll(t *testing.T) {
	repeat := strings.Repeat("1234567890", 100)
	all, err := ReadAll(strings.NewReader(repeat))
	require.NoError(t, err)
	assert.Equal(t, string(all), repeat)
}
