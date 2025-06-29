package app

import (
	"flag"
	"fmt"
	"log/slog"
	"strings"
)

var (
	logLevelMap = map[string]slog.Level{
		"info":    slog.LevelInfo,
		"debug":   slog.LevelDebug,
		"warning": slog.LevelWarn,
	}
	logLevelFlag string
)

func init() {
	flag.StringVar(&logLevelFlag, "log", "INFO", "Log level")
}

func SetLogLevel() error {
	level, ok := logLevelMap[strings.ToLower(logLevelFlag)]
	if !ok {
		err := fmt.Errorf("Unknown log level: %v\n", logLevelFlag)
		return err
	}
	slog.SetLogLoggerLevel(level)
	return nil
}
