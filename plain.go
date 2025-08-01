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
	"os"
	"runtime"
	"slices"
	"strconv"
	"time"

	"github.com/mattn/go-isatty"
)

// PlainHandlerOptions is used to configure a [PlainHandler].
type PlainHandlerOptions struct {
	// Standard options
	slog.HandlerOptions
	// Color defines the color mode to use for logging
	Color Color
}

// PlainHandler provides a simple plain log format with optional
// support for color formatting.
type PlainHandler struct {
	w               io.Writer
	opts            PlainHandlerOptions
	ansi            bool
	prerenderdAttrs [][]byte
	groups          []string
}

// NewPlainHandler creates a new [PlainHandler] using the given writer
// and setup according to the given [PlainHandlerOptions].
func NewPlainHandler(w io.Writer, opts *PlainHandlerOptions) *PlainHandler {
	handlerOpts := opts
	if handlerOpts == nil {
		handlerOpts = &PlainHandlerOptions{}
	}
	ansi := false
	switch handlerOpts.Color {
	case ColorOn:
		ansi = true
	case ColorAuto:
		file, ok := w.(*os.File)
		ansi = ok && (isatty.IsTerminal(file.Fd()) || isatty.IsCygwinTerminal(file.Fd()))
	}
	return &PlainHandler{
		w:               w,
		opts:            *handlerOpts,
		ansi:            ansi,
		prerenderdAttrs: [][]byte{},
		groups:          noGroups,
	}
}

func (h *PlainHandler) Enabled(_ context.Context, level slog.Level) bool {
	handlerLevel := defaultLevel
	if h.opts.Level != nil {
		handlerLevel = h.opts.Level.Level()
	}
	return level >= handlerLevel
}

func (h *PlainHandler) Handle(_ context.Context, record slog.Record) error {
	builder := getMessageBuilder(h.groups)
	defer builder.Release()
	ansi := h.ansiEscapesForLevel(record.Level)
	if h.opts.ReplaceAttr == nil {
		if !record.Time.IsZero() {
			timeValue := record.Time.Round(0)
			builder.AppendString(ansi.defaultEscape)
			h.appendTime(builder, timeValue)
			builder.AppendRune(' ')
		}
		builder.AppendString(ansi.levelEscape)
		h.appendLevel(builder, record.Level)
		if h.opts.AddSource && record.PC != 0 {
			builder.AppendRune(' ')
			builder.AppendString(ansi.defaultEscape)
			h.appendSource(builder, h.sourceFromPC(record.PC))
		}
		builder.AppendRune(' ')
		builder.AppendString(ansi.messageEscape)
		builder.AppendString(record.Message)
	} else {
		if !record.Time.IsZero() {
			timeValue := record.Time.Round(0)
			h.handleAttr(noGroups, slog.Time(slog.TimeKey, timeValue), func(attr slog.Attr) {
				builder.AppendString(ansi.defaultEscape)
				h.appendAttr(builder, attr)
				builder.AppendRune(' ')
			})
		}
		h.handleAttr(noGroups, slog.Any(slog.LevelKey, record.Level), func(attr slog.Attr) {
			builder.AppendString(ansi.levelEscape)
			h.appendAttr(builder, attr)
		})
		if h.opts.AddSource && record.PC != 0 {
			h.handleAttr(noGroups, slog.Any(slog.SourceKey, h.sourceFromPC(record.PC)), func(attr slog.Attr) {
				builder.AppendRune(' ')
				builder.AppendString(ansi.defaultEscape)
				h.appendAttr(builder, attr)
			})
		}
		h.handleAttr(noGroups, slog.String(slog.MessageKey, record.Message), func(attr slog.Attr) {
			builder.AppendRune(' ')
			builder.AppendString(ansi.messageEscape)
			h.appendAttr(builder, attr)
		})
	}
	for _, prerenderedAttr := range h.prerenderdAttrs {
		builder.AppendBytes(prerenderedAttr)
	}
	record.Attrs(builder.Attrs(func(attr slog.Attr) bool {
		h.handleAttr(builder.Groups(), attr, func(attr slog.Attr) {
			builder.AppendRune(' ')
			builder.AppendString(ansi.tagEscape)
			builder.AppendString(builder.GroupPath())
			builder.AppendString(attr.Key)
			builder.AppendRune('=')
			builder.AppendString(ansi.defaultEscape)
			builder.AppendString(strconv.Quote(attr.Value.String()))
		})
		return true
	}))
	builder.AppendString(ansi.resetEscape)
	_, err := builder.Write(h.w, true)
	return err
}

func (h *PlainHandler) handleAttr(groups []string, attr slog.Attr, handle func(attr slog.Attr)) {
	attr.Value = attr.Value.Resolve()
	if h.opts.ReplaceAttr != nil {
		attr = h.opts.ReplaceAttr(groups, attr)
		if attr.Equal(emptyAttr) {
			return
		}
		attr.Value = attr.Value.Resolve()
	}
	handle(attr)
}

func (h *PlainHandler) appendTime(builder *messageBuilder, t time.Time) {
	s := t.Truncate(time.Millisecond).Add(time.Millisecond / 10).Format(time.RFC3339Nano)
	builder.AppendString(s[:23])
	builder.AppendString(s[24:])
}

func (h *PlainHandler) appendLevel(builder *messageBuilder, level slog.Level) {
	s := "NOTICE"
	if level != LevelNotice {
		s = level.String()
	}
	slen := len(s)
	switch slen {
	case 6:
		builder.AppendString(s)
		builder.AppendRune(' ')
	case 7:
		builder.AppendString(s)
	default:
		{
			padding := 7 - slen
			builder.AppendString(s)
			builder.AppendString("       "[:padding])
		}
	}
}

func (h *PlainHandler) appendSource(builder *messageBuilder, source *slog.Source) {
	const filler = "........................................"
	s := source.File + ":" + strconv.Itoa(source.Line)
	slen := len(s)
	padding := len(filler) - 3 - slen
	if padding >= 0 {
		builder.AppendString(filler[:padding+3])
		builder.AppendString(s)
	} else {
		builder.AppendString(filler[:3])
		builder.AppendString(s[-padding:])
	}
}

func (h *PlainHandler) appendAttr(builder *messageBuilder, attr slog.Attr) {
	v := attr.Value
	switch v.Kind() {
	case slog.KindTime:
		h.appendTime(builder, v.Time())
	case slog.KindAny:
		if level, ok := v.Any().(slog.Level); ok {
			h.appendLevel(builder, level)
		} else if source, ok := v.Any().(*slog.Source); ok {
			h.appendSource(builder, source)
		} else {
			builder.AppendString(attr.Value.String())
		}
	default:
		builder.AppendString(attr.Value.String())
	}
}

func (h *PlainHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	builder := getMessageBuilder(h.groups)
	defer builder.Release()
	ansi := h.ansiEscapesForLevel(defaultLevel)
	appendAttr := builder.Attrs(func(attr slog.Attr) bool {
		h.handleAttr(builder.Groups(), attr, func(attr slog.Attr) {
			builder.AppendRune(' ')
			builder.AppendString(ansi.tagEscape)
			builder.AppendString(builder.GroupPath())
			builder.AppendString(attr.Key)
			builder.AppendRune('=')
			builder.AppendString(ansi.defaultEscape)
			builder.AppendString(strconv.Quote(attr.Value.String()))
		})
		return true
	})
	for _, attr := range attrs {
		appendAttr(attr)
	}
	h2 := h.clone()
	h2.prerenderdAttrs = append(h2.prerenderdAttrs, slices.Clone(builder.Bytes()))
	return h2
}

func (h *PlainHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := h.clone()
	h2.groups = append(h2.groups, name)
	return h2
}

func (h *PlainHandler) clone() *PlainHandler {
	return &PlainHandler{
		w:               h.w,
		opts:            h.opts,
		ansi:            h.ansi,
		prerenderdAttrs: slices.Clip(h.prerenderdAttrs),
		groups:          slices.Clip(h.groups),
	}
}

func (h *PlainHandler) sourceFromPC(pc uintptr) *slog.Source {
	frames := runtime.CallersFrames([]uintptr{pc})
	frame, _ := frames.Next()
	return &slog.Source{
		Function: frame.Function,
		File:     frame.File,
		Line:     frame.Line,
	}
}

func (h *PlainHandler) ansiEscapesForLevel(level slog.Level) *ansiEscapes {
	if !h.ansi {
		return noAnsi
	}
	var levelEscape, messageEscape string
	switch {
	case level < slog.LevelInfo:
		levelEscape, messageEscape = ansiDefault, ansiDefault
	case level < slog.LevelWarn:
		levelEscape, messageEscape = ansiInfo, ansiHighlight
	case level < slog.LevelError:
		levelEscape, messageEscape = ansiWarn, ansiHighlight
	case level == LevelNotice:
		levelEscape, messageEscape = ansiNotice, ansiHighlight
	default:
		levelEscape, messageEscape = ansiError, ansiHighlight
	}
	return &ansiEscapes{
		resetEscape:     ansiReset,
		defaultEscape:   ansiDefault,
		highlightEscape: ansiHighlight,
		warnEscape:      ansiWarn,
		errorEscape:     ansiError,
		tagEscape:       ansiTag,
		levelEscape:     levelEscape,
		messageEscape:   messageEscape,
	}
}

const ansiReset = "\x1b[0m"
const ansiDefault = "\x1b[37m"
const ansiHighlight = "\x1b[97m"
const ansiInfo = "\x1b[32m"
const ansiWarn = "\x1b[33m"
const ansiError = "\x1b[31m"
const ansiNotice = "\x1b[97m"
const ansiTag = "\x1b[36m"

type ansiEscapes struct {
	resetEscape     string
	defaultEscape   string
	highlightEscape string
	warnEscape      string
	errorEscape     string
	tagEscape       string
	levelEscape     string
	messageEscape   string
}

var noAnsi = &ansiEscapes{}
