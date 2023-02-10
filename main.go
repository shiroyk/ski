package main

import (
	"fmt"
	"strings"

	_ "github.com/shiroyk/cloudcat/analyzer"
	"github.com/shiroyk/cloudcat/ext"
	_ "github.com/shiroyk/cloudcat/js"
	_ "github.com/shiroyk/cloudcat/parser/parsers/gq"
	_ "github.com/shiroyk/cloudcat/parser/parsers/js"
	_ "github.com/shiroyk/cloudcat/parser/parsers/json"
	_ "github.com/shiroyk/cloudcat/parser/parsers/regex"
	_ "github.com/shiroyk/cloudcat/parser/parsers/xpath"
)

func main() {
	sb := new(strings.Builder)
	for _, e := range ext.GetAll() {
		sb.WriteString(e.String())
		sb.WriteByte('\n')
	}
	fmt.Println(sb.String())
}
