//
// Copyright (C) 2023-2025 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

package log

import (
	"context"
	"io"
	"log/slog"
)

// RawHandler emits raw log messages (without considering any of acompanying attributes)
// directly to the underlying [io.Writer].
type RawHandler struct {
	w io.Writer
}

// NewRawHandler creates a new [RawHandler] wrapping the given writer.
func NewRawHandler(w io.Writer) *RawHandler {
	return &RawHandler{w: w}
}

func (h *RawHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *RawHandler) Handle(_ context.Context, record slog.Record) error {
	builder := getMessageBuilder(nil)
	defer builder.Release()
	builder.AppendString(record.Message)
	_, err := builder.Write(h.w, true)
	return err
}

func (h *RawHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *RawHandler) WithGroup(_ string) slog.Handler {
	return h
}
