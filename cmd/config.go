package cmd

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/shiroyk/cloudcat/cache"
	"github.com/shiroyk/cloudcat/cache/bolt"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/js"
	"github.com/shiroyk/cloudcat/utils"
	"gopkg.in/yaml.v3"
)

// Config The configuration
type Config struct {
	// Cache
	Cache cache.Options `yaml:"cache"`

	// Fetch
	Fetch fetch.Options `yaml:"fetch"`

	// JS
	JS js.Options `yaml:"js"`
}

func defaultConfig() *Config {
	return &Config{
		Cache: cache.Options{
			Path: bolt.DefaultPath,
		},
		Fetch: fetch.Options{
			MaxBodySize:    fetch.DefaultMaxBodySize,
			RetryTimes:     fetch.DefaultRetryTimes,
			RetryHTTPCodes: fetch.DefaultRetryHTTPCodes,
			Timeout:        fetch.DefaultTimeout,
		},
		JS: js.Options{
			InitialVMs:         2,
			MaxVMs:             runtime.GOMAXPROCS(0),
			MaxRetriesGetVM:    js.DefaultMaxRetriesGetVM,
			MaxTimeToWaitGetVM: js.DefaultMaxTimeToWaitGetVM,
		},
	}
}

func readConfig(path string) (config *Config, err error) {
	file := path
	if strings.HasPrefix(strings.TrimSpace(path), "~") {
		usr, err := user.Current()
		if err != nil {
			return nil, err
		}
		file = filepath.Join(usr.HomeDir, path[2:])
		path = filepath.Dir(file)
	}
	if _, err = os.Stat(file); errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return nil, err
		}
		config = defaultConfig()
		bytes, err := yaml.Marshal(config)
		if err != nil {
			return nil, err
		}
		err = os.WriteFile(file, bytes, 0644)
		if err != nil {
			return nil, err
		}
		return config, nil
	}

	return utils.ReadYaml[Config](file)
}
