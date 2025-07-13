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
	flag.StringVar(&logLevelFlag, "log", "warning", "Log level")
}

// SetLogLevel sets the log level to the provided cli flag
func SetLogLevel() error {
	level, ok := logLevelMap[strings.ToLower(logLevelFlag)]
	if !ok {
		err := fmt.Errorf("unknown log level: %v", logLevelFlag)
		return err
	}
	slog.SetLogLoggerLevel(level)
	return nil
}
