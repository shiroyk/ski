package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/shiroyk/cloudcat/api"
	"github.com/spf13/cobra"
)

var (
	apiAddressArg string
	apiTokenArg   string
	apiTimeoutArg time.Duration
)

var rootCmd = &cobra.Command{
	Use:   "cloudcat",
	Short: "cloudcat is a tool for extracting structured data from websites.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if apiTokenArg == "" {
			bytes := make([]byte, 16)
			_, err := rand.Read(bytes)
			if err != nil {
				return err
			}
			apiTokenArg = hex.EncodeToString(bytes)
		}

		cmd.Printf("Secret: %v\n", apiTokenArg)
		cmd.Printf("Service start http://%s\n", apiAddressArg)

		return api.Server(api.Options{
			Address: apiAddressArg,
			Token:   apiTokenArg,
			Timeout: runTimeoutArg,
		}).ListenAndServe()
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.Flags().StringVarP(&apiAddressArg, "address", "a", api.DefaultAddress, "api service address")
	rootCmd.Flags().StringVarP(&apiTokenArg, "secret", "s", "", "api service secret")
	rootCmd.Flags().DurationVarP(&apiTimeoutArg, "timeout", "t", api.DefaultTimeout, "api service timeout")
}
