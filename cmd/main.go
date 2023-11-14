package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"os"
	"path/filepath"
	"strings"

	"log/slog"

	"github.com/shiroyk/cloudcat"
	"github.com/shiroyk/cloudcat/js"
	_ "github.com/shiroyk/cloudcat/js/modules"
	_ "github.com/shiroyk/cloudcat/parsers"
	"github.com/shiroyk/cloudcat/plugin"
	"gopkg.in/yaml.v3"
)

// Model the model
type Model struct {
	Source struct {
		Name  string `yaml:"name"`
		HTTP  string `yaml:"http"`
		Proxy string `yaml:"proxy"`
	} `yaml:"source"`
	Schema *cloudcat.Schema `yaml:"schema"`
}

var (
	scriptFlag  = flag.String("s", "", "run script")
	modelFlag   = flag.String("m", "", "run model")
	timeoutFlag = flag.Duration("t", plugin.DefaultTimeout, "run timeout")
	debugFlag   = flag.Bool("d", false, "output the debug log")
	outputFlag  = flag.String("o", "", "write to file instead of stdout")
	pluginFlag  = flag.String("p", "", "plugin directory path")
	versionFlag = flag.Bool("v", false, "output version")
)

func runModel() (err error) {
	var bytes []byte
	if *modelFlag == "-" {
		bytes, err = io.ReadAll(os.Stdin)
	} else {
		bytes, err = os.ReadFile(*modelFlag) //nolint:gosec
	}
	if err != nil {
		return
	}

	var model Model
	err = yaml.Unmarshal(bytes, &model)
	if err != nil {
		return
	}

	if model.Source.HTTP == "" || model.Schema == nil {
		return errors.New("model is invalid")
	}

	timeout := plugin.DefaultTimeout
	if timeoutFlag != nil {
		timeout = *timeoutFlag
	}

	requestURI := model.Source.HTTP
	if _, urlErr := urlpkg.Parse(requestURI); urlErr == nil {
		requestURI = fmt.Sprintf("GET %s HTTP/1.1\n\n", requestURI)
	}

	req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(requestURI)))
	if err != nil {
		return err
	}
	req.RequestURI = ""

	ctx := plugin.NewContext(plugin.ContextOptions{
		Timeout: timeout,
		Logger:  slog.New(loggerHandler()),
		URL:     req.URL.String(),
	})
	defer ctx.Cancel()

	fetch := cloudcat.MustResolve[cloudcat.Fetch]()

	if model.Source.Proxy != "" {
		url, err := urlpkg.Parse(model.Source.Proxy)
		if err != nil {
			return err
		}
		req = req.WithContext(cloudcat.WithProxyURL(ctx, url))
	} else {
		req = req.WithContext(ctx)
	}

	res, err := fetch.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return outputJSON(cloudcat.Analyze(ctx, model.Schema, string(body)))
}

func runScript() (err error) {
	var bytes []byte
	if *scriptFlag == "-" {
		bytes, err = io.ReadAll(os.Stdin)
	} else {
		bytes, err = os.ReadFile(*scriptFlag) //nolint:gosec
	}
	if err != nil {
		return
	}

	timeout := plugin.DefaultTimeout
	if timeoutFlag != nil {
		timeout = *timeoutFlag
	}

	ctx := plugin.NewContext(plugin.ContextOptions{
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
	opt := new(slog.HandlerOptions)
	if *debugFlag {
		opt.Level = slog.LevelDebug
	}
	return slog.NewTextHandler(os.Stdout, opt)
}

func outputJSON(data any) (err error) {
	bytes, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	if *outputFlag == "" {
		fmt.Println(string(bytes)) //nolint:forbidigo
		return
	}

	ext := filepath.Ext(*outputFlag)
	if ext == "" {
		*outputFlag += ".json"
	}
	return os.WriteFile(*outputFlag, bytes, 0o600)
}

// expandPath expands path "." or "~"
func expandPath(path string) (string, error) {
	// expand local directory
	if strings.HasPrefix(path, ".") {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(cwd, path[1:]), nil
	}
	// expand ~ as shortcut for home directory
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[1:]), nil
	}
	return path, nil
}

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Println(fmt.Sprintf("cloudcat %v/%v", Version, CommitSHA))
		os.Exit(0)
		return
	}

	cloudcat.Provide(cloudcat.NewCache())
	cloudcat.Provide(cloudcat.NewCookie())
	cloudcat.ProvideLazy[cloudcat.Fetch](func() (cloudcat.Fetch, error) {
		transport := http.DefaultTransport.(*http.Transport)
		transport.Proxy = cloudcat.ProxyFromRequest
		client := &http.Client{Transport: transport}
		return client, nil
	})

	if pluginFlag != nil && *pluginFlag != "" {
		pluginPath, err := expandPath(*pluginFlag)
		if err != nil {
			return
		}
		err = plugin.LoadPlugin(pluginPath)
		if err != nil {
			panic(fmt.Sprintf("load external plugin fail %v", err))
		}
	}

	if scriptFlag != nil && *scriptFlag != "" {
		if err := runScript(); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	} else if modelFlag != nil && *modelFlag != "" {
		if err := runModel(); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	} else {
		flag.Usage()
	}
}
