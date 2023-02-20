package cmd

import (
	"github.com/shiroyk/cloudcat/cache"
	"github.com/shiroyk/cloudcat/cache/bolt"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/js"
	"github.com/shiroyk/cloudcat/lib"
	"github.com/shiroyk/cloudcat/lib/logger"
)

var configPath = ""

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "~/.config/cloudcat/config.yml", "config file path")
}

func initConfig() {
	config, err := lib.ReadConfig(configPath)
	if err != nil {
		logger.Error("error reading config file", err)
	}
	err = initDependencies(*config)
	if err != nil {
		logger.Error("error initializing dependencies", err)
	}
}

func initDependencies(config lib.Config) error {
	di.Provide(fetch.NewFetcher(config.Fetch), false)
	di.Provide(fetch.DefaultTemplateFuncMap(), false)

	di.ProvideLazy(func() (cache.Cache, error) {
		return bolt.NewCache(config.Cache)
	}, false)
	di.ProvideLazy(func() (cache.Cookie, error) {
		return bolt.NewCookie(config.Cache)
	}, false)
	di.ProvideLazy(func() (cache.Shortener, error) {
		return bolt.NewShortener(config.Cache)
	}, false)

	js.SetScheduler(js.NewScheduler(config.JS))

	return nil
}
