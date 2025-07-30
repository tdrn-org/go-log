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

var syslogRFC3164TestMessages = []string{
	"<34>Oct 11 22:14:15 mymachine su: 'su root' failed for lonvick on /dev/pts/8\n",
}

func TestSyslogDecoderRFC3164(t *testing.T) {
	decoder := &log.SyslogDecoder{}
	for _, rawMessage := range syslogRFC3164TestMessages {
		decoder.Feed([]byte(rawMessage))
	}
	decodedMessages := decoder.Decode()
	require.Len(t, decodedMessages, 1)
	for _, decodedMessage := range decodedMessages {
		require.IsType(t, &log.RFC3164SyslogMessage{}, decodedMessage)
	}
}

var syslogRFC5424TestMessages = []string{
	"<34>1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - BOM'su root' failed for lonvick on /dev/pts/8\n",
	"<165>1 2003-08-24T05:14:15.000003-07:00 192.0.2.1 myproc 8710 - - %% It's time to make the do-nuts.\n",
	"<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog - ID47 [exampleSDID@32473 iut=\"3\" eventSource=\"Application\" eventID=\"1011\"] BOMAn application event log entry...\n",
	"<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog - ID47 [exampleSDID@32473 iut=\"3\" eventSource=\"Application\" eventID=\"1011\"][examplePriority@32473 class=\"high\"]\n",
}

func TestSyslogDecoderRFC5424(t *testing.T) {
	decoder := &log.SyslogDecoder{}
	for _, rawMessage := range syslogRFC5424TestMessages {
		decoder.Feed([]byte(rawMessage))
	}
	decodedMessages := decoder.Decode()
	require.Len(t, decodedMessages, 4)
	for _, decodedMessage := range decodedMessages {
		require.IsType(t, &log.RFC5424SyslogMessage{}, decodedMessage)
	}
}
