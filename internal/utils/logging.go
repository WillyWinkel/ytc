package utils

import (
	"golang.org/x/exp/slog"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
)

// SetupLogging configures slog to always log to stdout and, if logfile is set, also to a rotating file.
func SetupLogging(logfile string) {
	consoleHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	var handler slog.Handler = consoleHandler

	if logfile != "" {
		fileWriter := &lumberjack.Logger{
			Filename:   logfile,
			MaxSize:    10, // megabytes
			MaxBackups: 5,
			MaxAge:     28,   //days
			Compress:   true, // compress rotated files
		}
		multiWriter := io.MultiWriter(os.Stdout, fileWriter)
		handler = slog.NewTextHandler(multiWriter, &slog.HandlerOptions{Level: slog.LevelDebug})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
