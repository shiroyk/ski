package cache

import (
	"testing"
	"time"

	"github.com/shiroyk/cloudcat/cache/memory"
	"github.com/stretchr/testify/assert"
)

func TestShortener(t *testing.T) {
	t.Parallel()
	shortener := memory.NewShortener()

	req := `POST http://localhost
Content-Type: application/json

{"key":"foo"}`

	id := shortener.Set(req, time.Second)

	h, ok := shortener.Get(id)
	if !ok {
		t.Fatalf("not found: %v", id)
	}

	assert.Equal(t, h, req)
}
