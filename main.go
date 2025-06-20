package main

import (
	"github.com/WillyWinkel/ytc/internal/app"
	"github.com/WillyWinkel/ytc/internal/utils"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
	"os"
)

var port string
var logfile string

func main() {
	var rootCmd = &cobra.Command{
		Use:   "ytc-server",
		Short: "YTC Server",
		Run: func(cmd *cobra.Command, args []string) {
			utils.SetupLogging(logfile)
			err := app.Server(port)
			if err != nil {
				slog.Error("failed to run server", "err", err.Error())
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().StringVarP(&port, "port", "p", "80", "Port to run the server on")
	rootCmd.Flags().StringVar(&logfile, "logfile", "", "Log file path pattern (enables file logging with rotation)")

	if err := rootCmd.Execute(); err != nil {
		slog.Error("command execution failed", "err", err.Error())
		os.Exit(1)
	}
}
