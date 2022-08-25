package loadbalance_test

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"

	"github.com/travisjeffery/proglog/internal/grpc/loadbalance"
	"github.com/travisjeffery/proglog/internal/grpc/server"
	pb "github.com/travisjeffery/proglog/internal/proto/v1"
	"github.com/travisjeffery/proglog/internal/tls"
)

func TestResolver(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	tlsConfig, err := tls.SetupTLS(tls.Args{
		CertFile: tls.ServerCertFile,
		KeyFile:  tls.ServerKeyFile,
		CAFile:   tls.CAFile,
		Server:   true,
	})
	require.NoError(t, err)

	srv, err := server.NewGRPCServer(nil, nil, &getServers{}, tlsConfig)
	require.NoError(t, err)

	go srv.Serve(l)

	conn := &clientConn{}
	tlsConfig, err = tls.SetupTLS(tls.Args{
		CertFile: tls.RootClientCertFile,
		KeyFile:  tls.RootClientKeyFile,
		CAFile:   tls.CAFile,
		Server:   false,
	})
	require.NoError(t, err)
	clientCreds := credentials.NewTLS(tlsConfig)
	opts := resolver.BuildOptions{DialCreds: clientCreds}
	r := &loadbalance.Resolver{}
	_, err = r.Build(resolver.Target{Endpoint: l.Addr().String()}, conn, opts)
	require.NoError(t, err)

	wantState := resolver.State{
		Addresses: []resolver.Address{
			{Addr: "localhost:9001", Attributes: attributes.New("is_leader", true)},
			{Addr: "localhost:9002", Attributes: attributes.New("is_leader", false)},
		},
	}
	require.Equal(t, wantState, conn.state)

	conn.state.Addresses = nil
	r.ResolveNow(resolver.ResolveNowOptions{})
	require.Equal(t, wantState, conn.state)
}

type getServers struct{}

func (s *getServers) GetServers() ([]*pb.Server, error) {
	return []*pb.Server{
		{Id: "leader", RpcAddr: "localhost:9001", IsLeader: true},
		{Id: "follower", RpcAddr: "localhost:9002"},
	}, nil
}

type clientConn struct {
	resolver.ClientConn
	state resolver.State
}

func (c *clientConn) UpdateState(state resolver.State) {
	c.state = state
}

func (c *clientConn) ReportError(err error) {}

func (c *clientConn) NewAddress(addrs []resolver.Address) {}

func (c *clientConn) NewServiceConfig(config string) {}

func (c *clientConn) ParseServiceConfig(config string) *serviceconfig.ParseResult { return nil }
