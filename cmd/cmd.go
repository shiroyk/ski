package cmd

import (
	"os"

	_ "github.com/shiroyk/cloudcat/analyzer"
	"github.com/shiroyk/cloudcat/lib/logger"
)

// Execute main command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error("execute failed", err)
		os.Exit(1)
	}
}
