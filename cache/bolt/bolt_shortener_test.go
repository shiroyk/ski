package bolt

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestShortener(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "shortener")
	os.MkdirAll(tempDir, os.ModePerm)
	defer os.RemoveAll(tempDir)

	s, err := NewShortener(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	req := `POST http://localhost
Content-Type: application/json

{"key":"foo"}`

	id := s.Set(req, time.Second)

	h, ok := s.Get(id)
	if !ok {
		t.Fatal(fmt.Sprintf("not found: %v", id))
	}

	assert.Equal(t, h, req)

	time.Sleep(2 * time.Second)

	_, ok = s.Get(id)
	if ok {
		t.Fatal(fmt.Sprintf("not expired: %v", id))
	}

}
