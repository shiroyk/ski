package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shiroyk/cloudcat/analyzer"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/lib/logger"
	"github.com/shiroyk/cloudcat/lib/utils"
	"github.com/shiroyk/cloudcat/parser"
	"github.com/shiroyk/cloudcat/schema"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
)

// ErrInvalidModel invalid models error
var ErrInvalidModel = errors.New("model is invalid")

var (
	modelPath  string
	outputPath string
	debugMode  bool
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run a specific model",
	Run: func(cmd *cobra.Command, args []string) {
		if err := run(modelPath, outputPath); err != nil {
			logger.Error("run failed", err)
		}
	},
}

func run(path, output string) (err error) {
	model, err := utils.ReadYaml[schema.Model](path)
	if err != nil {
		return err
	}
	if model.Source == nil || model.Schema == nil {
		return ErrInvalidModel
	}

	fetcher := di.MustResolve[fetch.Fetch]()
	req, err := fetch.NewTemplateRequest(nil, model.Source.HTTP, nil)
	req.Proxy = model.Source.Proxy
	if err != nil {
		return err
	}

	var log slog.Handler = slog.NewTextHandler(os.Stdout)
	if debugMode {
		log = logger.NewConsoleHandler(slog.LevelDebug)
	}

	ctx := parser.NewContext(parser.Options{
		Timeout: model.Source.Timeout,
		Logger:  slog.New(log),
		URL:     model.Source.HTTP,
	})
	defer ctx.Cancel()

	res, err := fetcher.DoRequest(req.WithContext(ctx))
	if err != nil {
		return err
	}

	result := analyzer.Analyze(ctx, model.Schema, res.String())

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

func init() {
	runCmd.PersistentFlags().StringVarP(&modelPath, "model", "m", "", "model yml/yaml file path")
	runCmd.Flags().StringVarP(&outputPath, "output", "o", "", "write to file instead of stdout")
	runCmd.Flags().BoolVarP(&debugMode, "debug", "d", false, "output log for debugging parsing schema")
	rootCmd.AddCommand(runCmd)
}
