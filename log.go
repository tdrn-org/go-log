// log.go
//
// Copyright (C) 2023-2024 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

// Package log provides functionality for easy usage of the [github.com/rs/zerolog] logging framework.
package log

import (
	"io"
	stdlog "log"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/tdrn-org/go-log/console"
	"github.com/tdrn-org/go-log/file"
	"github.com/tdrn-org/go-log/syslog"
)

var defaultLogger = newDefaultLogger(console.NewDefaultWriter())
var defaultLevel = zerolog.WarnLevel
var defaultTimeFieldFormat = time.RFC3339
var rootLogger = defaultLogger
var rootLoggerMutex sync.RWMutex

func newDefaultLogger(w io.Writer) *zerolog.Logger {
	return NewLogger(w, true)
}

// NewLogger creates a new [github.com/rs/zerolog.Logger] for the given options.
func NewLogger(w io.Writer, timestamp bool) *zerolog.Logger {
	logger := zerolog.New(w)
	if timestamp {
		logger = logger.With().Timestamp().Logger()
	}
	return &logger
}

// RootLogger gets the current root logger.
func RootLogger() *zerolog.Logger {
	rootLoggerMutex.RLock()
	defer rootLoggerMutex.RUnlock()
	return rootLogger
}

// ResetRootLogger resets the root logger to it's default.
func ResetRootLogger() *zerolog.Logger {
	return SetRootLogger(defaultLogger, defaultLevel, defaultTimeFieldFormat)
}

// SetRootLogger sets a new root logger as well as log level and time field format.
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
	stdlog.SetFlags(0)
	stdlog.SetOutput(rootLogger)
	setLevel(level)
	setTimeFieldFormat(timeFieldFormat)
	return rootLogger
}

// RedirectRootLogger directs the root logger to the given [io.Writer] or optionally [zerolog.LevelWriter].
func RedirectRootLogger(w io.Writer, timestamp bool) *zerolog.Logger {
	return SetRootLogger(NewLogger(w, timestamp), defaultLevel, defaultTimeFieldFormat)
}

// Config provides a plugable interface for runtime logging configuration.
type Config interface {
	// Logger creates the [github.com/rs/zerolog.Logger]
	Logger() *zerolog.Logger
	// Level gets the global log level to use.
	Level() zerolog.Level
	// TimeFieldFormat gets the time field format to use.
	TimeFieldFormat() string
}

// SetRootLoggerFromConfig sets a new root logger as well as log level and time field format using a [github.com/tdrn-org/go-log/Config] interface.
func SetRootLoggerFromConfig(config Config) *zerolog.Logger {
	return SetRootLogger(config.Logger(), config.Level(), config.TimeFieldFormat())
}

// SetLevel sets the log level.
func SetLevel(level zerolog.Level) {
	rootLoggerMutex.Lock()
	defer rootLoggerMutex.Unlock()
	setLevel(level)
}

func setLevel(level zerolog.Level) {
	previousLevel := zerolog.GlobalLevel()
	if previousLevel != level {
		zerolog.SetGlobalLevel(level)
		rootLogger.Info().Msgf("adjusting log level '%s' -> '%s'", previousLevel, level)
	}
}

// SetTimeFieldFormat sets the time field format.
func SetTimeFieldFormat(timeFieldFormat string) {
	rootLoggerMutex.Lock()
	defer rootLoggerMutex.Unlock()
	setTimeFieldFormat(timeFieldFormat)
}

func setTimeFieldFormat(timeFieldFormat string) {
	previousTimeFieldFormat := zerolog.TimeFieldFormat
	if previousTimeFieldFormat != timeFieldFormat {
		zerolog.TimeFieldFormat = timeFieldFormat
		rootLogger.Info().Msgf("adjusting time field format '%s' -> '%s'", previousTimeFieldFormat, timeFieldFormat)
	}
}

// YAMLConfig supports a YAML file based logging configuration.
type YAMLConfig struct {
	LevelOption           string                    `yaml:"level"`
	TimestampOption       bool                      `yaml:"timestamp"`
	TimeFieldFormatOption string                    `yaml:"timeFieldFormat"`
	Console               console.YAMLConsoleConfig `yaml:"console"`
	File                  file.YAMLFileConfig       `yaml:"file"`
	Syslog                syslog.YAMLSyslogConfig   `yaml:"syslog"`
}

func (config *YAMLConfig) Logger() *zerolog.Logger {
	writers := make([]io.Writer, 0)
	if config.Console.EnabledOption {
		writers = append(writers, config.Console.NewWriter())
	}
	if config.File.EnabledOption {
		writers = append(writers, config.File.NewWriter())
	}
	if config.Syslog.EnabledOption {
		writers = append(writers, config.Syslog.NewWriter())
	}
	var logger *zerolog.Logger
	switch len(writers) {
	case 0:
		logger = defaultLogger
	case 1:
		logger = NewLogger(writers[0], config.TimestampOption)
	default:
		logger = NewLogger(zerolog.MultiLevelWriter(writers...), config.TimestampOption)
	}
	return logger
}

func (config *YAMLConfig) Level() zerolog.Level {
	level, err := zerolog.ParseLevel(config.LevelOption)
	if err != nil {
		return zerolog.WarnLevel
	}
	return level
}

func (config *YAMLConfig) TimeFieldFormat() string {
	return config.TimeFieldFormatOption
}

func init() {
	zerolog.SetGlobalLevel(defaultLevel)
}
