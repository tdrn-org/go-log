// handler.go
//
// Copyright (C) 2023-2025 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

package log

import (
	"io"
	"strings"
	"sync"
)

var messageBuilderPool = sync.Pool{
	New: func() any { return &messageBuilder{} },
}

func getMessageBuilder() *messageBuilder {
	return messageBuilderPool.Get().(*messageBuilder)
}

type messageBuilder struct {
	buffer strings.Builder
}

func (b *messageBuilder) release() {
	b.buffer.Reset()
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

func (b *messageBuilder) write(w io.Writer) (int, error) {
	_, _ = b.buffer.WriteRune('\n')
	return w.Write([]byte(b.buffer.String()))
}
