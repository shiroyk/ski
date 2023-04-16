package cmd

import (
	"os"
)

// Execute main command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
