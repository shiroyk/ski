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
	schemaFlag     = flag.String("f", "", "Schema yml/yaml filename")
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

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout)))

	if file := *schemaFlag; file != "" {
		output := *outputFlag
		err := run(file, output)
		if err != nil {
			fmt.Println(err, string(debug.Stack()))
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
