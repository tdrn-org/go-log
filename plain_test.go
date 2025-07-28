//
// Copyright (C) 2023-2025 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

package log_test

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/tdrn-org/go-log"
)

func TestPlainLogConfig(t *testing.T) {
	config := log.Config{
		Level:     slog.LevelDebug.String(),
		AddSource: true,
		Target:    log.TargetStdout,
		Color:     log.ColorOn,
	}
	logger, _ := config.GetLogger(nil)
	generateLogs(logger, slog.LevelDebug, slog.LevelError+1, 100)
}

func TestPlainHandler(t *testing.T) {
	h := log.NewPlainHandler(os.Stdout, &log.PlainHandlerOptions{
		HandlerOptions: slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
			ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
				switch attr.Key {
				case slog.TimeKey:
					return slog.Time(slog.TimeKey, time.Time{})
				case slog.SourceKey:
					return slog.Attr{}
				default:
					return attr
				}
			},
		},
		Color: log.ColorOn,
	})
	logger := slog.New(h)
	logger = logger.With(slog.Group("test", slog.String("name", "TestPlainHandler")))
	logger = logger.WithGroup("generate")
	generateLogs(logger, slog.LevelDebug, slog.LevelError+4, 100)
}
