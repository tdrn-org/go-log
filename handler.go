// handler.go
//
// Copyright (C) 2023-2025 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

package log

import (
	"io"
	"log/slog"
	"strings"
	"sync"
)

var messageBuilderPool = sync.Pool{
	New: func() any { return &messageBuilder{} },
}

func getMessageBuilder(groups []string) *messageBuilder {
	b := messageBuilderPool.Get().(*messageBuilder)
	for _, group := range groups {
		b.pushGroup(group)
	}
	return b
}

type messageBuilder struct {
	buffer strings.Builder
	stack  groupStack
}

func (b *messageBuilder) release() {
	b.buffer.Reset()
	b.stack.reset()
	messageBuilderPool.Put(b)
}

func (b *messageBuilder) appendRune(r rune) *messageBuilder {
	_, _ = b.buffer.WriteRune(r)
	return b
}

func (b *messageBuilder) appendString(s string) *messageBuilder {
	if s != "" {
		_, _ = b.buffer.WriteString(s)
	}
	return b
}

func (b *messageBuilder) appendBytes(p []byte) *messageBuilder {
	if len(p) > 0 {
		_, _ = b.buffer.Write(p)
	}
	return b
}

func (b *messageBuilder) attrs(f func(slog.Attr) bool) func(slog.Attr) bool {
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
	b.stack.push(group)
}

func (b *messageBuilder) popGroup(group string) {
	b.stack.pop(group)
}

func (b *messageBuilder) groups() []string {
	return b.stack.groups
}

func (b *messageBuilder) groupPath() string {
	if len(b.stack.path) == 0 {
		return ""
	}
	return b.stack.path[len(b.stack.path)-1]
}

func (b *messageBuilder) bytes() []byte {
	return []byte(b.buffer.String())
}

func (b *messageBuilder) write(w io.Writer) (int, error) {
	_, _ = b.buffer.WriteRune('\n')
	return w.Write([]byte(b.buffer.String()))
}

type groupStack struct {
	groups []string
	path   []string
}

func (s *groupStack) reset() {
	s.groups = nil
	s.path = nil
}

func (s *groupStack) push(group string) {
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

func (s *groupStack) pop(group string) {
	if group != "" {
		s.groups = s.groups[:len(s.groups)-1]
		s.path = s.groups[:len(s.path)-1]
	}
}
