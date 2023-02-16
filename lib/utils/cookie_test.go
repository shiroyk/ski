package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCookie(t *testing.T) {
	maxAge := "MaxAge=3600;"
	assert.Equal(t, "MaxAge=3600", CookieToString(ParseCookie(maxAge)))
}
