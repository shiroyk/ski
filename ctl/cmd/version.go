package cmd

import (
	"github.com/shiroyk/cloudcat/ctl/consts"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "print version information",
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Printf("%v\n cloudcat %v/%v\n", consts.Banner, consts.Version, consts.CommitSHA)
		},
	})
}
