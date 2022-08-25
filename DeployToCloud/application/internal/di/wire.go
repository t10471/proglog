//go:build wireinject
// +build wireinject

package di

import (
	"github.com/google/wire"

	"github.com/travisjeffery/proglog/internal/config"
	"github.com/travisjeffery/proglog/internal/grpc/auth"
	"github.com/travisjeffery/proglog/internal/grpc/server"
	"github.com/travisjeffery/proglog/internal/membership"
	"github.com/travisjeffery/proglog/internal/raft"
	"github.com/travisjeffery/proglog/internal/raftapp"
	"github.com/travisjeffery/proglog/internal/service"
)

var raftSet = wire.NewSet(
	ProvideLogStore,
	ProvideInnerTLSConfig,
	ProvideMux,
	ProvideRaftArgs,
	raft.NewStreamLayer,
	raft.NewFSM,
	raft.NewRaft,
)

func InitializeService(env *config.Env) (*service.Service, error) {
	wire.Build(
		ProvideSegmentConfig,
		ProvideLog,
		raftSet,
		raftapp.NewResource,
		raftapp.NewGetServers,
		server.NewGRPCServer,
		raftapp.NewMembershipHandler,
		ProvideACLArgs,
		auth.NewAuthorizer,
		ProvideMembershipArgs,
		ProvideTLSConfig,
		membership.NewMembership,
		wire.Bind(new(raftapp.IResource), new(*raftapp.Resource)),
		wire.Bind(new(raftapp.IMembershipHandler), new(*raftapp.MembershipHandler)),
		wire.Bind(new(raftapp.IServers), new(*raftapp.Servers)),
		wire.Bind(new(auth.IAuthorizer), new(*auth.Authorizer)),
		service.NewService,
	)
	return nil, nil
}
