// syslog.go
//
// Copyright (C) 2023 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

// Package syslog provides syslog logging related functionality.
package syslog

import (
	"io"
	"log/syslog"

	"github.com/rs/zerolog"
)

type YAMLSyslogConfig struct {
	EnabledOption bool `yaml:"enabled"`
	CEEOption     bool `yaml:"cee"`
}

func (config *YAMLSyslogConfig) NewWriter() io.Writer {
	if config.CEEOption {
		return zerolog.SyslogCEEWriter(&syslog.Writer{})
	}
	return zerolog.SyslogLevelWriter(&syslog.Writer{})
}
