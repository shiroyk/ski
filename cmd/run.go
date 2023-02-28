package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/shiroyk/cloudcat/analyzer"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/js"
	"github.com/shiroyk/cloudcat/lib/logger"
	"github.com/shiroyk/cloudcat/lib/utils"
	"github.com/shiroyk/cloudcat/parser"
	"github.com/shiroyk/cloudcat/schema"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
)

// ErrInvalidModel invalid models error
var ErrInvalidModel = errors.New("model is invalid")

var (
	modelArg   string
	outputArg  string
	scriptArg  string
	timeoutArg string
	debugArg   bool
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run a specified model or script",
	RunE: func(cmd *cobra.Command, args []string) error {
		switch {
		case scriptArg != "":
			return runScript()
		case modelArg != "":
			return analyzeModel()
		default:
			return errors.New("model and script cannot both be empty")
		}
	},
}

func analyzeModel() (err error) {
	var model schema.Model
	var bytes []byte
	if modelArg == "-" {
		bytes, err = io.ReadAll(os.Stdin)
	} else {
		bytes, err = os.ReadFile(modelArg)
	}
	if err != nil {
		return
	}
	err = yaml.Unmarshal(bytes, &model)
	if err != nil {
		return
	}

	if model.Source == nil || model.Schema == nil {
		return ErrInvalidModel
	}

	fetcher := di.MustResolve[fetch.Fetch]()
	req, err := fetch.NewTemplateRequest(fetch.DefaultTemplateFuncMap(), model.Source.HTTP, nil)
	req.Proxy = model.Source.Proxy
	if err != nil {
		return err
	}

	timeout, err := timeoutDuration()
	if err != nil {
		return err
	}

	ctx := parser.NewContext(parser.Options{
		Timeout: utils.ZeroOr(model.Source.Timeout, timeout),
		Logger:  slog.New(loggerHandler()),
		URL:     model.Source.HTTP,
	})
	defer ctx.Cancel()

	res, err := fetcher.DoRequest(req.WithContext(ctx))
	if err != nil {
		return err
	}

	return outputJSON(analyzer.Analyze(ctx, model.Schema, res.String()))
}

func runScript() (err error) {
	var bytes []byte
	if scriptArg == "-" {
		bytes, err = io.ReadAll(os.Stdin)
	} else {
		bytes, err = os.ReadFile(scriptArg)
	}
	if err != nil {
		return
	}

	timeout, err := timeoutDuration()
	if err != nil {
		return err
	}

	ctx := parser.NewContext(parser.Options{
		Timeout: timeout,
		Logger:  slog.New(loggerHandler()),
	})
	defer ctx.Cancel()

	value, err := js.RunString(ctx, string(bytes))
	if err != nil {
		return err
	}

	return outputJSON(value)
}

func loggerHandler() slog.Handler {
	var log slog.Handler = slog.NewTextHandler(os.Stdout)
	if debugArg {
		log = logger.NewConsoleHandler(slog.LevelDebug)
	}
	return log
}

func timeoutDuration() (timeout time.Duration, err error) {
	if timeoutArg != "" {
		timeout, err = time.ParseDuration(timeoutArg)
		if err != nil {
			return
		}
	}
	return
}

func outputJSON(data any) (err error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if outputArg == "" {
		fmt.Println(string(bytes))
		return
	}

	ext := filepath.Ext(outputArg)
	if ext == "" {
		outputArg += ".json"
	}
	err = os.WriteFile(outputArg, bytes, 0644)
	if err != nil {
		return
	}
	return
}

func init() {
	runCmd.Flags().StringVarP(&modelArg, "model", "m", "", "run a model")
	runCmd.Flags().StringVarP(&scriptArg, "script", "s", "", "run a script")
	runCmd.Flags().StringVarP(&timeoutArg, "timeout", "t", "", "run timeout")
	runCmd.Flags().StringVarP(&outputArg, "output", "o", "", "write to file instead of stdout")
	runCmd.Flags().BoolVarP(&debugArg, "debug", "d", false, "output the debug log for running")
	rootCmd.AddCommand(runCmd)
}
