// log.go
//
// Copyright (C) 2023-2025 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

// Package log provides functionality for easy setup and integration of [log/slog] for application logging.
package log

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

const defaultLevel slog.Level = slog.LevelInfo

var emptyAttr = slog.Attr{}
var noAttrs = []slog.Attr{}
var noGroups = []string{}

// Target defines a logging destination as well as the [slog.Handler] to use
type Target string

const (
	// Log to stdout using the [PlainHandler]
	TargetStdout Target = "stdout"
	// Log to stdout using the [slog.TextHandler]
	TargetStdoutText Target = "text@stdout"
	// Log to stdout using the [slog.JSONHandler]
	TargetStdoutJSON Target = "json@stdout"
	// Log to stderr using the [PlainHandler]
	TargetStderr Target = "stderr"
	// Log to stderr using the [slog.TextHandler]
	TargetStderrText Target = "text@stderr"
	// Log to stderr using the [slog.JSONHandler]
	TargetStderrJSON Target = "json@stderr"
	// Log to a file using the [slog.TextHandler]
	TargetFileText Target = "text@file"
	// Log to a file using the [slog.JSONHandler]
	TargetFileJSON Target = "json@file"
)

// Color mode for console logging
type Color int

const (
	// Auto-detect coloring
	ColorAuto Color = -1
	// Force coloring off
	ColorOff Color = 0
	// Force coloring on
	ColorOn Color = 1
)

// Config defines a complete application logging setup
type Config struct {
	// Level defines the initial log level to accept
	Level string
	// AddSource controls whether to log source file and line
	AddSource bool
	// Target defines the logging destination as well as the
	// [slog.Handler] to use
	Target Target
	// Color sets the color mode of the [PlainHandler] (if used)
	Color Color
	// FileName defines the file to log into (for file targets)
	FileName string
	// FileSizeLimit defines the file size to rotate after
	// (values <= 0 disable rotation)
	FileSizeLimit int64
}

// GetLevel determines the [slog.Level] defined by this configuration.
//
// This function always returns a result, falling back to [slog.LevelInfo]
// in case the configuration is not conclusive.
func (c *Config) GetLevel() slog.Level {
	switch strings.ToLower(c.Level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "":
		return defaultLevel
	}
	slog.Warn("unrecognized log level", "level", slog.StringValue(c.Level))
	return defaultLevel
}

// GetWriter determines the [io.Writer] defined by this configuration.
//
// This function always returns a result, falling back to [os.Stderr]
// in case the configuration is not conclusive.
func (c *Config) GetWriter() io.Writer {
	switch c.Target {
	case TargetStdout:
		return os.Stdout
	case TargetStdoutText:
		return os.Stdout
	case TargetStdoutJSON:
		return os.Stdout
	case TargetStderr:
		return os.Stderr
	case TargetStderrText:
		return os.Stderr
	case TargetStderrJSON:
		return os.Stderr
	case TargetFileText:
		return &fileWriter{fileName: c.FileName, fileSizeLimit: c.FileSizeLimit}
	case TargetFileJSON:
		return &fileWriter{fileName: c.FileName, fileSizeLimit: c.FileSizeLimit}
	case "":
		return os.Stderr
	}
	slog.Warn("unrecognized target option", "target", slog.StringValue(string(c.Target)))
	return os.Stderr
}

// GetHandler determines the [slog.Handler] defined by this configuration.
//
// This function always returns a result, falling back to [slog.TextHandler]
// in case the configuration is not conclusive.
// Beside the handler this function also returns the [slog.Leveler] instance
// assigned to it, enabling dynamic changing of the log level.
func (c *Config) GetHandler() (slog.Handler, slog.Leveler) {
	leveler := &slog.LevelVar{}
	leveler.Set(c.GetLevel())
	switch c.Target {
	case TargetStdout:
		return c.getPlainHandler(leveler)
	case TargetStdoutText:
		return c.getTextHandler(leveler)
	case TargetStdoutJSON:
		return c.getJSONHandler(leveler)
	case TargetStderr:
		return c.getPlainHandler(leveler)
	case TargetStderrText:
		return c.getTextHandler(leveler)
	case TargetStderrJSON:
		return c.getJSONHandler(leveler)
	case TargetFileText:
		return c.getTextHandler(leveler)
	case TargetFileJSON:
		return c.getJSONHandler(leveler)
	case "":
		return c.getTextHandler(leveler)
	}
	slog.Warn("unrecognized target option", "target", slog.StringValue(string(c.Target)))
	return c.getTextHandler(leveler)
}

func (c *Config) getPlainHandler(leveler slog.Leveler) (slog.Handler, slog.Leveler) {
	w := c.GetWriter()
	opts := &PlainHandlerOptions{
		HandlerOptions: slog.HandlerOptions{
			AddSource: c.AddSource,
			Level:     leveler,
		},
		Color: c.Color,
	}
	return NewPlainHandler(w, opts), leveler
}

func (c *Config) getTextHandler(leveler slog.Leveler) (slog.Handler, slog.Leveler) {
	w := c.GetWriter()
	opts := &slog.HandlerOptions{
		AddSource: c.AddSource,
		Level:     leveler,
	}
	return slog.NewTextHandler(w, opts), leveler
}

func (c *Config) getJSONHandler(leveler slog.Leveler) (slog.Handler, slog.Leveler) {
	w := c.GetWriter()
	opts := &slog.HandlerOptions{
		AddSource: c.AddSource,
		Level:     leveler,
	}
	return slog.NewJSONHandler(w, opts), leveler
}

// GetLogger determines the [slog.Logger] defined by this configuration.
//
// This function simply wraps [GetHandler] into a [slog.New] call.
// Beside the logger this function also returns the [slog.Leveler] instance
// assigned to it, enabling dynamic changing of the log level.
func (c *Config) GetLogger() (*slog.Logger, slog.Leveler) {
	h, l := c.GetHandler()
	return slog.New(h), l
}
