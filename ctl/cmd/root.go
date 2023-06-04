package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"golang.org/x/exp/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shiroyk/cloudcat/ctl/api"
	"github.com/spf13/cobra"
)

var (
	apiAddressArg string
	apiTokenArg   string
	apiTimeoutArg time.Duration
	apiRequestLog bool
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

		server := api.Server(api.Options{
			Address:    apiAddressArg,
			Token:      apiTokenArg,
			Timeout:    runTimeoutArg,
			RequestLog: apiRequestLog,
		})

		signals := make(chan os.Signal)
		signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-signals
			if err := server.Close(); err != nil {
				slog.Error(err.Error())
			}
			os.Exit(1)
		}()

		slog.Info("Secret: %v\n", apiTokenArg)
		slog.Info("Service start http://%s\n", apiAddressArg)

		err := server.ListenAndServe()
		if err == http.ErrServerClosed {
			slog.Info("Service closed")
			return nil
		}
		return err
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.Flags().StringVarP(&apiAddressArg, "address", "a", api.DefaultAddress, "api service address")
	rootCmd.Flags().StringVarP(&apiTokenArg, "secret", "s", "", "api service secret")
	rootCmd.Flags().DurationVarP(&apiTimeoutArg, "timeout", "t", api.DefaultTimeout, "api service timeout")
	rootCmd.Flags().BoolVarP(&apiRequestLog, "request", "r", true, "api service request log output")
}
