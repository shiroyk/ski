package cmd

import (
	"strings"

	"github.com/shiroyk/cloudcat/plugin"
	"github.com/spf13/cobra"
)

var extCmd = &cobra.Command{
	Use:     "extension",
	Aliases: []string{"ext"},
	Short:   "show extension list",
	Run: func(cmd *cobra.Command, args []string) {
		sb := new(strings.Builder)
		for _, e := range plugin.GetAll() {
			sb.WriteString(e.String())
			sb.WriteByte('\n')
		}
		cmd.Println(sb.String())
	},
}

func init() {
	rootCmd.AddCommand(extCmd)
}
