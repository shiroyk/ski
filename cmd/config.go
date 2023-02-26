package cmd

import (
	"context"

	"github.com/shiroyk/cloudcat/cache"
	"github.com/shiroyk/cloudcat/cache/bolt"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/js"
	"github.com/shiroyk/cloudcat/lib/config"
	"github.com/shiroyk/cloudcat/lib/logger"
)

var configPath = ""

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "~/.config/cloudcat/config.yml", "config file path")
}

func initConfig() {
	cfg, err := config.ReadConfig(configPath)
	if err != nil {
		logger.Error("error reading config file", err)
	}
	err = initDependencies(*cfg)
	if err != nil {
		logger.Error("error initializing dependencies", err)
	}
	rootCmd.SetContext(config.NewContext(context.Background(), *cfg))
}

func initDependencies(config config.Config) error {
	di.Provide(fetch.NewFetcher(config.Fetch), false)
	di.Provide(fetch.DefaultTemplateFuncMap(), false)

	di.ProvideLazy(func() (cache.Cache, error) {
		return bolt.NewCache(config.Cache)
	}, false)
	di.ProvideLazy(func() (cache.Cookie, error) {
		return bolt.NewCookie(config.Cache)
	}, false)

	js.SetScheduler(js.NewScheduler(config.JS))

	return nil
}
