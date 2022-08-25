package service

import (
	"sync"

	"github.com/soheilhy/cmux"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/travisjeffery/proglog/internal/membership"
	"github.com/travisjeffery/proglog/internal/raft"
)

type Service struct {
	mux        cmux.CMux
	raft       *raft.Raft
	server     *grpc.Server
	membership *membership.Membership

	shutdown     bool
	shutdowns    chan struct{}
	shutdownLock sync.Mutex

	logger *zap.Logger
}

func NewService(m cmux.CMux, r *raft.Raft, server *grpc.Server, mb *membership.Membership) *Service {
	return &Service{
		mux:        m,
		raft:       r,
		server:     server,
		membership: mb,
		shutdowns:  make(chan struct{}),
		logger:     zap.L().Named("service"),
	}
}

func (s *Service) Serve() {
	grpcLn := s.mux.Match(cmux.Any())
	go func() {
		if err := s.server.Serve(grpcLn); err != nil {
			s.logger.Error("failed grpc server", zap.Error(err))
		}
	}()
	go func() {
		if err := s.mux.Serve(); err != nil {
			s.logger.Error("failed service", zap.Error(err))
		}
	}()
}

func (s *Service) Shutdown() error {
	s.shutdownLock.Lock()
	defer s.shutdownLock.Unlock()
	if s.shutdown {
		return nil
	}
	s.shutdown = true
	close(s.shutdowns)

	if err := s.membership.Leave(); err != nil {
		return err
	}
	s.server.GracefulStop()
	return s.raft.Close()
}
