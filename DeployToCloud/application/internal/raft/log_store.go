package raft

import (
	"github.com/hashicorp/raft"

	"github.com/travisjeffery/proglog/internal/log"
	pb "github.com/travisjeffery/proglog/internal/proto/v1"
)

type LogStore struct {
	*log.Log
}

var _ raft.LogStore = (*LogStore)(nil)

func NewLogStore(cfg log.Config) (*LogStore, error) {
	l, err := log.NewLog(cfg)
	if err != nil {
		return nil, err
	}
	return &LogStore{l}, nil
}

func (l *LogStore) FirstIndex() (uint64, error) {
	return l.LowestOffset()
}

func (l *LogStore) LastIndex() (uint64, error) {
	off, err := l.HighestOffset()
	return off, err
}

func (l *LogStore) GetLog(index uint64, out *raft.Log) error {
	in, err := l.Read(index)
	if err != nil {
		return err
	}
	out.Data = in.Value
	out.Index = in.Offset
	out.Type = raft.LogType(in.Type)
	out.Term = in.Term
	return nil
}

func (l *LogStore) StoreLog(record *raft.Log) error {
	return l.StoreLogs([]*raft.Log{record})
}

func (l *LogStore) StoreLogs(records []*raft.Log) error {
	for _, record := range records {
		if _, err := l.Append(&pb.Record{
			Value: record.Data,
			Term:  record.Term,
			Type:  uint32(record.Type),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (l *LogStore) DeleteRange(min, max uint64) error {
	return l.Truncate(max)
}
