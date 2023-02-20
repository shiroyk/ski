package cmd

import (
	"github.com/shiroyk/cloudcat/cache/bolt"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/internal/di"
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
	di.Provide(fetch.NewFetcher(config.Fetch))
	di.Provide(fetch.DefaultTemplateFuncMap())
	cache, err := bolt.NewCache(config.Cache)
	if err != nil {
		return err
	}
	di.Provide(cache)
	cookie, err := bolt.NewCookie(config.Cache)
	if err != nil {
		return err
	}
	di.Provide(cookie)
	shortener, err := bolt.NewShortener(config.Cache)
	if err != nil {
		return err
	}
	di.Provide(shortener)

	js.SetScheduler(js.NewScheduler(config.JS))

	return nil
}
