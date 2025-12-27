//
// Copyright (C) 2023-2025 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

package log

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tdrn-org/go-conf"
	"github.com/tdrn-org/go-tlsconf/tlsclient"
)

// SyslogKey identifies the [slog.Attr] containing the syslog MsgID attribute.
const SyslogKey string = "syslog"

// SyslogEncoding defines the supported syslog formats.
type SyslogEncoding string

const (
	// SyslogEncodingDefault sets the format to RFC5424 + octet framing
	SyslogEncodingDefault SyslogEncoding = ""
	// SyslogEncodingRFC3164 sets the format to RFC3164 + implicit framing (https://datatracker.ietf.org/doc/html/rfc3164)
	SyslogEncodingRFC3164 SyslogEncoding = "rfc3164"
	// SyslogEncodingRFC3164 sets the format to RFC3164 + octet framing (https://datatracker.ietf.org/doc/html/rfc3164)
	SyslogEncodingRFC3164F SyslogEncoding = "rfc3164+framing"
	// SyslogEncodingRFC3164 sets the format to RFC5424 + implicit framing (https://datatracker.ietf.org/doc/html/rfc5224)
	SyslogEncodingRFC5424 SyslogEncoding = "rfc5424"
	// SyslogEncodingRFC3164 sets the format to RFC5424 + octet framing (https://datatracker.ietf.org/doc/html/rfc5224)
	SyslogEncodingRFC5424F SyslogEncoding = "rfc5424+framing"
)

const defaultSyslogFacility int = 16 // local0

// SyslogHandlerOptions is used to configure a [SyslogHandler].
type SyslogHandlerOptions struct {
	// Standard options
	slog.HandlerOptions
	// Encoding defines the syslog format to use
	Encoding SyslogEncoding
	// Facility defines the syslog facility to use
	// (see https://datatracker.ietf.org/doc/html/rfc5424#section-6.2.1)
	Facility int
	// AppName defines the application or service name to use
	// (see https://datatracker.ietf.org/doc/html/rfc5424#section-6.2.5)
	AppName string
}

// SyslogHandler is used to emit log messages to a syslog server.
type SyslogHandler struct {
	w               io.Writer
	opts            SyslogHandlerOptions
	encode          func(*messageBuilder, context.Context, slog.Record)
	header          string
	msgID           string
	prerenderdAttrs [][]byte
	groups          []string
}

// NewSyslogHandler creates a new [SyslogHandler] using the given writer
// and setup according to the given [SyslogHandlerOptions].
func NewSyslogHandler(w io.Writer, opts *SyslogHandlerOptions) *SyslogHandler {
	handlerOpts := opts
	if handlerOpts == nil {
		handlerOpts = &SyslogHandlerOptions{
			Encoding: SyslogEncodingRFC5424F,
			Facility: defaultSyslogFacility,
		}
	}
	facility := handlerOpts.Facility
	if facility < 0 || 23 < facility {
		slog.Warn("out-of-range facility value; using default", slog.Int("facility", facility))
		facility = defaultSyslogFacility
	}
	handler := &SyslogHandler{
		w:     w,
		opts:  *handlerOpts,
		msgID: "-",
	}
	return handler.initHeader().initEncoder(false)
}

func (h *SyslogHandler) initHeader() *SyslogHandler {
	host := syslogHostname()
	appName := syslogAppName(h.opts.AppName)
	procID := syslogProcID()
	switch h.opts.Encoding {
	case SyslogEncodingRFC3164, SyslogEncodingRFC3164F:
		h.header = fmt.Sprintf(" %s %s[%s]: ", host, appName, procID)
	default:
		h.header = fmt.Sprintf(" %s %s %s ", host, appName, procID)
	}
	return h
}

func (h *SyslogHandler) Enabled(_ context.Context, level slog.Level) bool {
	handlerLevel := defaultLevel
	if h.opts.Level != nil {
		handlerLevel = h.opts.Level.Level()
	}
	return level >= handlerLevel
}

func (h *SyslogHandler) Handle(ctx context.Context, record slog.Record) error {
	builder := getMessageBuilder(h.groups)
	defer builder.Release()
	h.encode(builder, ctx, record)
	implicitFraming := h.opts.Encoding != SyslogEncodingRFC3164F && h.opts.Encoding != SyslogEncodingRFC5424F
	_, err := builder.Write(h.w, implicitFraming)
	return err
}

func (h *SyslogHandler) encodeRFC3164(builder *messageBuilder, _ context.Context, record slog.Record) {
	h.appendPRI(builder, record.Level)
	recordTime := record.Time
	if recordTime.IsZero() {
		recordTime = time.Now()
	}
	if h.opts.ReplaceAttr == nil {
		h.appendTime(builder, recordTime)
		builder.AppendString(h.header)
		builder.AppendString(record.Message)
	} else {
		timeValue := record.Time.Round(0)
		h.handleAttr(noGroups, slog.Time(slog.TimeKey, timeValue), func(attr slog.Attr) {
			h.appendAttr(builder, attr)
			builder.AppendRune(' ')
		})
		builder.AppendString(h.header)
		h.handleAttr(noGroups, slog.String(slog.MessageKey, record.Message), func(attr slog.Attr) {
			h.appendAttr(builder, attr)
		})
	}
	for _, prerenderedAttr := range h.prerenderdAttrs {
		builder.AppendBytes(prerenderedAttr)
	}
	record.Attrs(builder.Attrs(func(attr slog.Attr) bool {
		h.handleAttr(builder.Groups(), attr, func(attr slog.Attr) {
			if attr.Key != SyslogKey {
				builder.AppendRune(' ')
				builder.AppendString(builder.GroupPath())
				builder.AppendString(attr.Key)
				builder.AppendString("=")
				builder.AppendString(strconv.Quote(attr.Value.String()))
			}
		})
		return true
	}))
}

func (h *SyslogHandler) encodeRFC5424(builder *messageBuilder, _ context.Context, record slog.Record) {
	h.appendPRI(builder, record.Level)
	builder.AppendString("1 ")
	recordTime := record.Time
	if recordTime.IsZero() {
		recordTime = time.Now()
	}
	if h.opts.ReplaceAttr == nil {
		h.appendTime(builder, recordTime)
	} else {
		timeValue := record.Time.Round(0)
		h.handleAttr(noGroups, slog.Time(slog.TimeKey, timeValue), func(attr slog.Attr) {
			h.appendAttr(builder, attr)
			builder.AppendRune(' ')
		})
	}
	builder.AppendString(h.header)
	h.appendMsgID(builder, &record)
	builder.AppendConditional(" [Attrs@1")
	for _, prerenderedAttr := range h.prerenderdAttrs {
		builder.AppendBytes(prerenderedAttr)
	}
	record.Attrs(builder.Attrs(func(attr slog.Attr) bool {
		h.handleAttr(builder.Groups(), attr, func(attr slog.Attr) {
			if attr.Key != SyslogKey {
				builder.AppendRune(' ')
				builder.AppendString(builder.GroupPath())
				builder.AppendString(attr.Key)
				builder.AppendString("=")
				builder.AppendString(strconv.Quote(attr.Value.String()))
			}
		})
		return true
	}))
	builder.CompleteConditional("] ", " - ")
	if h.opts.ReplaceAttr == nil {
		builder.AppendString(record.Message)
	} else {
		h.handleAttr(noGroups, slog.String(slog.MessageKey, record.Message), func(attr slog.Attr) {
			h.appendAttr(builder, attr)
		})
	}
}

func (h *SyslogHandler) appendPRI(builder *messageBuilder, level slog.Level) {
	builder.AppendRune('<')
	pri := h.opts.Facility << 3
	switch {
	case level == LevelNotice:
		pri = pri | 5
	case level >= slog.LevelError:
		pri = pri | 3
	case level <= slog.LevelWarn:
		pri = pri | 4
	case level <= slog.LevelInfo:
		pri = pri | 6
	default:
		pri = pri | 7
	}
	builder.AppendString(strconv.Itoa(pri))
	builder.AppendRune('>')
}

func (h *SyslogHandler) appendTime(builder *messageBuilder, t time.Time) {
	var layout string
	switch h.opts.Encoding {
	case SyslogEncodingRFC3164, SyslogEncodingRFC3164F:
		layout = time.Stamp
	default:
		layout = time.RFC3339
	}
	builder.AppendString(t.Format(layout))
}

func (h *SyslogHandler) appendMsgID(builder *messageBuilder, record *slog.Record) {
	msgID := h.msgID
	record.Attrs(func(attr slog.Attr) bool {
		if attr.Key == SyslogKey {
			msgID = attr.Value.String()
			return false
		}
		return true
	})
	builder.AppendString(msgID)
}

func (h *SyslogHandler) handleAttr(groups []string, attr slog.Attr, handle func(attr slog.Attr)) {
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

func (h *SyslogHandler) appendAttr(builder *messageBuilder, attr slog.Attr) {
	v := attr.Value
	switch v.Kind() {
	case slog.KindTime:
		h.appendTime(builder, v.Time())
	default:
		builder.AppendString(attr.Value.String())
	}
}

func (h *SyslogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	builder := getMessageBuilder(h.groups)
	defer builder.Release()
	msgID := ""
	appendAttr := builder.Attrs(func(attr slog.Attr) bool {
		h.handleAttr(builder.Groups(), attr, func(attr slog.Attr) {
			if attr.Key != SyslogKey {
				builder.AppendRune(' ')
				builder.AppendString(builder.GroupPath())
				builder.AppendString(attr.Key)
				builder.AppendString("=")
				builder.AppendString(strconv.Quote(attr.Value.String()))
			} else {
				msgID = attr.Value.String()
			}
		})
		return true
	})
	for _, attr := range attrs {
		appendAttr(attr)
	}
	h2 := h.clone()
	h2.prerenderdAttrs = append(h2.prerenderdAttrs, slices.Clone(builder.Bytes()))
	if msgID != "" {
		h2.msgID = msgID
	}
	return h2
}

func (h *SyslogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := h.clone()
	h2.groups = append(h2.groups, name)
	return h2
}

func (h *SyslogHandler) clone() *SyslogHandler {
	return (&SyslogHandler{
		w:               h.w,
		opts:            h.opts,
		header:          h.header,
		msgID:           h.msgID,
		prerenderdAttrs: slices.Clip(h.prerenderdAttrs),
		groups:          slices.Clip(h.groups),
	}).initEncoder(true)
}

func (h *SyslogHandler) initEncoder(quiet bool) *SyslogHandler {
	switch h.opts.Encoding {
	case SyslogEncodingDefault:
		h.opts.Encoding = SyslogEncodingRFC5424F
		h.encode = h.encodeRFC5424
	case SyslogEncodingRFC3164, SyslogEncodingRFC3164F:
		h.encode = h.encodeRFC3164
	case SyslogEncodingRFC5424, SyslogEncodingRFC5424F:
		h.encode = h.encodeRFC5424
	default:
		if !quiet {
			slog.Warn("unrecognized syslog encoding; using default", slog.String("encoding", string(h.opts.Encoding)))
		}
		h.encode = h.encodeRFC5424
	}
	return h
}

func syslogHostname() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "-"
	}
	return host
}

func syslogAppName(optsAppName string) string {
	appName := strings.TrimSpace(optsAppName)
	if appName != "" {
		return appName
	}
	appName = filepath.Base(os.Args[0])
	return appName
}

func syslogProcID() string {
	procID := os.Getpid()
	return strconv.Itoa(procID)
}

type syslogWriter struct {
	network string
	address string
	mutex   sync.Mutex
	conn    net.Conn
}

func (w *syslogWriter) Write(b []byte) (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if w.conn == nil {
		conn, err := w.dial()
		if err != nil {
			return 0, err
		}
		w.conn = conn
	}
	written, err := w.conn.Write(b)
	if err != nil {
		w.conn.Close()
		w.conn = nil
	}
	return written, err
}

func (w *syslogWriter) Close() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if w.conn == nil {
		return nil
	}
	err := w.conn.Close()
	w.conn = nil
	return err
}

func (w *syslogWriter) dial() (net.Conn, error) {
	switch w.network {
	case "":
		return net.Dial("tcp", w.address)
	case "udp", "udp4", "udp6", "tcp", "tcp4", "tcp6":
		return net.Dial(w.network, w.address)
	case "tcp+tls", "tcp4+tls", "tcp6+tls":
		return tls.Dial(w.network[:(len(w.network)-len("+tls"))], w.address, w.tlsConfig())
	}
	return nil, fmt.Errorf("unrecognized syslog network options: '%s'", w.network)
}

func (w *syslogWriter) tlsConfig() *tls.Config {
	tlsClientConfig, _ := conf.LookupConfiguration[*tlsclient.Config]()
	return &tlsClientConfig.Config
}
