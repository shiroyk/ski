package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/shiroyk/ski"
	"github.com/shiroyk/ski/js"

	_ "github.com/shiroyk/ski/modules/assert"
	_ "github.com/shiroyk/ski/modules/cache"
	_ "github.com/shiroyk/ski/modules/crypto"
	_ "github.com/shiroyk/ski/modules/encoding"
	_ "github.com/shiroyk/ski/modules/http"
	_ "github.com/shiroyk/ski/modules/timers"

	_ "github.com/shiroyk/ski/modules/gq"
	_ "github.com/shiroyk/ski/modules/jq"
	_ "github.com/shiroyk/ski/modules/xpath"
)

const defaultTimeout = time.Minute

var (
	timeoutFlag = flag.Duration("t", defaultTimeout, "run timeout")
	outputFlag  = flag.String("o", "", "write to file instead of stdout")
	versionFlag = flag.Bool("v", false, "output version")
	logger      = slog.New(loggerHandler())
)

func run() (err error) {
	var bytes []byte
	path := flag.Arg(0)
	if path == "-" {
		bytes, err = io.ReadAll(os.Stdin)
	} else {
		bytes, err = os.ReadFile(path) //nolint:gosec
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

	module, err := js.CompileModule("js", string(bytes))
	if err != nil {
		return err
	}

	ret, err := js.RunModule(js.WithLogger(ctx, logger), module)
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

	if err := run(); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
