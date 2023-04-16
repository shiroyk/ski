package main

import (
	"github.com/shiroyk/cloudcat/ctl/cmd"
	_ "github.com/shiroyk/cloudcat/jsmodules"
	_ "github.com/shiroyk/cloudcat/parsers"
)

func main() {
	cmd.Execute()
}
