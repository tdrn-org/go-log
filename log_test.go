//
// Copyright (C) 2023-2025 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

package log_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-log"
)

func TestConfigGetLevel(t *testing.T) {
	config := &log.Config{}
	require.Equal(t, slog.LevelInfo, config.GetLevel())

	config.Level = slog.LevelDebug.String()
	require.Equal(t, slog.LevelDebug, config.GetLevel())

	config.Level = slog.LevelInfo.String()
	require.Equal(t, slog.LevelInfo, config.GetLevel())

	config.Level = slog.LevelWarn.String()
	require.Equal(t, slog.LevelWarn, config.GetLevel())

	config.Level = slog.LevelError.String()
	require.Equal(t, slog.LevelError, config.GetLevel())

	config.Level = "undefined"
	require.Equal(t, slog.LevelInfo, config.GetLevel())
}

func TestConfigGetWriter(t *testing.T) {
	config := &log.Config{}
	require.Equal(t, os.Stderr, config.GetWriter())

	config.Target = log.TargetStdout
	require.Equal(t, os.Stdout, config.GetWriter())

	config.Target = log.TargetStdoutText
	require.Equal(t, os.Stdout, config.GetWriter())

	config.Target = log.TargetStdoutJSON
	require.Equal(t, os.Stdout, config.GetWriter())

	config.Target = log.TargetStderr
	require.Equal(t, os.Stderr, config.GetWriter())

	config.Target = log.TargetStderrText
	require.Equal(t, os.Stderr, config.GetWriter())

	config.Target = log.TargetStderrJSON
	require.Equal(t, os.Stderr, config.GetWriter())

	config.Target = log.Target("undefined")
	require.Equal(t, os.Stderr, config.GetWriter())
}

func TestConfigGetHandler(t *testing.T) {
	plainHandler := log.NewPlainHandler(os.Stderr, nil)
	textHandler := slog.NewTextHandler(os.Stderr, nil)
	jsonHandler := slog.NewJSONHandler(os.Stderr, nil)

	config := &log.Config{}
	handler, _ := config.GetHandler(nil)
	require.IsType(t, textHandler, handler)

	config.Target = log.TargetStdout
	handler, _ = config.GetHandler(nil)
	require.IsType(t, plainHandler, handler)

	config.Target = log.TargetStdoutText
	handler, _ = config.GetHandler(nil)
	require.IsType(t, textHandler, handler)

	config.Target = log.TargetStdoutJSON
	handler, _ = config.GetHandler(nil)
	require.IsType(t, jsonHandler, handler)

	config.Target = log.TargetStderr
	handler, _ = config.GetHandler(nil)
	require.IsType(t, plainHandler, handler)

	config.Target = log.TargetStderrText
	handler, _ = config.GetHandler(nil)
	require.IsType(t, textHandler, handler)

	config.Target = log.TargetStderrJSON
	handler, _ = config.GetHandler(nil)
	require.IsType(t, jsonHandler, handler)

	config.Target = log.TargetFileText
	handler, _ = config.GetHandler(nil)
	require.IsType(t, textHandler, handler)

	config.Target = log.TargetFileJSON
	handler, _ = config.GetHandler(nil)
	require.IsType(t, jsonHandler, handler)
}

func TestInitDefaultArgs(t *testing.T) {
	log.InitFromFlags(nil, nil)
	require.True(t, slog.Default().Enabled(context.Background(), slog.LevelInfo))
	require.False(t, slog.Default().Enabled(context.Background(), slog.LevelDebug))
}

func TestInitSilentArgs(t *testing.T) {
	log.InitFromFlags([]string{"--silent"}, nil)
	require.True(t, slog.Default().Enabled(context.Background(), slog.LevelError))
	require.False(t, slog.Default().Enabled(context.Background(), slog.LevelWarn))
}

func TestInitQuietArgs(t *testing.T) {
	log.InitFromFlags([]string{"--quiet"}, nil)
	require.True(t, slog.Default().Enabled(context.Background(), slog.LevelWarn))
	require.False(t, slog.Default().Enabled(context.Background(), slog.LevelInfo))
}

func TestInitVerboseArgs(t *testing.T) {
	log.InitFromFlags([]string{"--verbose"}, nil)
	require.True(t, slog.Default().Enabled(context.Background(), slog.LevelInfo))
	require.False(t, slog.Default().Enabled(context.Background(), slog.LevelDebug))
}

func TestInitDebugArgs(t *testing.T) {
	log.InitFromFlags([]string{"--debug"}, nil)
	require.True(t, slog.Default().Enabled(context.Background(), slog.LevelDebug))
	require.False(t, slog.Default().Enabled(context.Background(), slog.LevelDebug-1))
}

func TestInitDefault(t *testing.T) {
	log.InitDefault()
	require.True(t, slog.Default().Enabled(context.Background(), slog.LevelInfo))
	require.False(t, slog.Default().Enabled(context.Background(), slog.LevelDebug))
}

func TestInitDebug(t *testing.T) {
	log.InitDebug()
	require.True(t, slog.Default().Enabled(context.Background(), slog.LevelDebug))
	require.False(t, slog.Default().Enabled(context.Background(), slog.LevelDebug-1))
}

func generateLogs(logger *slog.Logger, min slog.Level, max slog.Level, n int, args ...any) {
	for i := range n {
		level := slog.Level(int(min) + (i % (int(max-min) + 1)))
		logArgs := make([]any, 0, len(args)+2)
		logArgs = append(logArgs, slog.String("tag", "test"))
		logArgs = append(logArgs, slog.Int("index", i))
		logArgs = append(logArgs, args...)
		logger.Log(context.Background(), level, "test message", logArgs...)
	}
}
