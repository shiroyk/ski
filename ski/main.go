package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shiroyk/ski"
	"github.com/shiroyk/ski/js"

	_ "github.com/shiroyk/ski/js/modules/cache"
	_ "github.com/shiroyk/ski/js/modules/crypto"
	_ "github.com/shiroyk/ski/js/modules/encoding"
	_ "github.com/shiroyk/ski/js/modules/http"

	_ "github.com/shiroyk/ski/executors/gq"
	_ "github.com/shiroyk/ski/executors/jq"
	_ "github.com/shiroyk/ski/executors/regex"
	_ "github.com/shiroyk/ski/executors/xpath"
)

const defaultTimeout = time.Minute

var (
	scriptFlag  = flag.String("s", "", "run script")
	modelFlag   = flag.String("m", "", "run model")
	timeoutFlag = flag.Duration("t", defaultTimeout, "run timeout")
	outputFlag  = flag.String("o", "", "write to file instead of stdout")
	versionFlag = flag.Bool("v", false, "output version")
)

type _fetch string

func new_fetch() ski.NewExecutor {
	return ski.StringExecutor(func(str string) (ski.Executor, error) {
		return _fetch(str), nil
	})
}

func (f _fetch) Exec(ctx context.Context, _ any) (any, error) {
	method, url, found := strings.Cut(string(f), " ")
	if !found {
		url = string(f)
		method = http.MethodGet
	}

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ski")

	res, err := ski.NewFetch().Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

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
	fmt.Println(string(bytes))

	ski.Register("fetch", new_fetch())

	executor, err := ski.Compile(string(bytes))
	if err != nil {
		return err
	}

	timeout := defaultTimeout
	if timeoutFlag != nil {
		timeout = *timeoutFlag
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ret, err := executor.Exec(ski.WithLogger(ctx, slog.New(loggerHandler())), nil)
	if err != nil {
		return err
	}

	return outputJSON(ret)
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

	timeout := defaultTimeout
	if timeoutFlag != nil {
		timeout = *timeoutFlag
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ctx = ski.NewContext(ctx, nil)

	vm, err := js.GetScheduler().Get()
	if err != nil {
		return err
	}
	module, err := vm.Loader().CompileModule("js", string(bytes))
	if err != nil {
		return err
	}

	ret, err := vm.RunModule(ski.WithLogger(ctx, slog.New(loggerHandler())), module)
	if err != nil {
		return err
	}

	v, err := js.Unwrap(ret)
	if err != nil {
		return err
	}

	return outputJSON(v)
}

func loggerHandler() slog.Handler {
	return slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
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

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Println(fmt.Sprintf("ski %v/%v", Version, CommitSHA))
		os.Exit(0)
		return
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
