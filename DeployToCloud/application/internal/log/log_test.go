package log

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	pb "github.com/travisjeffery/proglog/internal/proto/v1"
)

func TestLog(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T, log *Log){
		"append and read a record succeeds": testAppendRead,
		"offset out of range error":         testOutOfRangeErr,
		"init with existing segments":       testInitExisting,
		"reader":                            testReader,
	} {
		t.Run(scenario, func(t *testing.T) {
			dir, err := os.MkdirTemp("", "store-test")
			require.NoError(t, err)
			defer os.RemoveAll(dir)

			c := Config{MaxStoreBytes: 32}
			c.MaxStoreBytes = 32
			c.DataDir = dir
			log, err := NewLog(c)
			require.NoError(t, err)

			fn(t, log)
		})
	}
}

func testAppendRead(t *testing.T, log *Log) {
	t.Helper()
	a := &pb.Record{Value: []byte("hello world")}
	off, err := log.Append(a)
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)

	read, err := log.Read(off)
	require.NoError(t, err)
	require.Equal(t, a.Value, read.Value)
}

func testOutOfRangeErr(t *testing.T, log *Log) {
	t.Helper()
	read, err := log.Read(1)
	require.Nil(t, read)
	require.Error(t, err)
}

func testInitExisting(t *testing.T, o *Log) {
	t.Helper()
	a := &pb.Record{
		Value: []byte("hello world"),
	}
	for i := 0; i < 3; i++ {
		_, err := o.Append(a)
		require.NoError(t, err)
	}
	require.NoError(t, o.Close())

	n, err := NewLog(o.Config)
	require.NoError(t, err)
	off, err := n.Append(a)
	require.NoError(t, err)
	require.Equal(t, uint64(3), off)
}

func testReader(t *testing.T, log *Log) {
	t.Helper()
	a := &pb.Record{Value: []byte("hello world")}
	off, err := log.Append(a)
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)

	reader := log.Reader()
	b, err := io.ReadAll(reader)
	require.NoError(t, err)

	read := &pb.Record{}
	err = proto.Unmarshal(b, read)
	require.NoError(t, err)
	require.Equal(t, a.Value, read.Value)
}
