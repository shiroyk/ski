package cmd

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"

	_ "github.com/shiroyk/cloudcat/analyzer"
	"github.com/shiroyk/cloudcat/ext"
	"golang.org/x/exp/slog"
)

const banner = `
        .__                   .___             __   
   ____ |  |   ____  __ __  __| _/____ _____ _/  |_ 
 _/ ___\|  |  /  _ \|  |  \/ __ |/ ___\\__  \\   __\
 \  \___|  |_(  <_> )  |  / /_/ \  \___ / __ \|  |  
  \___  >____/\____/|____/\____ |\___  >____  /__|  
      \/                       \/    \/     \/    
`

var (
	metaFlag       = flag.String("m", "", "Meta yml/yaml file path")
	configFlag     = flag.String("c", "~/.config/cloudcat/config.yml", "Config file path")
	outputFlag     = flag.String("o", "", "Write to file instead of stdout")
	versionFlag    = flag.Bool("v", false, "Version")
	extensionsFlag = flag.Bool("e", false, "Extensions list")
)

func version() string {
	return fmt.Sprintf("%v\n\t\tcloudcat %v %v", banner, "v1", runtime.Version())
}

// Execute main command
func Execute() {
	flag.Parse()
	var err error

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout)))

	config, err := readConfig(*configFlag)
	if err != nil {
		fmt.Printf("Error reading config file: \n %v", err)
	}

	if file := *metaFlag; file != "" {
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
