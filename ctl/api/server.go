// Package api the api service
package api

import (
	"net/http"
	"time"

	"github.com/shiroyk/cloudcat/ctl/api/v1"
)

const (
	// DefaultTimeout the default timeout
	DefaultTimeout = time.Minute
	// DefaultAddress the api default address
	DefaultAddress = "localhost:8080"
)

// Options the api server configuration
type Options struct {
	Token      string        `yaml:"token"`
	Address    string        `yaml:"address"`
	Timeout    time.Duration `yaml:"timeout"`
	RequestLog bool          `yaml:"request-log"`
}

// Server the api service
func Server(opt Options) *http.Server {
	return &http.Server{
		Addr:              opt.Address,
		Handler:           v1.Routes(opt.Token, opt.Timeout, opt.RequestLog),
		ReadHeaderTimeout: 10 * time.Second,
	}
}
