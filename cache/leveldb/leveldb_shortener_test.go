package leveldb

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestShortener(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "shortener")
	defer os.RemoveAll(tempDir)

	s, err := NewShortener(filepath.Join(tempDir, "Db"))
	if err != nil {
		t.Fatalf("New leveldb: %v", err)
	}

	req := `POST http://localhost
Content-Type: application/json

{"key":"foo"}`

	id := s.Set(req)

	h, ok := s.Get(id)
	if !ok {
		t.Fatal(fmt.Sprintf("not found: %v", id))
	}

	if req != h {
		t.Fatalf("want: %s, got %s", req, h)
	}
}
