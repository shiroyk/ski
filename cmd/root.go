package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cloudcat",
	Short: "cloudcat is a tool for extracting structured data from websites",
	Long: `cloudcat is a tool for extracting structured data from websites 
using YAML configuration and the syntax rule is extensible.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	cobra.OnInitialize(initConfig)
}
