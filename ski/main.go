package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"

	_ "github.com/shiroyk/ski/modules/buffer"
	_ "github.com/shiroyk/ski/modules/encoding"
	_ "github.com/shiroyk/ski/modules/fetch"
	_ "github.com/shiroyk/ski/modules/signal"
	_ "github.com/shiroyk/ski/modules/stream"
	_ "github.com/shiroyk/ski/modules/timers"
	_ "github.com/shiroyk/ski/modules/url"

	_ "github.com/shiroyk/ski/modules/cache"
	_ "github.com/shiroyk/ski/modules/crypto"
	_ "github.com/shiroyk/ski/modules/encoding/base64"
	_ "github.com/shiroyk/ski/modules/ext"
)

var (
	timeoutFlag = flag.Duration("t", 0, "run timeout")
	outputFlag  = flag.String("o", "", "write to file instead of stdout")
	versionFlag = flag.Bool("v", false, "output version")
	logger      = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
)

func run() (err error) {
	var bytes []byte
	path := flag.Arg(0)
	if path == "-" {
		bytes, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
	} else {
		bytes, err = os.ReadFile(path) //nolint:gosec
		if err != nil {
			return fmt.Errorf("read script file: %w", err)
		}
	}

	ctx := context.Background()
	if timeoutFlag != nil && *timeoutFlag > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeoutFlag)
		defer cancel()
	}

	module, err := js.CompileModule("js", string(bytes))
	if err != nil {
		return err
	}

	ret, err := js.RunModule(js.WithLogger(ctx, logger), module)
	if err != nil {
		return err
	}

	if ret == nil || sobek.IsUndefined(ret) {
		return nil
	}

	if *outputFlag == "" {
		fmt.Println(ret.String()) //nolint:forbidigo
		return
	}

	ext := filepath.Ext(*outputFlag)
	if ext == "" {
		*outputFlag += ".txt"
	}
	return os.WriteFile(*outputFlag, []byte(ret.String()), 0o600)
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
