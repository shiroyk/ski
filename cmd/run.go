package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shiroyk/cloudcat/analyzer"
	"github.com/shiroyk/cloudcat/cache/bolt"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/js"
	"github.com/shiroyk/cloudcat/lib"
	"github.com/shiroyk/cloudcat/lib/utils"
	"github.com/shiroyk/cloudcat/parser"
	"github.com/shiroyk/cloudcat/schema"
)

var ErrInvalidMeta = errors.New("meta is invalid")

func run(config lib.Config, path, output string) (err error) {
	if err = initDependencies(config); err != nil {
		return err
	}

	meta, err := utils.ReadYaml[schema.Meta](path)
	if err != nil {
		return err
	}
	if meta.Source == nil || meta.Schema == nil {
		return ErrInvalidMeta
	}

	fetcher := di.MustResolve[fetch.Fetch]()
	req, err := fetch.NewTemplateRequest(nil, meta.Source.URL, nil)
	if err != nil {
		return err
	}

	res, err := fetcher.DoRequest(req)
	if err != nil {
		return err
	}

	ctx := parser.NewContext(parser.Options{
		Timeout: meta.Source.Timeout,
		URL:     meta.Source.URL,
	})

	anal := analyzer.NewAnalyzer()
	result := anal.ExecuteSchema(ctx, meta.Schema, res.String())

	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}

	if output == "" {
		fmt.Println(string(bytes))
		return
	}

	ext := filepath.Ext(output)
	if ext == "" {
		output += ".json"
	}
	err = os.WriteFile(output, bytes, 0644)
	if err != nil {
		return
	}

	return
}

func initDependencies(config lib.Config) error {
	di.Provide(fetch.NewFetcher(fetch.Options{
		CharsetDetectDisabled: config.Fetch.CharsetDetectDisabled,
		MaxBodySize:           config.Fetch.MaxBodySize,
		RetryTimes:            config.Fetch.RetryTimes,
		RetryHTTPCodes:        config.Fetch.RetryHTTPCodes,
		Timeout:               config.Fetch.Timeout,
	}))
	di.Provide(fetch.DefaultTemplateFuncMap())
	cache, err := bolt.NewCache(config.Cache.Path)
	if err != nil {
		return err
	}
	di.Provide(cache)
	cookie, err := bolt.NewCookie(config.Cache.Path)
	if err != nil {
		return err
	}
	di.Provide(cookie)
	shortener, err := bolt.NewShortener(config.Cache.Path)
	if err != nil {
		return err
	}
	di.Provide(shortener)

	js.SetScheduler(js.NewScheduler(js.Options{
		InitialVMs:         config.JS.InitialVMs,
		MaxVMs:             config.JS.MaxVMs,
		MaxRetriesGetVM:    config.JS.MaxRetriesGetVM,
		MaxTimeToWaitGetVM: config.JS.MaxTimeToWaitGetVM,
		UseStrict:          config.JS.UseStrict,
	}))

	return nil
}
