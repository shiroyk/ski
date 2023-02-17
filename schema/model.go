package schema

import (
	"time"
)

// Source the Model source
type Source struct {
	Name    string            `yaml:"name"`
	URL     string            `yaml:"url"`
	Proxy   []string          `yaml:"proxy"`
	Timeout time.Duration     `yaml:"timeout"`
	Header  map[string]string `yaml:"header"`
}

// Model the model
type Model struct {
	Source *Source `yaml:"source"`
	Schema *Schema `yaml:"schema"`
}
