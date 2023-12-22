// console.go
//
// Copyright (C) 2023 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

// Package console provides console logging related functionality.
package console

import (
	"io"
	"os"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
)

// Console color mode
type Color int

const (
	// Auto-detect coloring
	ColorAuto Color = -1
	// Force coloring off
	ColorOff Color = 0
	// Force coloring on
	ColorOn Color = 1
)

// NewWriter creates a new [io.Writer] with default options for console logging.
func NewDefaultWriter() io.Writer {
	return NewWriter(os.Stderr, ColorOff, time.RFC3339)
}

// NewWriter creates a new [io.Writer] for console logging.
func NewWriter(out *os.File, color Color, timeFormat string) io.Writer {
	return &zerolog.ConsoleWriter{
		Out:        out,
		NoColor:    !colorFlag(out, color),
		TimeFormat: timeFormat,
	}
}

func colorFlag(out *os.File, color Color) bool {
	switch color {
	case ColorAuto:
		return isatty.IsTerminal(out.Fd()) || isatty.IsCygwinTerminal(out.Fd())
	case ColorOff:
		return false
	case ColorOn:
		return true
	}
	return false
}

type YAMLConsoleConfig struct {
	EnabledOption    bool   `yaml:"enabled"`
	OutOption        string `yaml:"out"`
	ColorOption      string `yaml:"color"`
	TimeFormatOption string `yaml:"timeformat"`
}

func (config *YAMLConsoleConfig) NewWriter() io.Writer {
	if !config.EnabledOption {
		return nil
	}
	return NewWriter(config.outOption(), config.colorOption(), config.timeFormatOption())
}

func (config *YAMLConsoleConfig) outOption() *os.File {
	switch config.OutOption {
	case "stdout":
		return os.Stdout
	case "stderr":
		return os.Stderr
	}
	return os.Stderr
}

func (config *YAMLConsoleConfig) colorOption() Color {
	switch config.ColorOption {
	case "auto":
		return ColorAuto
	case "off":
		return ColorOff
	case "on":
		return ColorOn
	}
	return ColorOff
}

func (config *YAMLConsoleConfig) timeFormatOption() string {
	return config.TimeFormatOption
}
