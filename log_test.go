// log_test.go
//
// Copyright (C) 2023-2024 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

package log_test

import (
	"os"
	"testing"

	"github.com/hdecarne-github/go-log"
	"github.com/hdecarne-github/go-log/console"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestResetRootLogger(t *testing.T) {
	_ = log.ResetRootLogger()
	require.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())
	require.Equal(t, zerolog.TimeFormatUnix, zerolog.TimeFieldFormat)
}

func TestSetRootLogger(t *testing.T) {
	_ = log.SetRootLogger(log.NewLogger(console.NewDefaultWriter(), true), zerolog.TraceLevel, zerolog.TimeFormatUnixMs)
	require.Equal(t, zerolog.TraceLevel, zerolog.GlobalLevel())
	require.Equal(t, zerolog.TimeFormatUnixMs, zerolog.TimeFieldFormat)
}

func TestSetLevel(t *testing.T) {
	_ = log.ResetRootLogger()
	require.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())
	log.SetLevel(zerolog.TraceLevel)
	require.Equal(t, zerolog.TraceLevel, zerolog.GlobalLevel())
}

func TestSetTimeFieldFormat(t *testing.T) {
	_ = log.ResetRootLogger()
	require.Equal(t, zerolog.TimeFormatUnix, zerolog.TimeFieldFormat)
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
