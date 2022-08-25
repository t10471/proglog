package raft

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/hashicorp/raft"
	"github.com/soheilhy/cmux"

	innertls "github.com/travisjeffery/proglog/internal/tls"
)

const RPC = 1

type StreamLayer struct {
	ln              net.Listener
	serverTLSConfig *tls.Config
	peerTLSConfig   *tls.Config
}

var _ raft.StreamLayer = (*StreamLayer)(nil)

func NewStreamLayer(mux cmux.CMux, cfg innertls.Config) *StreamLayer {
	raftLn := mux.Match(func(reader io.Reader) bool {
		b := make([]byte, 1)
		if _, err := reader.Read(b); err != nil {
			return false
		}
		return bytes.Equal(b, []byte{byte(RPC)})
	})
	return &StreamLayer{
		ln:              raftLn,
		serverTLSConfig: cfg.ServerTLSConfig,
		peerTLSConfig:   cfg.PeerTLSConfig,
	}
}

func (s *StreamLayer) Dial(addr raft.ServerAddress, timeout time.Duration) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.Dial("tcp", string(addr))
	if err != nil {
		return nil, err
	}
	// identify to mux this is a raft rpc
	if _, err := conn.Write([]byte{byte(RPC)}); err != nil {
		return nil, err
	}
	if s.peerTLSConfig != nil {
		conn = tls.Client(conn, s.peerTLSConfig)
	}
	return conn, err
}

func (s *StreamLayer) Accept() (net.Conn, error) {
	conn, err := s.ln.Accept()
	if err != nil {
		return nil, err
	}
	b := make([]byte, 1)
	if _, err = conn.Read(b); err != nil {
		return nil, err
	}
	if !bytes.Equal([]byte{byte(RPC)}, b) {
		return nil, fmt.Errorf("not a raft rpc")
	}
	if s.serverTLSConfig != nil {
		return tls.Server(conn, s.serverTLSConfig), nil
	}
	return conn, nil
}

func (s *StreamLayer) Close() error {
	return s.ln.Close()
}

func (s *StreamLayer) Addr() net.Addr {
	return s.ln.Addr()
}
