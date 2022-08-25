package raftapp

import (
	pb "github.com/travisjeffery/proglog/internal/proto/v1"
	"github.com/travisjeffery/proglog/internal/raft"
)

type IServers interface {
	GetServers() ([]*pb.Server, error)
}

type Servers struct {
	raft *raft.Raft
}

func NewGetServers(r *raft.Raft) *Servers {
	return &Servers{raft: r}
}

func (s *Servers) GetServers() ([]*pb.Server, error) {
	future := s.raft.GetConfiguration()
	if err := future.Error(); err != nil {
		return nil, err
	}
	servers := make([]*pb.Server, 0, len(future.Configuration().Servers))
	for _, server := range future.Configuration().Servers {
		servers = append(servers, &pb.Server{
			Id:       string(server.ID),
			RpcAddr:  string(server.Address),
			IsLeader: s.raft.Leader() == server.Address,
		})
	}
	return servers, nil
}
