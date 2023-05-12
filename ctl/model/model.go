package model

import (
	"time"

	"github.com/shiroyk/cloudcat/core"
)

// Source the Model source
type Source struct {
	Name    string        `yaml:"name"`
	HTTP    string        `yaml:"http"`
	Proxy   []string      `yaml:"proxy"`
	Timeout time.Duration `yaml:"timeout"`
}

// Model the model
type Model struct {
	Source *Source          `yaml:"source"`
	Schema *cloudcat.Schema `yaml:"schema"`
}
