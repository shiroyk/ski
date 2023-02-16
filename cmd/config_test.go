package cmd

import "testing"

func TestReadConfig(t *testing.T) {
	_, err := readConfig("~/.config/cloudcat/config.yml")
	if err != nil {
		t.Fatal(err)
	}
}
