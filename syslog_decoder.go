//
// Copyright (C) 2023-2025 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

package log

import (
	"bytes"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

const syslogDecodeLimit int = 0x7fff

// SyslogMessage defines the common interface for all kinds of syslog messages.
type SyslogMessage interface {
	// Len gets the length of the syslog message.
	Len() int
	// Raw gets the raw bytes of the syslog message.
	Raw() []byte
	// String gets the syslog message as a quoted string (using [strconv.Quote]).
	String() string
}

// UndecodedSyslogMessage represents a raw, but fully received syslog message.
type UndecodedSyslogMessage struct {
	raw []byte
}

// Len gets the length of the syslog message.
func (m *UndecodedSyslogMessage) Len() int {
	return len(m.raw)
}

// Raw gets the raw bytes of the syslog message.
func (m *UndecodedSyslogMessage) Raw() []byte {
	return m.raw
}

// String gets the syslog message as a quoted string (using [strconv.Quote]).
func (m *UndecodedSyslogMessage) String() string {
	return strconv.Quote(string(m.raw))
}

func (m *UndecodedSyslogMessage) append(b ...byte) {
	m.raw = append(m.raw, b...)
}

func (m *UndecodedSyslogMessage) decode(off int) SyslogMessage {
	decodeOff := m.decodeByte(off, '<')
	if decodeOff < 0 {
		return m
	}
	decodeOff, facility, priority := m.decodePRI(decodeOff)
	if decodeOff < 0 {
		return m
	}
	decodeOff = m.decodeByte(decodeOff, '>')
	if decodeOff < 0 {
		return m
	}
	var decoded SyslogMessage
	switch m.raw[decodeOff] {
	case 'J', 'F', 'M', 'A', 'S', 'O', 'N', 'D':
		decoded = m.decodeRFC3164(decodeOff, facility, priority)
	case '1':
		decoded = m.decodeRFC5424(decodeOff, facility, priority)
	default:
		decoded = m
	}
	return decoded
}

func (m *UndecodedSyslogMessage) decodeByte(off int, b byte) int {
	if off >= len(m.raw) || m.raw[off] != b {
		return -1
	}
	return off + 1
}

func (m *UndecodedSyslogMessage) decodePRI(off int) (int, int, int) {
	decodeOff := off
	pri := 0
	for {
		if decodeOff >= len(m.raw) {
			return -1, 0, 0
		}
		switch m.raw[decodeOff] {
		case '0':
			pri *= 10
		case '1':
			pri = 10*pri + 1
		case '2':
			pri = 10*pri + 2
		case '3':
			pri = 10*pri + 3
		case '4':
			pri = 10*pri + 4
		case '5':
			pri = 10*pri + 5
		case '6':
			pri = 10*pri + 6
		case '7':
			pri = 10*pri + 7
		case '8':
			pri = 10*pri + 8
		case '9':
			pri = 10*pri + 9
		default:
			if decodeOff == off {
				return -1, 0, 0
			}
			return decodeOff, (pri >> 3), (pri & 0x7)
		}
		decodeOff++
	}
}

func (m *UndecodedSyslogMessage) decodeTimestamp(off int, layout string, layoutLen int) (int, time.Time) {
	var decodeOff int
	var timestampString string
	if layoutLen > 0 {
		decodeOff = off + layoutLen
		if decodeOff >= len(m.raw) {
			return -1, time.Time{}
		}
		timestampString = string(m.raw[off:decodeOff])
	} else {
		decodeOff, timestampString = m.decodeString(off)
		if decodeOff < 0 {
			return -1, time.Time{}
		}
	}
	timestamp, err := time.Parse(layout, timestampString)
	if err != nil {
		return -1, time.Time{}
	}
	return decodeOff, timestamp
}

func (m *UndecodedSyslogMessage) decodeString(off int) (int, string) {
	if off >= len(m.raw) {
		return -1, ""
	}
	decodeOff := off
	for {
		switch m.raw[decodeOff] {
		case ' ', '\n':
			return decodeOff, string(m.raw[off:decodeOff])
		default:
			decodeOff++
		}
		if decodeOff >= len(m.raw) {
			return decodeOff, string(m.raw[off:])
		}
	}
}

func (m *UndecodedSyslogMessage) decodeTrailer(off int) (int, string) {
	if off >= len(m.raw) {
		return -1, ""
	}
	decodeOff := off
	for {
		if m.raw[decodeOff] == '\n' {
			return decodeOff + 1, string(m.raw[off:decodeOff])
		}
		decodeOff++
		if decodeOff >= len(m.raw) {
			return decodeOff, string(m.raw[off:])
		}
	}
}

func (m *UndecodedSyslogMessage) decodeSD(off int) (int, []SyslogSDElement) {
	if off >= len(m.raw) {
		return -1, nil
	}
	elements := make([]SyslogSDElement, 0)
	if m.raw[off] == '-' {
		return off + 1, elements
	}
	decodeOff := off
	for {
		element := &SyslogSDElement{}
		decodeOff = m.decodeSDElement(decodeOff, element)
		if decodeOff < 0 {
			return -1, nil
		}
		elements = append(elements, *element)
		if decodeOff >= len(m.raw) || m.raw[decodeOff] != '[' {
			break
		}
	}
	return decodeOff, elements
}

func (m *UndecodedSyslogMessage) decodeSDElement(off int, element *SyslogSDElement) int {
	decodeOff := m.decodeByte(off, '[')
	if decodeOff < 0 {
		return -1
	}
	decodeOff, id := m.decodeString(decodeOff)
	if decodeOff < 0 {
		return -1
	}
	for {
		if decodeOff >= len(m.raw) {
			return -1
		}
		if m.raw[decodeOff] == ']' {
			decodeOff++
			break
		}
		decodeOff = m.decodeByte(decodeOff, ' ')
		if decodeOff < 0 {
			return -1
		}
		param := &SyslogSDParam{}
		decodeOff = m.decodeSDParam(decodeOff, param)
		if decodeOff < 0 {
			return -1
		}
		element.Params = append(element.Params, *param)
	}
	element.ID = id
	return decodeOff
}

func (m *UndecodedSyslogMessage) decodeSDParam(off int, param *SyslogSDParam) int {
	decodeOff, key := m.decodeSDParamKey(off)
	if decodeOff < 0 {
		return -1
	}
	decodeOff = m.decodeByte(decodeOff, '=')
	if decodeOff < 0 {
		return -1
	}
	decodeOff = m.decodeByte(decodeOff, '"')
	if decodeOff < 0 {
		return -1
	}
	decodeOff, value := m.decodeSDParamValue(decodeOff)
	if decodeOff < 0 {
		return -1
	}
	decodeOff = m.decodeByte(decodeOff, '"')
	if decodeOff < 0 {
		return -1
	}
	param.Key = key
	param.Value = value
	return decodeOff
}

func (m *UndecodedSyslogMessage) decodeSDParamKey(off int) (int, string) {
	decodeOff := off
	key := strings.Builder{}
	for {
		if decodeOff >= len(m.raw) {
			return -1, ""
		}
		nextByte := m.raw[decodeOff]
		if nextByte == '=' {
			return decodeOff, key.String()
		}
		key.WriteByte(nextByte)
		decodeOff++
	}
}

func (m *UndecodedSyslogMessage) decodeSDParamValue(off int) (int, string) {
	decodeOff := off
	escaped := false
	value := strings.Builder{}
	for {
		if decodeOff >= len(m.raw) {
			return -1, ""
		}
		nextByte := m.raw[decodeOff]
		if !escaped {
			switch nextByte {
			case '"':
				return decodeOff, value.String()
			case '\\':
				escaped = true
			default:
				value.WriteByte(nextByte)
			}
			decodeOff++
		} else {
			escaped = false
			value.WriteByte(nextByte)
			decodeOff++
		}
	}
}

func (m *UndecodedSyslogMessage) decodeRFC3164(off, facility, priority int) SyslogMessage {
	decodeOff, timestamp := m.decodeTimestamp(off, time.Stamp, len(time.Stamp))
	if decodeOff < 0 {
		return m
	}
	decodeOff = m.decodeByte(decodeOff, ' ')
	if decodeOff < 0 {
		return m
	}
	decodeOff, hostname := m.decodeString(decodeOff)
	if decodeOff < 0 {
		return m
	}
	decodeOff = m.decodeByte(decodeOff, ' ')
	if decodeOff < 0 {
		return m
	}
	decodeOff, messageTag := m.decodeString(decodeOff)
	if decodeOff < 0 {
		return m
	}
	decodeOff = m.decodeByte(decodeOff, ' ')
	if decodeOff < 0 {
		return m
	}
	decodeOff, messageContent := m.decodeTrailer(decodeOff)
	if decodeOff < 0 {
		return m
	}
	return &RFC3164SyslogMessage{
		UndecodedSyslogMessage: UndecodedSyslogMessage{
			raw: m.raw,
		},
		Facility:       facility,
		Priority:       priority,
		Timestamp:      timestamp,
		Hostname:       hostname,
		MessageTag:     messageTag,
		MessageContent: messageContent,
	}
}

func (m *UndecodedSyslogMessage) decodeRFC5424(off, facility, priority int) SyslogMessage {
	decodeOff := m.decodeByte(off, '1')
	if decodeOff < 0 {
		return m
	}
	decodeOff = m.decodeByte(decodeOff, ' ')
	if decodeOff < 0 {
		return m
	}
	decodeOff, timestamp := m.decodeTimestamp(decodeOff, time.RFC3339Nano, -1)
	if decodeOff < 0 {
		return m
	}
	decodeOff = m.decodeByte(decodeOff, ' ')
	if decodeOff < 0 {
		return m
	}
	decodeOff, hostname := m.decodeString(decodeOff)
	if decodeOff < 0 {
		return m
	}
	decodeOff = m.decodeByte(decodeOff, ' ')
	if decodeOff < 0 {
		return m
	}
	decodeOff, appName := m.decodeString(decodeOff)
	if decodeOff < 0 {
		return m
	}
	decodeOff = m.decodeByte(decodeOff, ' ')
	if decodeOff < 0 {
		return m
	}
	decodeOff, procID := m.decodeString(decodeOff)
	if decodeOff < 0 {
		return m
	}
	decodeOff = m.decodeByte(decodeOff, ' ')
	if decodeOff < 0 {
		return m
	}
	decodeOff, msgID := m.decodeString(decodeOff)
	if decodeOff < 0 {
		return m
	}
	decodeOff = m.decodeByte(decodeOff, ' ')
	if decodeOff < 0 {
		return m
	}
	decodeOff, sd := m.decodeSD(decodeOff)
	if decodeOff < 0 {
		return m
	}
	decodeOff, msg := m.decodeTrailer(decodeOff)
	if decodeOff < 0 {
		return m
	}
	return &RFC5424SyslogMessage{
		UndecodedSyslogMessage: UndecodedSyslogMessage{
			raw: m.raw,
		},
		Facility:  facility,
		Priority:  priority,
		Version:   1,
		Timestamp: timestamp,
		Hostname:  hostname,
		AppName:   appName,
		ProcID:    procID,
		MsgID:     msgID,
		SD:        sd,
		Msg:       msg,
	}
}

// RFC3164SyslogMessage represents a RFC3164 encoded syslog message
// (see https://datatracker.ietf.org/doc/html/rfc3164).
type RFC3164SyslogMessage struct {
	UndecodedSyslogMessage
	Facility       int
	Priority       int
	Timestamp      time.Time
	Hostname       string
	MessageTag     string
	MessageContent string
}

// RFC5424SyslogMessage represents a RFC5424 encoded syslog message
// (see https://datatracker.ietf.org/doc/html/rfc5424).
type RFC5424SyslogMessage struct {
	UndecodedSyslogMessage
	Facility  int
	Priority  int
	Version   int
	Timestamp time.Time
	Hostname  string
	AppName   string
	ProcID    string
	MsgID     string
	SD        []SyslogSDElement
	Msg       string
}

type SyslogSDElement struct {
	ID     string
	Params []SyslogSDParam
}

type SyslogSDParam struct {
	Key   string
	Value string
}

type syslogDecoderState int

const (
	syslogDecodeFraming                syslogDecoderState = 0
	syslogDecodeImplicitFraming        syslogDecoderState = 1
	syslogDecodeImplicitFramingMessage syslogDecoderState = 2
	syslogDecodeOctetFramingHeader     syslogDecoderState = 3
	syslogDecodeOctetFraming           syslogDecoderState = 4
	syslogDecodeOctetFramingMessage    syslogDecoderState = 5
	syslogDecodeUnknown                syslogDecoderState = 6
	syslogDecodeUnknownMessage         syslogDecoderState = 7
)

// SyslogDecoder decodes syslog messages from a byte stream.
type SyslogDecoder struct {
	buffer          bytes.Buffer
	state           syslogDecoderState
	decoding        *UndecodedSyslogMessage
	octets          int
	octetsRemaining int
}

// Reset reverts the state of the decode without releasing any already
// allocated buffer.
func (d *SyslogDecoder) Reset() {
	d.buffer.Reset()
	d.state = syslogDecodeFraming
	d.decoding = nil
	d.octets = 0
	d.octetsRemaining = 0
}

// Feed adds the given bytes to the decode buffer.
func (d *SyslogDecoder) Feed(b []byte) {
	_, _ = d.buffer.Write(b)
}

// Read reads bytes from the given [io.Reader] and feeds them
// into the decode buffer.
func (d *SyslogDecoder) Read(r io.Reader) error {
	d.buffer.Grow(1024)
	availableBuffer := d.buffer.AvailableBuffer()
	availableBuffer = availableBuffer[:cap(availableBuffer)]
	read, err := r.Read(availableBuffer)
	if err != nil {
		return err
	}
	d.buffer.Write(availableBuffer[:read])
	return nil
}

// ReadFrom reads bytes from the given [net.PacketConn] and feeds them
// into the decode buffer.
func (d *SyslogDecoder) ReadFrom(c net.PacketConn) error {
	d.buffer.Grow(1024)
	availableBuffer := d.buffer.AvailableBuffer()
	availableBuffer = availableBuffer[:cap(availableBuffer)]
	read, _, err := c.ReadFrom(availableBuffer)
	if err != nil {
		return err
	}
	d.buffer.Write(availableBuffer[:read])
	return nil
}

// Decode decodes all fully received syslog messages
// from the decode buffer.
func (d *SyslogDecoder) Decode() []SyslogMessage {
	messages := make([]SyslogMessage, 0)
	for {
		message := d.decode()
		if message == nil {
			break
		}
		messages = append(messages, message)
	}
	return messages
}

func (d *SyslogDecoder) decode() SyslogMessage {
	for {
		oldState := d.state
		switch d.state {
		case syslogDecodeFraming:
			d.decodeFraming()
		case syslogDecodeImplicitFraming:
			d.decodeImplicitFraming()
		case syslogDecodeImplicitFramingMessage:
			return d.decodeImplicitFramingMessage()
		case syslogDecodeOctetFramingHeader:
			d.decodeOctetFramingHeader()
		case syslogDecodeOctetFraming:
			d.decodeOctetFraming()
		case syslogDecodeOctetFramingMessage:
			return d.decodeOctetFramingMessage()
		case syslogDecodeUnknown:
			d.decodeUnknown()
		case syslogDecodeUnknownMessage:
			return d.decodeUnknownMessage()
		}
		if d.state == oldState {
			return nil
		}
	}
}

func (d *SyslogDecoder) decodeFraming() {
	nextByte, err := d.buffer.ReadByte()
	if err != nil {
		return
	}
	_ = d.buffer.UnreadByte()
	d.decoding = &UndecodedSyslogMessage{}
	switch nextByte {
	case '<':
		d.state = syslogDecodeImplicitFraming
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		d.state = syslogDecodeOctetFramingHeader
	default:
		d.state = syslogDecodeUnknown
	}
}

func (d *SyslogDecoder) decodeImplicitFraming() {
	b, err := d.buffer.ReadBytes('\n')
	d.decoding.append(b...)
	if err != nil {
		return
	}
	d.state = syslogDecodeImplicitFramingMessage
}

func (d *SyslogDecoder) decodeImplicitFramingMessage() SyslogMessage {
	return d.decodeMessage(0)
}

func (d *SyslogDecoder) decodeOctetFramingHeader() {
	for {
		b, err := d.buffer.ReadByte()
		if err != nil {
			return
		}
		d.decoding.append(b)
		switch b {
		case ' ':
			d.octetsRemaining = d.octets
			d.state = syslogDecodeOctetFraming
			return
		case '0':
			d.octets *= 10
		case '1':
			d.octets = 10*d.octets + 1
		case '2':
			d.octets = 10*d.octets + 2
		case '3':
			d.octets = 10*d.octets + 3
		case '4':
			d.octets = 10*d.octets + 4
		case '5':
			d.octets = 10*d.octets + 5
		case '6':
			d.octets = 10*d.octets + 6
		case '7':
			d.octets = 10*d.octets + 7
		case '8':
			d.octets = 10*d.octets + 8
		case '9':
			d.octets = 10*d.octets + 9
		default:
			d.state = syslogDecodeUnknown
			return
		}
		if d.octets > syslogDecodeLimit {
			d.state = syslogDecodeUnknown
			return
		}
	}
}

func (d *SyslogDecoder) decodeOctetFraming() {
	b := d.buffer.Next(d.octetsRemaining)
	d.decoding.append(b...)
	d.octetsRemaining -= len(b)
	if d.octetsRemaining <= 0 {
		d.state = syslogDecodeOctetFramingMessage
	}
}

func (d *SyslogDecoder) decodeOctetFramingMessage() SyslogMessage {
	len := d.decoding.Len()
	off := len - d.octets
	if off < 2 {
		return d.decodeUnknownMessage()
	}
	return d.decodeMessage(off)
}

func (d *SyslogDecoder) decodeUnknown() {
	for {
		nextByte, err := d.buffer.ReadByte()
		if err != nil {
			return
		}
		d.decoding = &UndecodedSyslogMessage{}
		switch nextByte {
		case '<', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			_ = d.buffer.UnreadByte()
			d.state = syslogDecodeFraming
		default:
			d.decoding.append(nextByte)
			if d.decoding.Len() > syslogDecodeLimit {
				d.state = syslogDecodeUnknownMessage
			}
		}
	}
}

func (d *SyslogDecoder) decodeMessage(off int) SyslogMessage {
	decoded := d.decoding.decode(off)
	d.state = syslogDecodeFraming
	d.decoding = nil
	d.octets = 0
	d.octetsRemaining = 0
	return decoded
}

func (d *SyslogDecoder) decodeUnknownMessage() SyslogMessage {
	decoded := d.decoding
	d.state = syslogDecodeFraming
	d.decoding = nil
	d.octets = 0
	d.octetsRemaining = 0
	return decoded
}
