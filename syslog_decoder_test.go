//
// Copyright (C) 2023-2025 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

package log_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-log"
)

func TestSyslogDecoderRFC3614(t *testing.T) {
	raw := []byte("<34>Oct 11 22:14:15 mymachine su: 'su root' failed for lonvick on /dev/pts/8\n")
	decoder := &log.SyslogDecoder{}
	decoder.Feed(raw)
	messages := decoder.Decode()
	require.Len(t, messages, 1)
}

func TestSyslogDecoderRFC5424(t *testing.T) {
	raw := []byte("<165>1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 [exampleSDID@32473 iut=\"3\" eventSource=\"\" eventID=\"1011\"] BOMAn application log entry...\n")
	decoder := &log.SyslogDecoder{}
	decoder.Feed(raw)
	messages := decoder.Decode()
	require.Len(t, messages, 1)
}
