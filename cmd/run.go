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
)

// ErrInvalidModel invalid models error
var ErrInvalidModel = errors.New("model is invalid")

var (
	modelPath  = ""
	outputPath = ""
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a specific model",
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
	req, err := fetch.NewTemplateRequest(nil, model.Source.URL, nil)
	req.Proxy = model.Source.Proxy
	if err != nil {
		return err
	}

	res, err := fetcher.DoRequest(req)
	if err != nil {
		return err
	}

	ctx := parser.NewContext(parser.Options{
		Timeout: model.Source.Timeout,
		URL:     model.Source.URL,
	})
	defer ctx.Cancel()

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
	runCmd.PersistentFlags().StringVarP(&modelPath, "model", "m", "", "Model yml/yaml file path")
	runCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Write to file instead of stdout")
	rootCmd.AddCommand(runCmd)
}
