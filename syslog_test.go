//
// Copyright (C) 2023-2025 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

package log_test

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-conf"
	"github.com/tdrn-org/go-log"
	"github.com/tdrn-org/go-tlsconf/tlsclient"
	"github.com/tdrn-org/go-tlsconf/tlsserver"
)

func TestSyslogLogEncodings(t *testing.T) {
	receiver := newSyslogTCPReceiver(t, false).Accept()
	defer receiver.Close()
	encodings := []log.SyslogEncoding{log.SyslogEncodingRFC3164, log.SyslogEncodingRFC3164F, log.SyslogEncodingRFC5424, log.SyslogEncodingRFC5424F}
	for _, encoding := range encodings {
		config := log.Config{
			Level:          slog.LevelDebug.String(),
			Target:         log.TargetSyslog,
			SyslogNetwork:  "tcp",
			SyslogAddress:  receiver.Address(),
			SyslogEncoding: string(encoding),
		}
		logger, _ := config.GetLogger(nil)
		generateLogs(logger, slog.LevelDebug, slog.LevelError+1, 100, slog.String(log.SyslogKey, "ID47"))
	}
	receiver.Wait(10)
	valid, invalid := receiver.Counts()
	require.Greater(t, valid, 0)
	require.Zero(t, invalid)
}

func TestSyslogUDP(t *testing.T) {
	receiver := newSyslogUDPReceiver(t).Read()
	defer receiver.Close()
	config := log.Config{
		Level:          slog.LevelDebug.String(),
		Target:         log.TargetSyslog,
		SyslogNetwork:  "udp",
		SyslogEncoding: string(log.SyslogEncodingRFC3164),
		SyslogAddress:  receiver.Address(),
	}
	logger, _ := config.GetLogger(nil)
	generateLogs(logger, slog.LevelDebug, slog.LevelError+1, 100, slog.String(log.SyslogKey, "ID47"))
	receiver.Wait(10)
	valid, invalid := receiver.Counts()
	require.Greater(t, valid, 0)
	require.Zero(t, invalid)
}

func TestSyslogTLS(t *testing.T) {
	receiver := newSyslogTCPReceiver(t, true).Accept()
	defer receiver.Close()
	config := log.Config{
		Level:         slog.LevelDebug.String(),
		Target:        log.TargetSyslog,
		SyslogNetwork: "tcp+tls",
		SyslogAddress: receiver.Address(),
	}
	logger, _ := config.GetLogger(nil)
	generateLogs(logger, slog.LevelDebug, slog.LevelError+1, 100, slog.String(log.SyslogKey, "ID47"))
	receiver.Wait(10)
	valid, invalid := receiver.Counts()
	require.Greater(t, valid, 0)
	require.Zero(t, invalid)
}

const syslogListenAddress = "localhost:"

var rfc3164Pattern = regexp.MustCompile(`^<\d+>[JFMASOND]`)
var rfc3164FPattern = regexp.MustCompile(`^\d+ <\d+>[JFMASOND]`)
var rfc5424Pattern = regexp.MustCompile(`^<\d+>1 `)
var rfc5424FPattern = regexp.MustCompile(`^\d+ <\d+>1 `)

type syslogReceiver struct {
	validCount   int
	invalidCount int
	mutex        sync.Mutex
}

func (r *syslogReceiver) evalMessage(msg string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if rfc3164Pattern.MatchString(msg) {
		r.validCount++
	} else if rfc3164FPattern.MatchString(msg) {
		r.validCount++
	} else if rfc5424Pattern.MatchString(msg) {
		r.validCount++
	} else if rfc5424FPattern.MatchString(msg) {
		r.validCount++
	} else {
		fmt.Printf("invalid syslog message: %s\n", strconv.Quote(string(msg)))
		r.invalidCount++
	}
}

func (r *syslogReceiver) Wait(total int) {
	for {
		valid, invalid := r.Counts()
		if valid+invalid >= total {
			return
		}
		time.Sleep(time.Second)
	}
}

func (r *syslogReceiver) Counts() (int, int) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.validCount, r.invalidCount
}

type syslogTCPReceiver struct {
	syslogReceiver
	listener net.Listener
	address  string
}

func newSyslogTCPReceiver(t *testing.T, enableTLS bool) *syslogTCPReceiver {
	var listener net.Listener
	if enableTLS {
		tlsServerConfig, _ := conf.LookupConfiguration[*tlsserver.Config]()
		tlsListener, err := tls.Listen("tcp", syslogListenAddress, &tlsServerConfig.Config)
		require.NoError(t, err)
		listener = tlsListener
	} else {
		netListener, err := net.Listen("tcp", syslogListenAddress)
		require.NoError(t, err)
		listener = netListener
	}
	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	address := syslogListenAddress + port
	return &syslogTCPReceiver{
		listener: listener,
		address:  address,
	}
}

func (r *syslogTCPReceiver) Address() string {
	return r.address
}

func (r *syslogTCPReceiver) Close() error {
	return r.listener.Close()
}

func (r *syslogTCPReceiver) Accept() *syslogTCPReceiver {
	go func() {
		for {
			conn, err := r.listener.Accept()
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if err != nil {
				slog.Error("accept connection failure", slog.Any("err", err))
				return
			}
			go r.handleConn(conn)
		}
	}()
	return r
}

func (r *syslogTCPReceiver) handleConn(conn net.Conn) {
	defer conn.Close()
	buffer := make([]byte, 4096)
	for {
		len, err := conn.Read(buffer)
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			slog.Error("tcp read failure", slog.Any("err", err))
			return
		}
		r.syslogReceiver.evalMessage(string(buffer[:len]))
	}
}

type syslogUDPReceiver struct {
	syslogReceiver
	packetConn net.PacketConn
}

func newSyslogUDPReceiver(t *testing.T) *syslogUDPReceiver {
	packetConn, err := net.ListenPacket("udp", "localhost:")
	require.NoError(t, err)
	return &syslogUDPReceiver{
		packetConn: packetConn,
	}
}

func (r *syslogUDPReceiver) Address() string {
	return r.packetConn.LocalAddr().String()
}

func (r *syslogUDPReceiver) Close() error {
	return r.packetConn.Close()
}

func (r *syslogUDPReceiver) Read() *syslogUDPReceiver {
	go func() {
		buffer := make([]byte, 4096)
		for {
			len, _, err := r.packetConn.ReadFrom(buffer)
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if err != nil {
				slog.Error("udp read failure", slog.Any("err", err))
				return
			}
			r.syslogReceiver.evalMessage(string(buffer[:len]))
		}
	}()
	return r
}

func init() {
	_ = tlsserver.SetOptions(tlsserver.UseEphemeralCertificate("localhost"))
	_ = tlsclient.SetOptions(tlsclient.AppendServerCertificates())
}
