package raft

import (
	"bytes"
	"errors"
	"io"

	"github.com/hashicorp/raft"
	"google.golang.org/protobuf/proto"

	"github.com/travisjeffery/proglog/internal/log"
	pb "github.com/travisjeffery/proglog/internal/proto/v1"
)

const (
	AppendRequestType RequestType = 0
)

type FSM struct {
	Log *log.Log
}

type RequestType uint8

var _ raft.FSM = (*FSM)(nil)

func NewFSM(l *log.Log) *FSM {
	return &FSM{Log: l}
}

func (f *FSM) Apply(record *raft.Log) interface{} {
	buf := record.Data
	reqType := RequestType(buf[0])
	//nolint:gocritic //reason: for productivity
	switch reqType {
	case AppendRequestType:
		return f.applyAppend(buf[1:])
	}
	return nil
}

func (f *FSM) applyAppend(b []byte) interface{} {
	var req pb.ProduceRequest
	err := proto.Unmarshal(b, &req)
	if err != nil {
		return err
	}
	offset, err := f.Log.Append(req.Record)
	if err != nil {
		return err
	}
	return &pb.ProduceResponse{Offset: offset}
}

func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	r := f.Log.Reader()
	return &snapshot{reader: r}, nil
}

func (f *FSM) Restore(r io.ReadCloser) error {
	if err := f.Log.Reset(); err != nil {
		return err
	}
	b := make([]byte, log.LenWidth)
	var buf bytes.Buffer
	for i := 0; ; i++ {
		_, err := io.ReadFull(r, b)
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		}
		size := int64(log.Enc.Uint64(b))
		if _, err = io.CopyN(&buf, r, size); err != nil {
			return err
		}
		record := &pb.Record{}
		if err := proto.Unmarshal(buf.Bytes(), record); err != nil {
			return err
		}
		if i == 0 {
			f.Log.Config.InitialOffset = record.Offset
			if err := f.Log.Reset(); err != nil {
				return err
			}
		}
		if _, err = f.Log.Append(record); err != nil {
			return err
		}
		buf.Reset()
	}
	return nil
}

var _ raft.FSMSnapshot = (*snapshot)(nil)

type snapshot struct {
	reader io.Reader
}

func (s *snapshot) Persist(sink raft.SnapshotSink) error {
	if _, err := io.Copy(sink, s.reader); err != nil {
		//nolint:errcheck //reason: error already exists
		_ = sink.Cancel()
		return err
	}
	return sink.Close()
}

func (s *snapshot) Release() {}
