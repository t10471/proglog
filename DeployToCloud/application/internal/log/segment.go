package log

import (
	"fmt"
	"os"
	"path"

	"google.golang.org/protobuf/proto"

	pb "github.com/travisjeffery/proglog/internal/proto/v1"
)

type segment struct {
	store      *store
	index      *index
	baseOffset uint64
	nextOffset uint64
	config     Config
}

type Config struct {
	DataDir       string
	MaxStoreBytes uint64
	MaxIndexBytes uint64
	InitialOffset uint64
}

func newSegment(baseOffset uint64, cfg Config) (*segment, error) {
	s := &segment{
		baseOffset: baseOffset,
		config:     cfg,
	}
	var err error
	p := path.Join(cfg.DataDir, fmt.Sprintf("%d%s", baseOffset, ".store"))
	storeFile, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	if s.store, err = newStore(storeFile); err != nil {
		return nil, err
	}
	p = path.Join(cfg.DataDir, fmt.Sprintf("%d%s", baseOffset, ".index"))
	indexFile, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}
	if s.index, err = newIndex(indexFile, cfg.MaxIndexBytes); err != nil {
		return nil, err
	}
	if off, _, err := s.index.Read(-1); err != nil {
		s.nextOffset = baseOffset
	} else {
		s.nextOffset = baseOffset + uint64(off) + 1
	}
	return s, nil
}

func (s *segment) Append(record *pb.Record) (offset uint64, err error) {
	cur := s.nextOffset
	record.Offset = cur
	p, err := proto.Marshal(record)
	if err != nil {
		return 0, err
	}
	_, pos, err := s.store.Append(p)
	if err != nil {
		return 0, err
	}
	// index offsets are relative to base offset
	if err := s.index.Write(uint32(s.nextOffset-s.baseOffset), pos); err != nil {
		return 0, err
	}
	s.nextOffset++
	return cur, nil
}

func (s *segment) Read(off uint64) (*pb.Record, error) {
	_, pos, err := s.index.Read(int64(off - s.baseOffset))
	if err != nil {
		return nil, err
	}
	p, err := s.store.Read(pos)
	if err != nil {
		return nil, err
	}
	record := &pb.Record{}
	err = proto.Unmarshal(p, record)
	return record, err
}

func (s *segment) IsMaxed() bool {
	return s.store.size >= s.config.MaxStoreBytes ||
		s.index.size >= s.config.MaxIndexBytes
}

func (s *segment) Close() error {
	if err := s.index.Close(); err != nil {
		return err
	}
	if err := s.store.Close(); err != nil {
		return err
	}
	return nil
}

func (s *segment) Remove() error {
	if err := s.Close(); err != nil {
		return err
	}
	if err := os.Remove(s.index.Name()); err != nil {
		return err
	}
	if err := os.Remove(s.store.Name()); err != nil {
		return err
	}
	return nil
}
