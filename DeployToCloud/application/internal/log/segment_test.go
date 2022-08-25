package log

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	pb "github.com/travisjeffery/proglog/internal/proto/v1"
)

func TestSegment(t *testing.T) {
	dir, _ := os.MkdirTemp("", "segment-test")
	defer os.RemoveAll(dir)

	want := &pb.Record{Value: []byte("hello world")}
	t.Run("maxed index", func(t *testing.T) {
		cfg := Config{
			DataDir:       dir,
			MaxStoreBytes: 1024,
			MaxIndexBytes: entWidth * 3,
		}

		s, err := newSegment(16, cfg)
		require.NoError(t, err)
		require.Equal(t, uint64(16), s.nextOffset, s.nextOffset)
		require.False(t, s.IsMaxed())

		for i := uint64(0); i < 3; i++ {
			off, err := s.Append(want)
			require.NoError(t, err)
			require.Equal(t, 16+i, off)

			got, err := s.Read(off)
			require.NoError(t, err)
			require.Equal(t, want.Value, got.Value)
		}

		_, err = s.Append(want)
		require.Equal(t, io.EOF, err)

		// maxed index
		require.True(t, s.IsMaxed())
	})
	t.Run("maxed store", func(t *testing.T) {
		cfg := Config{
			DataDir:       dir,
			MaxStoreBytes: uint64(len(want.Value) * 3),
			MaxIndexBytes: 1024,
		}

		s, err := newSegment(16, cfg)
		require.NoError(t, err)
		// maxed store
		require.True(t, s.IsMaxed())

		err = s.Remove()
		require.NoError(t, err)
		s, err = newSegment(16, cfg)
		require.NoError(t, err)
		require.False(t, s.IsMaxed())
	})
}
