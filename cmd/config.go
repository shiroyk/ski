package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/shiroyk/cloudcat/cache"
	"github.com/shiroyk/cloudcat/cache/bolt"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/js"
	"github.com/shiroyk/cloudcat/lib/config"
	"github.com/shiroyk/cloudcat/lib/logger"
	"github.com/shiroyk/cloudcat/lib/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	configArg    string
	configGenArg string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "cloudcat configuration",
	RunE: func(_ *cobra.Command, _ []string) error {
		if configGenArg != "" {
			return writeDiskConfig()
		}
		return nil
	},
}

func writeDiskConfig() error {
	file, err := utils.ExpandPath(configGenArg)
	if err != nil {
		return err
	}
	path := filepath.Dir(file)
	if _, err = os.Stat(file); errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
		cfg := config.DefaultConfig()
		bytes, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		err = os.WriteFile(file, bytes, 0o600)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("configuration file is already exists")
}

func init() {
	configCmd.Flags().StringVarP(&configGenArg, "gen", "g", "", "generate default configuration file")
	rootCmd.PersistentFlags().StringVar(&configArg, "config", "~/.config/cloudcat/config.yml", "config file path")
	rootCmd.AddCommand(configCmd)
}

func initConfig() {
	cfg, err := config.ReadConfig(configArg)
	if err != nil {
		logger.Error("error reading config file", err)
	}
	initDependencies(cfg)
	rootCmd.SetContext(config.NewContext(context.Background(), cfg))
}

func initDependencies(config config.Config) {
	di.Provide(fetch.NewFetcher(config.Fetch))
	di.Provide(fetch.DefaultTemplateFuncMap())

	di.ProvideLazy(func() (cache.Cache, error) {
		return bolt.NewCache(config.Cache)
	})
	di.ProvideLazy(func() (cache.Cookie, error) {
		return bolt.NewCookie(config.Cache)
	})

	js.SetScheduler(js.NewScheduler(config.JS))
}
