package raft

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"

	"github.com/travisjeffery/proglog/internal/log"
)

type Raft struct {
	*raft.Raft
	log *log.Log
}

type Args struct {
	DataDir            string
	NodeName           string
	IsBootstrap        bool
	BindAddr           string
	BootstrapTimeout   time.Duration
	HeartbeatTimeout   time.Duration
	ElectionTimeout    time.Duration
	LeaderLeaseTimeout time.Duration
	CommitTimeout      time.Duration
}

func NewRaft(l *log.Log, logStore *LogStore, sl *StreamLayer, args Args) (*Raft, error) {
	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(args.DataDir, "raft", "stable"))
	if err != nil {
		return nil, err
	}
	snapshotStore, err := raft.NewFileSnapshotStore(filepath.Join(args.DataDir, "raft"), 1, os.Stderr)
	if err != nil {
		return nil, err
	}

	transport := raft.NewNetworkTransport(sl, 5, 10*time.Second, os.Stderr)
	r, err := raft.NewRaft(setupConfig(args), &FSM{Log: l}, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, err
	}
	rf := &Raft{Raft: r, log: l}
	if !args.IsBootstrap {
		return rf, nil
	}
	if err := rf.bootstrap(args); err != nil {
		return nil, err
	}
	return rf, nil
}

func setupConfig(args Args) *raft.Config {
	c := raft.DefaultConfig()
	c.LocalID = raft.ServerID(args.NodeName)
	if args.HeartbeatTimeout != 0 {
		c.HeartbeatTimeout = args.HeartbeatTimeout
	}
	if args.ElectionTimeout != 0 {
		c.ElectionTimeout = args.ElectionTimeout
	}
	if args.LeaderLeaseTimeout != 0 {
		c.LeaderLeaseTimeout = args.LeaderLeaseTimeout
	}
	if args.CommitTimeout != 0 {
		c.CommitTimeout = args.CommitTimeout
	}
	return c
}

func (r *Raft) bootstrap(args Args) error {
	c := raft.Configuration{
		Servers: []raft.Server{{
			ID:      raft.ServerID(args.NodeName),
			Address: raft.ServerAddress(args.BindAddr),
		}},
	}
	if err := r.BootstrapCluster(c).Error(); err != nil {
		return err
	}
	return r.waitForLeader(args.BootstrapTimeout)
}

func (r *Raft) waitForLeader(timeout time.Duration) error {
	timeoutc := time.After(timeout)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-timeoutc:
			return fmt.Errorf("timed out")
		case <-ticker.C:
			if l := r.Leader(); l != "" {
				return nil
			}
		}
	}
}

func (r *Raft) Close() error {
	f := r.Shutdown()
	if err := f.Error(); err != nil {
		return err
	}
	return r.log.Close()
}
