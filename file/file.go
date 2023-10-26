// file.go
//
// Copyright (C) 2023 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

// Package file provides file logging related functionality.
package file

import (
	"io"

	"gopkg.in/natefinch/lumberjack.v2"
)

type YAMLFileConfig struct {
	EnabledOption    bool   `yaml:"enabled"`
	FilenameOption   string `yaml:"filename"`
	MaxSizeOption    int    `yaml:"max_size"`
	MaxAgeOption     int    `yaml:"max_age"`
	MaxBackupsOption int    `yaml:"max_backups"`
	CompressOption   bool   `yaml:"compress"`
}

func (config *YAMLFileConfig) NewWriter() io.Writer {
	if !config.EnabledOption {
		return nil
	}
	return &lumberjack.Logger{
		Filename:   config.filenameOption(),
		MaxSize:    config.maxSizeOption(),
		MaxAge:     config.maxAgeOption(),
		MaxBackups: config.maxBackupsOption(),
		Compress:   config.compressOption(),
	}
}

func (config *YAMLFileConfig) filenameOption() string {
	return config.FilenameOption
}

func (config *YAMLFileConfig) maxSizeOption() int {
	if config.MaxSizeOption < 0 {
		return 0
	}
	return config.MaxSizeOption
}

func (config *YAMLFileConfig) maxAgeOption() int {
	if config.MaxAgeOption < 0 {
		return 0
	}
	return config.MaxAgeOption
}

func (config *YAMLFileConfig) maxBackupsOption() int {
	if config.MaxBackupsOption < 0 {
		return 0
	}
	return config.MaxBackupsOption
}

func (config *YAMLFileConfig) compressOption() bool {
	return config.CompressOption
}
