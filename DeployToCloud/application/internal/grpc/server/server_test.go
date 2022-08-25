package server

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/travisjeffery/go-dynaport"
	"go.opencensus.io/examples/exporter"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	"github.com/travisjeffery/proglog/internal/grpc/auth"
	"github.com/travisjeffery/proglog/internal/log"
	pb "github.com/travisjeffery/proglog/internal/proto/v1"
	innertls "github.com/travisjeffery/proglog/internal/tls"
)

var debug = flag.Bool("debug", false, "Enable observability for debugging.")

func TestMain(m *testing.M) {
	flag.Parse()
	if *debug {
		logger, err := zap.NewDevelopment()
		if err != nil {
			panic(err)
		}
		zap.ReplaceGlobals(logger)
	}
	os.Exit(m.Run())
}

func TestServer(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T, clients clients){
		"produce/consume a message to/from the log succeeeds": testProduceConsume,
		"produce/consume stream succeeds":                     testProduceConsumeStream,
		"consume past log boundary fails":                     testConsumePastBoundary,
		"unauthorized fails":                                  testUnauthorized,
		"healthcheck succeeds":                                testHealthCheck,
	} {
		t.Run(scenario, func(t *testing.T) {
			cs, teardown := setupTest(t)
			defer teardown()
			fn(t, cs)
		})
	}
}

type clients struct {
	Root   pb.LogClient
	Nobody pb.LogClient
	Health healthpb.HealthClient
}

func setupTest(t *testing.T) (clients clients, teardown func()) {
	t.Helper()

	ports := dynaport.Get(3)

	rpcAddr := &net.TCPAddr{IP: []byte{127, 0, 0, 1}, Port: ports[0]}

	tlsConfig, err := innertls.SetupTLS(innertls.Args{
		CertFile: innertls.ServerCertFile,
		KeyFile:  innertls.ServerKeyFile,
		CAFile:   innertls.CAFile,
		Server:   true,
	})
	require.NoError(t, err)

	l, err := net.Listen("tcp", rpcAddr.String())
	require.NoError(t, err)

	dataDir, err := os.MkdirTemp("", "server-test-log")
	require.NoError(t, err)
	defer os.RemoveAll(dataDir)

	clog, err := log.NewLog(log.Config{DataDir: dataDir})
	require.NoError(t, err)

	authorizer := auth.NewAuthorizer(auth.Args{ModelFile: innertls.ACLModelFile, PolicyFile: innertls.ACLPolicyFile})

	var telemetryExporter *exporter.LogExporter
	if *debug {
		metricsLogFile, err := os.CreateTemp("", "metrics-*.log")
		require.NoError(t, err)
		t.Logf("metrics log file: %s", metricsLogFile.Name())

		tracesLogFile, err := os.CreateTemp("", "traces-*.log")
		require.NoError(t, err)
		t.Logf("traces log file: %s", tracesLogFile.Name())

		telemetryExporter, err = exporter.NewLogExporter(exporter.Options{
			MetricsLogFile:    metricsLogFile.Name(),
			TracesLogFile:     tracesLogFile.Name(),
			ReportingInterval: time.Second,
		})
		require.NoError(t, err)
		err = telemetryExporter.Start()
		require.NoError(t, err)
	}

	server, err := NewGRPCServer(clog, authorizer, nil, tlsConfig)
	require.NoError(t, err)

	go func() {
		if err := server.Serve(l); err != nil {
			panic(err)
		}
	}()

	newLogClient := func(crtPath, keyPath string) (*grpc.ClientConn, pb.LogClient) {
		tlsConfig, err := innertls.SetupTLS(innertls.Args{
			CertFile: crtPath,
			KeyFile:  keyPath,
			CAFile:   innertls.CAFile,
			Server:   false,
		})
		require.NoError(t, err)
		tlsCreds := credentials.NewTLS(tlsConfig)
		opts := []grpc.DialOption{grpc.WithTransportCredentials(tlsCreds)}
		conn, err := grpc.Dial(l.Addr().String(), opts...)
		require.NoError(t, err)
		client := pb.NewLogClient(conn)
		return conn, client
	}

	var rootConn *grpc.ClientConn
	rootConn, clients.Root = newLogClient(innertls.RootClientCertFile, innertls.RootClientKeyFile)

	var nobodyConn *grpc.ClientConn
	nobodyConn, clients.Nobody = newLogClient(innertls.NobodyClientCertFile, innertls.NobodyClientKeyFile)

	clients.Health = healthpb.NewHealthClient(nobodyConn)

	return clients, func() {
		server.Stop()
		rootConn.Close()
		nobodyConn.Close()
		l.Close()
		if telemetryExporter != nil {
			time.Sleep(1500 * time.Millisecond)
			telemetryExporter.Stop()
			telemetryExporter.Close()
		}
		clog.Remove()
	}
}

func testProduceConsume(t *testing.T, clients clients) {
	t.Helper()
	ctx := context.Background()

	want := &pb.Record{Value: []byte("hello world")}

	produce, err := clients.Root.Produce(ctx, &pb.ProduceRequest{Record: want})
	require.NoError(t, err)

	consume, err := clients.Root.Consume(ctx, &pb.ConsumeRequest{Offset: produce.Offset})
	require.NoError(t, err)
	require.Equal(t, want.Value, consume.Record.Value)
	require.Equal(t, want.Offset, produce.Offset)
}

func testConsumePastBoundary(t *testing.T, clients clients) {
	t.Helper()
	ctx := context.Background()

	produce, err := clients.Root.Produce(ctx, &pb.ProduceRequest{Record: &pb.Record{Value: []byte("hello world")}})
	require.NoError(t, err)

	consume, err := clients.Root.Consume(ctx, &pb.ConsumeRequest{Offset: produce.Offset + 1})
	require.Nil(t, consume)
	got := status.Code(err)
	want := status.Code((&OffsetOutOfRangeError{}).GRPCStatus().Err())
	require.Equal(t, want, got)
}

func testProduceConsumeStream(t *testing.T, clients clients) {
	t.Helper()
	ctx := context.Background()

	records := []*pb.Record{
		{Value: []byte("first message"), Offset: 0},
		{Value: []byte("second message"), Offset: 1},
	}
	t.Run("ProduceStream", func(t *testing.T) {
		stream, err := clients.Root.ProduceStream(ctx)
		require.NoError(t, err)

		for i, record := range records {
			err = stream.Send(&pb.ProduceRequest{Record: record})
			require.NoError(t, err)
			res, err := stream.Recv()
			require.NoError(t, err)
			offset := uint64(i)
			if res.Offset != offset {
				t.Fatalf("got offset: %d, want: %d", res.Offset, offset)
			}
		}
	})
	t.Run("ConsumeStream", func(t *testing.T) {
		stream, err := clients.Root.ConsumeStream(ctx, &pb.ConsumeRequest{Offset: 0})
		require.NoError(t, err)

		for i, record := range records {
			res, err := stream.Recv()
			require.NoError(t, err)
			require.Equal(t, res.Record, &pb.Record{Value: record.Value, Offset: uint64(i)})
		}
	})
}

func testUnauthorized(t *testing.T, clients clients) {
	t.Helper()
	ctx := context.Background()
	produce, err := clients.Nobody.Produce(ctx, &pb.ProduceRequest{Record: &pb.Record{Value: []byte("hello world")}})
	if produce != nil {
		t.Fatalf("produce response should be nil")
	}
	gotCode, wantCode := status.Code(err), codes.PermissionDenied
	if gotCode != wantCode {
		t.Fatalf("got code: %d, want: %d", gotCode, wantCode)
	}
	consume, err := clients.Nobody.Consume(ctx, &pb.ConsumeRequest{Offset: 0})
	if consume != nil {
		t.Fatalf("consume response should be nil")
	}
	gotCode, wantCode = status.Code(err), codes.PermissionDenied
	if gotCode != wantCode {
		t.Fatalf("got code: %d, want: %d", gotCode, wantCode)
	}
}

func testHealthCheck(t *testing.T, clients clients) {
	t.Helper()
	ctx := context.Background()
	res, err := clients.Health.Check(ctx, &healthpb.HealthCheckRequest{})
	require.NoError(t, err)
	require.Equal(t, healthpb.HealthCheckResponse_SERVING, res.Status)
}

func TestErrOffsetOutOfRange(t *testing.T) {
	err := func() error {
		return log.OffsetOutOfRangeError{Offset: 2}
	}()
	require.Equal(t, true, errors.As(err, &log.OffsetOutOfRangeError{}))

	fmt.Println(OffsetOutOfRangeError{err.(log.OffsetOutOfRangeError).Offset})
}
