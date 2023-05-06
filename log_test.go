// log.go
//
// Copyright (C) 2023 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

package log

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestSetDefaultRootLogger(t *testing.T) {
	_ = SetDefaultRootLogger()
	require.Equal(t, zerolog.WarnLevel, zerolog.GlobalLevel())
	require.Equal(t, zerolog.TimeFormatUnix, zerolog.TimeFieldFormat)
}

func TestSetRootLogger(t *testing.T) {
	_ = SetRootLogger(NewConsoleLogger(os.Stdout, ColorOff), zerolog.TraceLevel, zerolog.TimeFormatUnixMs)
	require.Equal(t, zerolog.TraceLevel, zerolog.GlobalLevel())
	require.Equal(t, zerolog.TimeFormatUnixMs, zerolog.TimeFieldFormat)
}
