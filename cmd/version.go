package cmd

import (
	"fmt"

	"github.com/shiroyk/cloudcat/lib/consts"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("%v\n cloudcat %v/%v\n", consts.Banner, consts.Version, consts.CommitSHA)
		},
	})
}
