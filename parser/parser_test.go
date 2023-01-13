package parser

import (
	"fmt"
	"testing"
)

func TestManager(t *testing.T) {
	Each(func(key string, parser Parser) {
		fmt.Println(key)
	})
}
