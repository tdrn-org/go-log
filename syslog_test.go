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
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-conf"
	"github.com/tdrn-org/go-log"
	"github.com/tdrn-org/go-tlsconf/tlsclient"
	"github.com/tdrn-org/go-tlsconf/tlsserver"
)

func TestSyslogConfig(t *testing.T) {
	receiver := newSyslogTCPReceiver(t, false).Accept()
	defer receiver.Close()
	config := log.Config{
		Level:         slog.LevelDebug.String(),
		Target:        log.TargetSyslog,
		SyslogAddress: receiver.Address(),
	}
	logger, _ := config.GetLogger(nil)
	generateLogs(logger, slog.LevelDebug, log.LevelNotice, 100)
	receiver.Wait(100)
}

func TestSyslogHanlder(t *testing.T) {
	h := log.NewSyslogHandler(os.Stdout, &log.SyslogHandlerOptions{
		HandlerOptions: slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
			ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
				switch attr.Key {
				case slog.TimeKey:
					return slog.Time(slog.TimeKey, time.Time{})
				case slog.SourceKey:
					return slog.Attr{}
				default:
					return attr
				}
			},
		},
		Encoding: log.SyslogEncodingDefault,
		Facility: 99,
	})
	logger := slog.New(h)
	logger = logger.With(slog.Group("test", slog.String("name", "SyslogPlainHandler")))
	logger = logger.WithGroup("generate")
	generateLogs(logger, slog.LevelDebug, log.LevelNotice, 100)
}

func TestSyslogLogEncodings(t *testing.T) {
	encodings := []log.SyslogEncoding{
		log.SyslogEncodingDefault,
		log.SyslogEncodingRFC3164,
		log.SyslogEncodingRFC3164F,
		log.SyslogEncodingRFC5424,
		log.SyslogEncodingRFC5424F,
	}
	for _, encoding := range encodings {
		t.Run(string(encoding), func(t *testing.T) {
			receiver := newSyslogTCPReceiver(t, false).Accept()
			defer receiver.Close()
			config := log.Config{
				Level:          slog.LevelDebug.String(),
				Target:         log.TargetSyslog,
				SyslogNetwork:  "tcp",
				SyslogAddress:  receiver.Address(),
				SyslogEncoding: string(encoding),
			}
			logger, _ := config.GetLogger(nil)
			logger = logger.With(slog.String(log.SyslogKey, "ID47"))
			generateLogs(logger, slog.LevelDebug, log.LevelNotice, 100)
			receiver.Wait(100)
		})
	}
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
	logger = logger.With(slog.String(log.SyslogKey, "ID47"))
	generateLogs(logger, slog.LevelDebug, log.LevelNotice, 100)
	receiver.Wait(100)
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
	logger = logger.With(slog.String(log.SyslogKey, "ID47"))
	generateLogs(logger, slog.LevelDebug, log.LevelNotice, 100)
	receiver.Wait(100)
}

const syslogListenAddress = "localhost:"

type syslogReceiver struct {
	undecodedMessageCount int
	rfc3614MessageCount   int
	rfc5424MessageCount   int
	mutex                 sync.Mutex
}

func (r *syslogReceiver) evalMessage(message log.SyslogMessage) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	switch message.(type) {
	case *log.UndecodedSyslogMessage:
		r.undecodedMessageCount++
	case *log.RFC3164SyslogMessage:
		r.rfc3614MessageCount++
	case *log.RFC5424SyslogMessage:
		r.rfc5424MessageCount++
	}
	fmt.Printf("syslog: %s\n", message.String())
}

func (r *syslogReceiver) Wait(total int) {
	for {
		undecoded, rfc3614, rfc5424 := r.MessageCounts()
		if undecoded+rfc3614+rfc5424 >= total {
			return
		}
		time.Sleep(time.Second)
	}
}

func (r *syslogReceiver) MessageCounts() (int, int, int) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.undecodedMessageCount, r.rfc3614MessageCount, r.rfc5424MessageCount
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
	decoder := &log.SyslogDecoder{}
	for {
		err := decoder.Read(conn)
		if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			slog.Error("tcp read failure", slog.Any("err", err))
			return
		}
		messages := decoder.Decode()
		for _, message := range messages {
			r.syslogReceiver.evalMessage(message)
		}
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
		decoder := &log.SyslogDecoder{}
		for {
			err := decoder.ReadFrom(r.packetConn)
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if err != nil {
				slog.Error("udp read failure", slog.Any("err", err))
				return
			}
			messages := decoder.Decode()
			for _, message := range messages {
				r.syslogReceiver.evalMessage(message)
			}
		}
	}()
	return r
}

func init() {
	_ = tlsserver.SetOptions(tlsserver.UseEphemeralCertificate("localhost", tlsserver.CertificateAlgorithmDefault))
	_ = tlsclient.SetOptions(tlsclient.AppendServerCertificates())
}
