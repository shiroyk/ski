package cmd

import (
	"os"

	_ "github.com/shiroyk/cloudcat/analyzer"
)

// Execute main command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
