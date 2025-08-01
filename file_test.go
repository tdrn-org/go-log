//
// Copyright (C) 2023-2025 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

package log_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-log"
)

func TestFileLogWithLimit(t *testing.T) {
	logdir := t.TempDir()
	config := &log.Config{
		Level:         slog.LevelDebug.String(),
		Target:        log.TargetFileText,
		AddSource:     true,
		FileName:      filepath.Join(logdir, "with-limit.log"),
		FileSizeLimit: 4000,
	}
	logger, _ := config.GetLogger(nil)
	generateLogs(logger, slog.LevelInfo, log.LevelNotice, 100)
	entries, err := os.ReadDir(logdir)
	require.NoError(t, err)
	require.Len(t, entries, 4)
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
	logger, _ := config.GetLogger(nil)
	generateLogs(logger, slog.LevelInfo, log.LevelNotice, 100)
	entries, err := os.ReadDir(logdir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

func TestFileLogInvalidFile(t *testing.T) {
	logdir := t.TempDir()
	config := log.Config{
		Level:         slog.LevelDebug.String(),
		Target:        log.TargetFileText,
		FileName:      filepath.Join(logdir, "."),
		FileSizeLimit: 0,
	}
	logger, _ := config.GetLogger(nil)
	generateLogs(logger, slog.LevelInfo, log.LevelNotice, 10)
	entries, err := os.ReadDir(logdir)
	require.NoError(t, err)
	require.Len(t, entries, 0)
}
