// log_test.go
//
// Copyright (C) 2023-2024 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

package log_test

import (
	stdlog "log"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-log"
	"github.com/tdrn-org/go-log/console"
	"gopkg.in/yaml.v3"
)

func TestResetRootLogger(t *testing.T) {
	_ = log.ResetRootLogger()
	require.Equal(t, zerolog.WarnLevel, zerolog.GlobalLevel())
	require.Equal(t, time.RFC3339, zerolog.TimeFieldFormat)
}

func TestSetRootLogger(t *testing.T) {
	_ = log.SetRootLogger(log.NewLogger(console.NewDefaultWriter(), true), zerolog.TraceLevel, zerolog.TimeFormatUnixMs)
	require.Equal(t, zerolog.TraceLevel, zerolog.GlobalLevel())
	require.Equal(t, zerolog.TimeFormatUnixMs, zerolog.TimeFieldFormat)
}

func TestRedirectRootLogger(t *testing.T) {
	_ = log.RedirectRootLogger(os.Stderr, false)
	require.Equal(t, zerolog.WarnLevel, zerolog.GlobalLevel())
	require.Equal(t, time.RFC3339, zerolog.TimeFieldFormat)
}

func TestRedirectStdLog(t *testing.T) {
	log.RedirectStdLog()
	require.Equal(t, 0, stdlog.Flags())
}

func TestSetLevel(t *testing.T) {
	_ = log.ResetRootLogger()
	require.Equal(t, zerolog.WarnLevel, zerolog.GlobalLevel())
	log.SetLevel(zerolog.TraceLevel)
	require.Equal(t, zerolog.TraceLevel, zerolog.GlobalLevel())
}

func TestSetTimeFieldFormat(t *testing.T) {
	_ = log.ResetRootLogger()
	require.Equal(t, time.RFC3339, zerolog.TimeFieldFormat)
	log.SetTimeFieldFormat(zerolog.TimeFormatUnixMs)
	require.Equal(t, zerolog.TimeFormatUnixMs, zerolog.TimeFieldFormat)
}

func TestRootLogger(t *testing.T) {
	logger := log.RootLogger()
	require.NotNil(t, logger)
}

func TestConfig(t *testing.T) {
	configBytes, err := os.ReadFile("testdata/log.yaml")
	require.NoError(t, err)
	var config log.YAMLConfig
	err = yaml.Unmarshal(configBytes, &config)
	require.NoError(t, err)
	log.SetRootLoggerFromConfig(&config)
}
