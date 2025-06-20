package main

import (
	"github.com/WillyWinkel/ytc/internal/app"
	"log/slog"
	"os"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	err := app.Server()
	if err != nil {
		slog.Error("failed to run server", "err", err.Error())
		os.Exit(1)
	}
}
