package log

import (
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"

	pb "github.com/travisjeffery/proglog/internal/proto/v1"
)

type Log struct {
	mu sync.RWMutex

	Config Config

	activeSegment *segment
	segments      []*segment
}

type OffsetOutOfRangeError struct {
	Offset uint64
}

func (e OffsetOutOfRangeError) Error() string {
	return fmt.Sprintf("out of range error offset %d", e.Offset)
}

func NewLog(cfg Config) (*Log, error) {
	if cfg.MaxStoreBytes == 0 {
		cfg.MaxStoreBytes = 1024
	}
	if cfg.MaxIndexBytes == 0 {
		cfg.MaxIndexBytes = 1024
	}
	l := &Log{Config: cfg}

	if err := l.setup(cfg); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *Log) setup(cfg Config) error {
	files, err := os.ReadDir(l.Config.DataDir)
	if err != nil {
		return err
	}
	baseOffsets := make([]uint64, 0, len(files))
	for _, file := range files {
		offStr := strings.TrimSuffix(file.Name(), path.Ext(file.Name()))
		off, err := strconv.ParseUint(offStr, 10, 0)
		if err != nil {
			return err
		}
		baseOffsets = append(baseOffsets, off)
	}
	sort.Slice(baseOffsets, func(i, j int) bool {
		return baseOffsets[i] < baseOffsets[j]
	})
	for i := 0; i < len(baseOffsets); i++ {
		if err := l.newSegment(baseOffsets[i], cfg); err != nil {
			return err
		}
		// baseOffset contains dup for index and store so we skip
		// the dup
		i++
	}
	if l.segments != nil {
		return nil
	}
	return l.newSegment(l.Config.InitialOffset, l.Config)
}

func (l *Log) Append(record *pb.Record) (uint64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	off, err := l.activeSegment.Append(record)
	if err != nil {
		return 0, err
	}
	if l.activeSegment.IsMaxed() {
		err = l.newSegment(off+1, l.Config)
	}
	return off, err
}

func (l *Log) Read(off uint64) (*pb.Record, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var s *segment
	for _, segment := range l.segments {
		if segment.baseOffset <= off && off < segment.nextOffset {
			s = segment
			break
		}
	}
	if s == nil || s.nextOffset <= off {
		return nil, OffsetOutOfRangeError{Offset: off}
	}
	return s.Read(off)
}

func (l *Log) newSegment(off uint64, cfg Config) error {
	s, err := newSegment(off, cfg)
	if err != nil {
		return err
	}
	l.segments = append(l.segments, s)
	l.activeSegment = s
	return nil
}

func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, segment := range l.segments {
		if err := segment.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (l *Log) Remove() error {
	if err := l.Close(); err != nil {
		return err
	}
	return os.RemoveAll(l.Config.DataDir)
}

func (l *Log) Reset() error {
	if err := l.Remove(); err != nil {
		return err
	}
	return l.setup(l.Config)
}

//nolint:unparam //reason: interface
func (l *Log) LowestOffset() (uint64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.segments[0].baseOffset, nil
}

//nolint:unparam //reason: interface
func (l *Log) HighestOffset() (uint64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	off := l.segments[len(l.segments)-1].nextOffset
	if off == 0 {
		return 0, nil
	}
	return off - 1, nil
}

func (l *Log) Truncate(lowest uint64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	segments := make([]*segment, 0, len(l.segments))
	for _, s := range l.segments {
		if s.nextOffset <= lowest+1 {
			if err := s.Remove(); err != nil {
				return err
			}
			continue
		}
		segments = append(segments, s)
	}
	l.segments = segments
	return nil
}

func (l *Log) Reader() io.Reader {
	l.mu.RLock()
	defer l.mu.RUnlock()
	readers := make([]io.Reader, len(l.segments))
	for i, segment := range l.segments {
		readers[i] = &originReader{segment.store, 0}
	}
	return io.MultiReader(readers...)
}

type originReader struct {
	*store
	off int64
}

func (o *originReader) Read(p []byte) (int, error) {
	n, err := o.ReadAt(p, o.off)
	o.off += int64(n)
	return n, err
}
