package raftapp

import (
	"bytes"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/travisjeffery/proglog/internal/log"
	pb "github.com/travisjeffery/proglog/internal/proto/v1"
	"github.com/travisjeffery/proglog/internal/raft"
)

type IResource interface {
	Append(*pb.Record) (uint64, error)
	Read(uint64) (*pb.Record, error)
}

type Resource struct {
	log  *log.Log
	raft *raft.Raft
}

func NewResource(l *log.Log, r *raft.Raft) *Resource {
	return &Resource{
		log:  l,
		raft: r,
	}
}

func (r *Resource) Append(record *pb.Record) (uint64, error) {
	res, err := r.apply(raft.AppendRequestType, &pb.ProduceRequest{Record: record})
	if err != nil {
		return 0, err
	}
	rs, ok := res.(*pb.ProduceResponse)
	if !ok {
		return 0, fmt.Errorf("failed to cast response %v", res)
	}

	return rs.Offset, nil
}

func (r *Resource) apply(reqType raft.RequestType, req proto.Message) (interface{}, error) {
	var buf bytes.Buffer
	_, err := buf.Write([]byte{byte(reqType)})
	if err != nil {
		return nil, err
	}
	b, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(b)
	if err != nil {
		return nil, err
	}
	timeout := 10 * time.Second
	future := r.raft.Apply(buf.Bytes(), timeout)
	if future.Error() != nil {
		return nil, future.Error()
	}
	res := future.Response()
	if err, ok := res.(error); ok {
		return nil, err
	}
	return res, nil
}

func (r *Resource) Read(offset uint64) (*pb.Record, error) {
	return r.log.Read(offset)
}
