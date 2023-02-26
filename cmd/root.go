package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/shiroyk/cloudcat/api"
	"github.com/shiroyk/cloudcat/lib/config"
	"github.com/shiroyk/cloudcat/lib/logger"
	"github.com/shiroyk/cloudcat/lib/utils"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cloudcat",
	Short: "cloudcat is a tool for extracting structured data from websites.",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.FromContext(cmd.Context())
		if cfg.Api.Token == "" {
			bytes := make([]byte, 16)
			_, err := rand.Read(bytes)
			if err != nil {
				logger.Error("generate token failed", err)
				return
			}
			cfg.Api.Token = hex.EncodeToString(bytes)
		}
		e := api.Server(cfg.Api)
		fmt.Printf("🔒Token: %v\n", cfg.Api.Token)
		e.Logger.Fatal(e.Start(utils.ZeroOr(cfg.Api.Address, api.DefaultAddress)))
	},
}

func init() {
	cobra.OnInitialize(initConfig)
}
