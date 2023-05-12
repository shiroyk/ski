package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/shiroyk/cloudcat/core"
	"github.com/shiroyk/cloudcat/core/js"
	"github.com/shiroyk/cloudcat/ctl/model"
	"github.com/shiroyk/cloudcat/ctl/utils"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/plugin"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
)

// ErrInvalidModel invalid models error
var ErrInvalidModel = errors.New("model is invalid")

var (
	runModelArg   string
	runOutputArg  string
	runScriptArg  string
	runTimeoutArg time.Duration
	runDebugArg   bool
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run a specified model or script",
	RunE: func(cmd *cobra.Command, args []string) error {
		switch {
		case runScriptArg != "":
			return runScript()
		case runModelArg != "":
			return analyzeModel()
		default:
			return errors.New("model and script cannot both be empty")
		}
	},
}

func analyzeModel() (err error) {
	var model model.Model
	var bytes []byte
	if runModelArg == "-" {
		bytes, err = io.ReadAll(os.Stdin)
	} else {
		bytes, err = os.ReadFile(runModelArg) //nolint:gosec
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

	fetcher := cloudcat.MustResolve[cloudcat.Fetch]()
	req, err := fetch.NewTemplateRequest(fetch.DefaultTemplateFuncMap(), model.Source.HTTP, nil)
	req = fetch.WithRequestConfig(req, fetch.RequestConfig{Proxy: model.Source.Proxy})
	if err != nil {
		return err
	}

	ctx := plugin.NewContext(plugin.Options{
		Timeout: cloudcat.ZeroOr(model.Source.Timeout, runTimeoutArg),
		Logger:  slog.New(loggerHandler()),
		URL:     model.Source.HTTP,
	})
	defer ctx.Cancel()

	res, err := fetch.DoString(fetcher, req.WithContext(ctx))
	if err != nil {
		return err
	}

	return outputJSON(cloudcat.Analyze(ctx, model.Schema, res))
}

func runScript() (err error) {
	var bytes []byte
	if runScriptArg == "-" {
		bytes, err = io.ReadAll(os.Stdin)
	} else {
		bytes, err = os.ReadFile(runScriptArg) //nolint:gosec
	}
	if err != nil {
		return
	}

	ctx := plugin.NewContext(plugin.Options{
		Timeout: runTimeoutArg,
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
	if runDebugArg {
		log = utils.NewConsoleHandler(slog.LevelDebug)
	}
	return log
}

func outputJSON(data any) (err error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if runOutputArg == "" {
		fmt.Println(string(bytes)) //nolint:forbidigo
		return
	}

	ext := filepath.Ext(runOutputArg)
	if ext == "" {
		runOutputArg += ".json"
	}
	err = os.WriteFile(runOutputArg, bytes, 0o600)
	if err != nil {
		return
	}
	return
}

func init() {
	runCmd.Flags().StringVarP(&runModelArg, "model", "m", "", "run a model")
	runCmd.Flags().StringVarP(&runScriptArg, "script", "s", "", "run a script")
	runCmd.Flags().DurationVarP(&runTimeoutArg, "timeout", "t", plugin.DefaultTimeout, "run timeout")
	runCmd.Flags().StringVarP(&runOutputArg, "output", "o", "", "write to file instead of stdout")
	runCmd.Flags().BoolVarP(&runDebugArg, "debug", "d", false, "output the debug log for running")
	rootCmd.AddCommand(runCmd)
}
