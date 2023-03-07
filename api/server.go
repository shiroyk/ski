package api

import (
	"net/http"
	"time"

	v1 "github.com/shiroyk/cloudcat/api/v1"
	"github.com/shiroyk/cloudcat/lib/utils"
)

const (
	// DefaultTimeout the default timeout
	DefaultTimeout = time.Minute
	// DefaultAddress the api default address
	DefaultAddress = "localhost:8080"
)

// Options the api server configuration
type Options struct {
	Token   string        `yaml:"token"`
	Address string        `yaml:"address"`
	Timeout time.Duration `yaml:"timeout"`
}

// Server the api service
func Server(opt Options) *http.Server {
	return &http.Server{
		Addr:              opt.Address,
		Handler:           v1.RouteRun(opt.Token),
		ReadHeaderTimeout: utils.ZeroOr(opt.Timeout, DefaultTimeout),
		WriteTimeout:      utils.ZeroOr(opt.Timeout, DefaultTimeout),
	}
}
