// Package config the configuration
package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"github.com/shiroyk/cloudcat/cache"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/js"
	"github.com/shiroyk/cloudcat/lib/utils"
)

type configKey struct{}

// NewContext returns a context that contains the given Config.
func NewContext(ctx context.Context, config Config) context.Context {
	return context.WithValue(ctx, configKey{}, config)
}

// FromContext returns the Config stored in ctx by NewContext, or the default
// Config if there is none.
func FromContext(ctx context.Context) Config {
	if config, ok := ctx.Value(configKey{}).(Config); ok {
		return config
	}
	return DefaultConfig()
}

// Config The cloudcat configuration
type Config struct {
	// Cache
	Cache cache.Options `yaml:"cache"`

	// Fetch
	Fetch fetch.Options `yaml:"fetch"`

	// JS
	JS js.Options `yaml:"js"`
}

// DefaultConfig The default configuration
func DefaultConfig() Config {
	return Config{
		Cache: cache.Options{
			Path: cache.DefaultPath,
		},
		Fetch: fetch.Options{
			MaxBodySize:    fetch.DefaultMaxBodySize,
			RetryTimes:     fetch.DefaultRetryTimes,
			RetryHTTPCodes: fetch.DefaultRetryHTTPCodes,
			Timeout:        fetch.DefaultTimeout,
			CachePolicy:    cache.RFC2616,
		},
		JS: js.Options{
			InitialVMs:         2,
			MaxVMs:             runtime.GOMAXPROCS(0),
			MaxRetriesGetVM:    js.DefaultMaxRetriesGetVM,
			MaxTimeToWaitGetVM: js.DefaultMaxTimeToWaitGetVM,
		},
	}
}

// ReadConfig read configuration from the file.
// If the configuration file is not existing then create it with default configuration.
func ReadConfig(path string) (config Config, err error) {
	file, err := utils.ExpandPath(path)
	if err != nil {
		return config, err
	}
	path = filepath.Dir(file)
	if _, err = os.Stat(file); errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return config, err
		}
		return DefaultConfig(), nil
	}

	return utils.ReadYaml[Config](file)
}
