// log.go
//
// Copyright (C) 2023 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

package log

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
)

type Color int

// Coloring mode
const (
	// Force coloring off
	ColorOff Color = -1
	// Auto-detect coloring
	ColorAuto Color = 0
	// Force coloring on
	ColorOn Color = 1
)

var defaultLogger = NewConsoleLogger(os.Stderr, ColorAuto)
var rootLogger = defaultLogger
var rootLoggerMutex sync.RWMutex

// RootLogger gets the currently set root logger
func RootLogger() *zerolog.Logger {
	rootLoggerMutex.RLock()
	defer rootLoggerMutex.RUnlock()
	return rootLogger
}

func SetDefaultRootLogger() *zerolog.Logger {
	return SetRootLogger(defaultLogger, zerolog.WarnLevel, zerolog.TimeFormatUnix)
}

// SetRootLogger sets a new root logger as well as log level and time field format
func SetRootLogger(logger *zerolog.Logger, level zerolog.Level, timeFieldFormat string) *zerolog.Logger {
	rootLoggerMutex.Lock()
	defer rootLoggerMutex.Unlock()
	if rootLogger != logger {
		previousRootLogger := rootLogger
		rootLogger = logger
		if previousRootLogger == defaultLogger {
			rootLogger.Info().Msg("root logger set")
		} else {
			rootLogger.Info().Msg("root logger re-set")
		}
	}
	log.SetFlags(0)
	log.SetOutput(rootLogger)
	setLevel(level)
	setTimeFieldFormat(timeFieldFormat)
	return rootLogger
}

// SetLevel sets the log level
func SetLevel(level zerolog.Level) {
	rootLoggerMutex.Lock()
	defer rootLoggerMutex.Unlock()
	setLevel(level)
}

func setLevel(level zerolog.Level) {
	previousLevel := zerolog.GlobalLevel()
	if previousLevel != level {
		zerolog.SetGlobalLevel(level)
		rootLogger.Info().Msgf("adjust log level %s -> %s", previousLevel, level)
	}
}

// SetTimeFieldFormat sets the time field format
func SetTimeFieldFormat(timeFieldFormat string) {
	rootLoggerMutex.Lock()
	defer rootLoggerMutex.Unlock()
	setTimeFieldFormat(timeFieldFormat)
}

func setTimeFieldFormat(timeFieldFormat string) {
	previousTimeFieldFormat := zerolog.TimeFieldFormat
	if previousTimeFieldFormat != timeFieldFormat {
		zerolog.TimeFieldFormat = timeFieldFormat
		rootLogger.Info().Msgf("adjust time field format %s -> %s", previousTimeFieldFormat, timeFieldFormat)
	}
}

// NewConsoleLogger creates a new console logger
func NewConsoleLogger(out *os.File, color Color) *zerolog.Logger {
	writer := zerolog.ConsoleWriter{
		Out:        out,
		NoColor:    !colorFlag(out, color),
		TimeFormat: time.RFC3339,
	}
	logger := zerolog.New(writer).With().Timestamp().Logger()
	return &logger
}

func colorFlag(out *os.File, color Color) bool {
	switch color {
	case ColorOff:
		return false
	case ColorAuto:
		return isatty.IsTerminal(out.Fd()) || isatty.IsCygwinTerminal(out.Fd())
	case ColorOn:
		return true
	}
	return false
}
