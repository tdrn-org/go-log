//
// Copyright (C) 2023-2025 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

// Package log provides functionality for easy setup and integration
// of [log/slog] for application logging.
package log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

// LevelNotice defines a dedicated level (slog.LevelError + 4) used for
// important message not actually related to an error state but to be
// shown even in case of a high level filter.
const LevelNotice slog.Level = slog.LevelError + 4

// Notice emits a log message on [LevelNotice].
func Notice(logger *slog.Logger, msg string, args ...any) {
	logger.Log(context.Background(), LevelNotice, msg, args...)
}

const defaultLevel slog.Level = slog.LevelInfo

// Target defines a logging destination as well as the [slog.Handler] to use
type Target string

const (
	// Log to stdout using the PlainHandler
	TargetStdout Target = "stdout"
	// Log to stdout using the slog.TextHandler
	TargetStdoutText Target = "text@stdout"
	// Log to stdout using the slog.JSONHandler
	TargetStdoutJSON Target = "json@stdout"
	// Log to stderr using the PlainHandler
	TargetStderr Target = "stderr"
	// Log to stderr using the slog.TextHandler
	TargetStderrText Target = "text@stderr"
	// Log to stderr using the slog.JSONHandler
	TargetStderrJSON Target = "json@stderr"
	// Log to a file using the slog.TextHandler
	TargetFileText Target = "text@file"
	// Log to a file using the slog.JSONHandler
	TargetFileJSON Target = "json@file"
	// Log to syslog using the SyslogHandler
	TargetSyslog Target = "syslog"
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
	// slog.Handler to use
	Target Target
	// Color sets the color mode for the PlainHandler (if used)
	Color Color
	// FileName defines the file to log into (for file targets)
	FileName string
	// FileSizeLimit defines the file size to rotate after
	// (values <= 0 disable rotation)
	FileSizeLimit int64
	// SyslogNetwork defines the network to use for connecting
	// to the syslog server. Possible valures are:
	//	"udp"		// connect using udp
	//	"udp4"		// connect using udp (IPv4 only)
	//	"udp6"		// connect using udp (IPv6 only)
	//	"tcp"		// connect using tcp
	//	"tcp4"		// connect using tcp (IPv4 only)
	//	"tcp6"		// connect using tcp (IPv6 only)
	//	"tcp+tls"	// connect using TLS via tcp
	//	"tcp4+tls"	// connect using TLS via tcp (IPv4 only)
	//	"tcp6+tls"	// connect using TLS via tcp (IPv6 only)
	// Defaults to "tcp"
	SyslogNetwork string
	// SyslogAddress defines the syslog server address to
	// to connect to (host:port)
	SyslogAddress string
	// SyslogEncoding defines the syslog encoding to use.
	// Supported formats are:
	//	"rfc3164"			// RFC3164 (BSD) format with implicit framing
	//	"rfc3164+framing"	// RFC3164 (BSD) format with octet framing
	//	"rfc5424"			// RFC5424 (Syslog) format with implicit framing
	//	"rfc5424+framing"	// RFC5424 (Syslog) format with octet framing
	// Defaults to "rfc5424+framing"
	SyslogEncoding string
	// SyslogFaclity defines the syslog facility to use
	// (see https://datatracker.ietf.org/doc/html/rfc5424#section-6.2.1)
	SyslogFacility int
}

// GetLevel determines the [slog.Level] defined by this configuration.
//
// This function always returns a result, falling back to [slog.LevelInfo]
// in case the configuration is not conclusive.
func (c *Config) GetLevel() slog.Level {
	switch strings.ToLower(c.Level) {
	case "debug":
		return slog.LevelDebug
	case "debug+1":
		return slog.LevelDebug + 1
	case "debug+2":
		return slog.LevelDebug + 2
	case "debug+3":
		return slog.LevelDebug + 3
	case "info":
		return slog.LevelInfo
	case "info+1":
		return slog.LevelInfo + 1
	case "info+2":
		return slog.LevelInfo + 2
	case "info+3":
		return slog.LevelInfo + 3
	case "warn":
		return slog.LevelWarn
	case "warn+1":
		return slog.LevelWarn + 1
	case "warn+2":
		return slog.LevelWarn + 2
	case "warn+3":
		return slog.LevelWarn + 3
	case "error":
		return slog.LevelError
	case "error+1":
		return slog.LevelError + 1
	case "error+2":
		return slog.LevelError + 2
	case "error+3":
		return slog.LevelError + 3
	case "notice":
		return slog.LevelError + 4
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
	case TargetSyslog:
		return &syslogWriter{network: c.SyslogNetwork, address: c.SyslogAddress}
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
// Beside the handler this function also returns the [slog.LevelVar] instance
// assigned to it, enabling dynamic changing of the log level.
func (c *Config) GetHandler(levelVar *slog.LevelVar) (slog.Handler, *slog.LevelVar) {
	handlerLevelVar := levelVar
	if handlerLevelVar == nil {
		handlerLevelVar = &slog.LevelVar{}
	}
	handlerLevelVar.Set(c.GetLevel())
	switch c.Target {
	case TargetStdout:
		return c.getPlainHandler(handlerLevelVar)
	case TargetStdoutText:
		return c.getTextHandler(handlerLevelVar)
	case TargetStdoutJSON:
		return c.getJSONHandler(handlerLevelVar)
	case TargetStderr:
		return c.getPlainHandler(handlerLevelVar)
	case TargetStderrText:
		return c.getTextHandler(handlerLevelVar)
	case TargetStderrJSON:
		return c.getJSONHandler(handlerLevelVar)
	case TargetFileText:
		return c.getTextHandler(handlerLevelVar)
	case TargetFileJSON:
		return c.getJSONHandler(handlerLevelVar)
	case TargetSyslog:
		return c.getSyslogHandler(handlerLevelVar)
	case "":
		return c.getTextHandler(handlerLevelVar)
	}
	slog.Warn("unrecognized target option", "target", slog.StringValue(string(c.Target)))
	return c.getTextHandler(handlerLevelVar)
}

func (c *Config) getPlainHandler(levelVar *slog.LevelVar) (slog.Handler, *slog.LevelVar) {
	w := c.GetWriter()
	opts := &PlainHandlerOptions{
		HandlerOptions: slog.HandlerOptions{
			AddSource: c.AddSource,
			Level:     levelVar,
		},
		Color: c.Color,
	}
	return NewPlainHandler(w, opts), levelVar
}

func (c *Config) getTextHandler(levelVar *slog.LevelVar) (slog.Handler, *slog.LevelVar) {
	w := c.GetWriter()
	opts := &slog.HandlerOptions{
		AddSource: c.AddSource,
		Level:     levelVar,
	}
	return slog.NewTextHandler(w, opts), levelVar
}

func (c *Config) getJSONHandler(levelVar *slog.LevelVar) (slog.Handler, *slog.LevelVar) {
	w := c.GetWriter()
	opts := &slog.HandlerOptions{
		AddSource: c.AddSource,
		Level:     levelVar,
	}
	return slog.NewJSONHandler(w, opts), levelVar
}

func (c *Config) getSyslogHandler(levelVar *slog.LevelVar) (slog.Handler, *slog.LevelVar) {
	w := c.GetWriter()
	opts := &SyslogHandlerOptions{
		HandlerOptions: slog.HandlerOptions{
			AddSource: c.AddSource,
			Level:     levelVar,
		},
		Encoding: SyslogEncoding(c.SyslogEncoding),
		Facility: c.SyslogFacility,
	}
	return NewSyslogHandler(w, opts), levelVar
}

// GetLogger determines the [slog.Logger] defined by this configuration.
//
// This function simply wraps GetHandler into a [slog.New] call.
// Beside the logger this function also returns the [slog.LevelVar] instance
// assigned to it, enabling dynamic changing of the log level.
func (c *Config) GetLogger(levelVar *slog.LevelVar) (*slog.Logger, *slog.LevelVar) {
	h, l := c.GetHandler(levelVar)
	return slog.New(h), l
}

// Init sets up the default logger using the given parameters.
func Init(level slog.Level, target Target, color Color) {
	init := &Config{
		Level:  level.String(),
		Target: target,
		Color:  color,
	}
	logger, _ := init.GetLogger(nil)
	slog.SetDefault(logger)
}

// InitDefaults is equivalent to invoking
//
//	Init(slog.LevelInfo, log.TargetStdout, log.ColorAuto)
func InitDefault() {
	Init(defaultLevel, TargetStdout, ColorAuto)
}

// InitDebug is equivalent to invoking
//
//	Init(slog.LevelDebug, log.TargetStdout, log.ColorAuto)
func InitDebug() {
	Init(slog.LevelDebug, TargetStdout, ColorAuto)
}

// InitFromFlags sets the default logger to [TargetStdout] as well as the requested
// log level based on the command line flags.
//
// The log level is derived from the given command line arguments
// as well as the given command flag to log level map.
// If the command line arguments are nil, [os.Args] is used instead.
// If the map is nil, the following default mapping is used:
//
//	'-s', '--silent': slog.LevelError
//
//	'-q', '--quiet': slog.LevelWarn
//
//	'-v', '--verbose': slog.LevelInfo
//
//	'-d', '--debug' slog.LevelDebug
//
// If no command flag matches, the [slog.LevelInfo] is used.
func InitFromFlags(args []string, flags map[string]slog.Level) {
	initLevel := defaultLevel
	initTarget := TargetStdout
	initColor := ColorAuto
	initArgs := args
	if initArgs == nil {
		initArgs = os.Args
	}
	for _, arg := range initArgs {
		if flags != nil {
			level, ok := flags[arg]
			if !ok {
				continue
			}
			initLevel = level
		} else {
			switch arg {
			case "-s", "--silent":
				initLevel = slog.LevelError
			case "-q", "--quiet":
				initLevel = slog.LevelWarn
			case "-v", "--verbose":
				initLevel = slog.LevelInfo
			case "-d", "--debug":
				initLevel = slog.LevelDebug
			}
		}
	}
	Init(initLevel, initTarget, initColor)
}
