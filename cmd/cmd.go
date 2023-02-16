package cmd

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	_ "github.com/shiroyk/cloudcat/analyzer"
	"github.com/shiroyk/cloudcat/ext"
	"github.com/shiroyk/cloudcat/lib"
	"golang.org/x/exp/slog"
)

var (
	metaFlag       = flag.String("m", "", "Meta yml/yaml file path")
	configFlag     = flag.String("c", "~/.config/cloudcat/config.yml", "Config file path")
	outputFlag     = flag.String("o", "", "Write to file instead of stdout")
	versionFlag    = flag.Bool("v", false, "Version")
	extensionsFlag = flag.Bool("e", false, "Extensions list")
)

func version() string {
	return fmt.Sprintf("%v\n cloudcat %v/%v\n", lib.Banner, lib.Version, lib.CommitSHA)
}

// Execute main command
func Execute() {
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout)))

	if file := *metaFlag; file != "" {
		config, err := lib.ReadConfig(*configFlag)
		if err != nil {
			fmt.Printf("Error reading config file: \n %v", err)
		}

		output := *outputFlag
		err = run(*config, file, output)
		if err != nil {
			fmt.Printf("Error run parse meta: \n %v%v", err, string(debug.Stack()))
		}
		return
	}

	if ver := *versionFlag; ver {
		fmt.Println(version())
		return
	}

	if extension := *extensionsFlag; extension {
		sb := new(strings.Builder)
		for _, e := range ext.GetAll() {
			sb.WriteString(e.String())
			sb.WriteByte('\n')
		}
		fmt.Println(sb.String())
		return
	}

	flag.Usage()
}
