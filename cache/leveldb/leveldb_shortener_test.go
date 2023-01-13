package leveldb

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestShortener(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "shortener")
	defer os.RemoveAll(tempDir)

	s, err := NewShortener(filepath.Join(tempDir, "Db"))
	if err != nil {
		t.Fatalf("New leveldb: %v", err)
	}

	url := "http://localhost"
	headers := map[string]string{
		"Referer": "http://localhost",
	}
	id := s.Set(url, headers)

	u, h, ok := s.Get(id)
	if !ok {
		t.Fatal(fmt.Sprintf("not found: %v", id))
	}

	if u != url {
		t.Fatalf("unexpected url: %s", u)
	}

	if !reflect.DeepEqual(headers, h) {
		t.Fatalf("unexpected headers: %v", headers)
	}
}
