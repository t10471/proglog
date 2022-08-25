package server

import (
	"context"
	"errors"

	"github.com/travisjeffery/proglog/internal/grpc/auth"
	"github.com/travisjeffery/proglog/internal/log"
	pb "github.com/travisjeffery/proglog/internal/proto/v1"
	"github.com/travisjeffery/proglog/internal/raftapp"
)

type service struct {
	CommitLog   raftapp.IResource
	Authorizer  auth.IAuthorizer
	GetServerer raftapp.IServers
	pb.UnimplementedLogServer
}

func newService(commitLog raftapp.IResource, authorizable auth.IAuthorizer, getServerer raftapp.IServers) *service {
	srv := &service{
		CommitLog:   commitLog,
		Authorizer:  authorizable,
		GetServerer: getServerer,
	}
	return srv
}

func (s *service) Produce(ctx context.Context, req *pb.ProduceRequest) (*pb.ProduceResponse, error) {
	if err := s.Authorizer.Authorize(subject(ctx), objectWildcard, produceAction); err != nil {
		return nil, err
	}
	offset, err := s.CommitLog.Append(req.Record)
	if err != nil {
		return nil, err
	}
	return &pb.ProduceResponse{Offset: offset}, nil
}

func (s *service) Consume(ctx context.Context, req *pb.ConsumeRequest) (*pb.ConsumeResponse, error) {
	if err := s.Authorizer.Authorize(subject(ctx), objectWildcard, consumeAction); err != nil {
		return nil, err
	}
	record, err := s.CommitLog.Read(req.Offset)
	if err != nil {
		if errors.As(err, &log.OffsetOutOfRangeError{}) {
			//nolint:errorlint,forcetypeassert //reason: false positive
			return nil, OffsetOutOfRangeError{err.(log.OffsetOutOfRangeError).Offset}
		}
		return nil, err
	}
	return &pb.ConsumeResponse{Record: record}, nil
}

func (s *service) ProduceStream(stream pb.Log_ProduceStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}
		res, err := s.Produce(stream.Context(), req)
		if err != nil {
			return err
		}
		if err := stream.Send(res); err != nil {
			return err
		}
	}
}

func (s *service) ConsumeStream(req *pb.ConsumeRequest, stream pb.Log_ConsumeStreamServer) error {
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			res, err := s.Consume(stream.Context(), req)
			//nolint:errorlint //reason: false positive
			switch err.(type) {
			case nil:
			case OffsetOutOfRangeError:
				continue
			default:
				return err
			}
			if err := stream.Send(res); err != nil {
				return err
			}
			req.Offset++
		}
	}
}

func (s *service) GetServers(ctx context.Context, req *pb.GetServersRequest) (*pb.GetServersResponse, error) {
	servers, err := s.GetServerer.GetServers()
	if err != nil {
		return nil, err
	}
	return &pb.GetServersResponse{Servers: servers}, nil
}
