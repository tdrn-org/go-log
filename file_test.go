// file_test.go
//
// Copyright (C) 2023-2025 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

package log_test

import (
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/tdrn-org/go-log"
)

func TestFileLogWithLimit(t *testing.T) {
	logdir := t.TempDir()
	config := &log.Config{
		Level:         slog.LevelDebug.String(),
		Target:        log.TargetFileText,
		AddSource:     true,
		FileName:      filepath.Join(logdir, "with-limit.log"),
		FileSizeLimit: 1024,
	}
	logger, _ := config.GetLogger()
	generateLogs(logger, slog.LevelInfo, slog.LevelError, 1000)
}

func TestFileLogWithoutLimit(t *testing.T) {
	logdir := t.TempDir()
	config := log.Config{
		Level:         slog.LevelDebug.String(),
		Target:        log.TargetFileText,
		AddSource:     true,
		FileName:      filepath.Join(logdir, "without-limit.log"),
		FileSizeLimit: 0,
	}
	logger, _ := config.GetLogger()
	generateLogs(logger, slog.LevelInfo, slog.LevelError, 1000)
}
