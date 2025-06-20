package main

import (
	"github.com/WillyWinkel/ytc/internal/app"
	"github.com/spf13/cobra"
	"log/slog"
	"os"
)

var port string

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	var rootCmd = &cobra.Command{
		Use:   "ytc-server",
		Short: "YTC Server",
		Run: func(cmd *cobra.Command, args []string) {
			err := app.Server(port)
			if err != nil {
				slog.Error("failed to run server", "err", err.Error())
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().StringVarP(&port, "port", "p", "80", "Port to run the server on")
	if err := rootCmd.Execute(); err != nil {
		slog.Error("command execution failed", "err", err.Error())
		os.Exit(1)
	}
}
