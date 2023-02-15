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
	"github.com/shiroyk/cloudcat/parser"
	"github.com/shiroyk/cloudcat/schema"
	"gopkg.in/yaml.v3"
)

var ErrInvalidMeta = errors.New("meta is invalid")

func run(path, output string) (err error) {
	if err = initDependencies(); err != nil {
		return err
	}

	meta, err := readMeta(path)
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

func initDependencies() error {
	di.Provide(fetch.NewFetcher(fetch.Options{}))
	di.Provide(fetch.DefaultTemplateFuncMap())
	cache, err := bolt.NewCache("")
	if err != nil {
		return err
	}
	di.Provide(cache)
	cookie, err := bolt.NewCookie("")
	if err != nil {
		return err
	}
	di.Provide(cookie)
	shortener, err := bolt.NewShortener("")
	if err != nil {
		return err
	}
	di.Provide(shortener)

	return nil
}

func readMeta(path string) (meta *schema.Meta, err error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return
	}

	meta = new(schema.Meta)
	err = yaml.Unmarshal(bytes, meta)
	if err != nil {
		return
	}

	return
}
