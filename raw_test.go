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

	"github.com/tdrn-org/go-log"
)

func TestRawHandler(t *testing.T) {
	h := log.NewRawHandler(os.Stdout)
	logger := slog.New(h)
	generateLogs(logger, slog.LevelDebug, log.LevelNotice, 100)
}
