//
// Copyright (C) 2023-2025 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

package log

import (
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"sync"
	"unicode/utf8"
)

const messageBuilderBufferStart = 16
const messageBuilderBufferSize = 1024 + messageBuilderBufferStart

var emptyAttr = slog.Attr{}
var noGroups = []string{}

var messageBuilderPool = sync.Pool{
	New: func() any {
		return &messageBuilder{
			buffer: make([]byte, messageBuilderBufferStart, messageBuilderBufferSize),
		}
	},
}

func getMessageBuilder(groups []string) *messageBuilder {
	b := messageBuilderPool.Get().(*messageBuilder)
	for _, group := range groups {
		b.pushGroup(group)
	}
	return b
}

type messageBuilder struct {
	buffer      []byte
	stack       groupStack
	conditional string
}

func (b *messageBuilder) Release() {
	b.buffer = b.buffer[:messageBuilderBufferStart]
	b.stack.Reset()
	messageBuilderPool.Put(b)
}

func (b *messageBuilder) AppendConditional(s string) *messageBuilder {
	b.conditional = s
	return b
}

func (b *messageBuilder) CompleteConditional(yes, no string) bool {
	fired := b.conditional == ""
	if fired {
		b.AppendString(yes)
	} else {
		b.conditional = ""
		b.AppendString(no)
	}
	return fired
}

func (b *messageBuilder) writeConditional() {
	if b.conditional != "" {
		b.buffer = append(b.buffer, b.conditional...)
		b.conditional = ""
	}
}

func (b *messageBuilder) AppendRune(r rune) *messageBuilder {
	b.writeConditional()
	b.buffer = utf8.AppendRune(b.buffer, r)
	return b
}

func (b *messageBuilder) AppendString(s string) *messageBuilder {
	if s != "" {
		b.writeConditional()
		b.buffer = append(b.buffer, s...)
	}
	return b
}

func (b *messageBuilder) AppendBytes(p []byte) *messageBuilder {
	if len(p) > 0 {
		b.writeConditional()
		b.buffer = append(b.buffer, p...)
	}
	return b
}

func (b *messageBuilder) Attrs(f func(slog.Attr) bool) func(slog.Attr) bool {
	return func(attr slog.Attr) bool {
		if attr.Equal(emptyAttr) {
			return true
		} else if attr.Value.Kind() == slog.KindGroup {
			b.pushGroup(attr.Key)
			defer b.popGroup(attr.Key)
			groupAttrs := attr.Value.Group()
			for _, groupAttr := range groupAttrs {
				if !f(groupAttr) {
					break
				}
			}
			return true
		}
		return f(attr)
	}
}

func (b *messageBuilder) pushGroup(group string) {
	b.stack.Push(group)
}

func (b *messageBuilder) popGroup(group string) {
	b.stack.Pop(group)
}

func (b *messageBuilder) Groups() []string {
	return b.stack.groups
}

func (b *messageBuilder) GroupPath() string {
	if len(b.stack.path) == 0 {
		return ""
	}
	return b.stack.path[len(b.stack.path)-1]
}

func (b *messageBuilder) Bytes() []byte {
	return b.buffer[messageBuilderBufferStart:]
}

func (b *messageBuilder) Write(w io.Writer, implicit bool) (int, error) {
	writeStart := messageBuilderBufferStart
	if implicit {
		b.buffer = utf8.AppendRune(b.buffer, '\n')
	} else {
		frameHeader := []byte(strconv.Itoa(len(b.buffer) - messageBuilderBufferStart))
		writeStart = messageBuilderBufferStart - len(frameHeader) - 1
		if writeStart < 0 {
			return 0, fmt.Errorf("message too large: %s", frameHeader)
		}
		copy(b.buffer[writeStart:], frameHeader)
		b.buffer[messageBuilderBufferStart-1] = ' '
	}
	return w.Write(b.buffer[writeStart:])
}

type groupStack struct {
	groups []string
	path   []string
}

func (s *groupStack) Reset() {
	s.groups = nil
	s.path = nil
}

func (s *groupStack) Push(group string) {
	if group != "" {
		if len(s.groups) > 0 {
			s.groups = append(s.groups, group)
			s.path = append(s.path, s.path[len(s.path)-1]+group+".")
		} else {
			s.groups = []string{group}
			s.path = []string{group + "."}
		}
	}
}

func (s *groupStack) Pop(group string) {
	if group != "" {
		s.groups = s.groups[:len(s.groups)-1]
		s.path = s.groups[:len(s.path)-1]
	}
}
