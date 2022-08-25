package raftapp

import (
	"github.com/hashicorp/raft"

	innerraft "github.com/travisjeffery/proglog/internal/raft"
)

type IMembershipHandler interface {
	Join(name, addr string) error
	Leave(name string) error
}

type MembershipHandler struct {
	raft *innerraft.Raft
}

func NewMembershipHandler(r *innerraft.Raft) *MembershipHandler {
	return &MembershipHandler{raft: r}
}

func (h *MembershipHandler) Join(id, addr string) error {
	configFuture := h.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return err
	}
	serverID := raft.ServerID(id)
	serverAddr := raft.ServerAddress(addr)
	for _, srv := range configFuture.Configuration().Servers {
		if (srv.ID == serverID && srv.Address != serverAddr) || (srv.ID != serverID && srv.Address == serverAddr) {
			// remove the existing server
			removeFuture := h.raft.RemoveServer(serverID, 0, 0)
			if err := removeFuture.Error(); err != nil {
				return err
			}
		}
	}
	addFuture := h.raft.AddVoter(serverID, serverAddr, 0, 0)
	if err := addFuture.Error(); err != nil {
		return err
	}
	return nil
}

func (h *MembershipHandler) Leave(id string) error {
	removeFuture := h.raft.RemoveServer(raft.ServerID(id), 0, 0)
	return removeFuture.Error()
}
