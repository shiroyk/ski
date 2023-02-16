package lib

import "testing"

func TestReadConfig(t *testing.T) {
	_, err := ReadConfig("~/.config/cloudcat/config.yml")
	if err != nil {
		t.Fatal(err)
	}
}
