package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shiroyk/cloudcat/core"
	"github.com/shiroyk/cloudcat/core/js"
	"github.com/shiroyk/cloudcat/ctl/cache"
	"github.com/shiroyk/cloudcat/ctl/config"
	"github.com/shiroyk/cloudcat/ctl/utils"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/plugin"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
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
		slog.Error("error reading config file", "error", err)
	}
	initDependencies(cfg)
	rootCmd.SetContext(config.NewContext(context.Background(), cfg))
}

func initDependencies(config config.Config) {
	core.Provide(fetch.NewFetcher(config.Fetch))
	core.Provide(fetch.DefaultTemplateFuncMap())

	if config.Plugin.Path != "" {
		errs := plugin.LoadPlugin(config.Plugin.Path)
		if len(errs) > 0 {
			slog.Error("error load external plugin", "error", fmt.Sprintf("%v", errs))
		}
	}

	core.ProvideLazy(func() (core.Cache, error) {
		return cache.NewCache(config.Cache)
	})
	core.ProvideLazy(func() (core.Cookie, error) {
		return cache.NewCookie(config.Cache)
	})

	js.SetScheduler(js.NewScheduler(config.JS))
}
