package main

import (
	"os"

	"github.com/shiroyk/cloudcat/plugin/jsmodule"
)

type Module struct{}

func (m Module) Exports() any { return new(Env) }

func init() {
	jsmodule.Register("env", new(Module))
}

type Env struct{}

func (e Env) Get(key string) string { return os.Getenv(key) }
