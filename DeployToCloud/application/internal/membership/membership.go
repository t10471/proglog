package membership

import (
	"errors"
	"net"

	"github.com/hashicorp/raft"
	"github.com/hashicorp/serf/serf"
	"go.uber.org/zap"

	"github.com/travisjeffery/proglog/internal/raftapp"
)

type Membership struct {
	handler raftapp.IMembershipHandler
	serf    *serf.Serf
	events  chan serf.Event
	logger  *zap.Logger
}

type Args struct {
	NodeName       string
	Tags           map[string]string
	BindAddr       string
	RPCAddr        string
	StartJoinAddrs []string
}

func NewMembership(handler raftapp.IMembershipHandler, args Args) (*Membership, error) {
	c := &Membership{
		handler: handler,
		logger:  zap.L().Named("membership"),
	}

	addr, err := net.ResolveTCPAddr("tcp", args.BindAddr)
	if err != nil {
		return nil, err
	}
	c.events = make(chan serf.Event)
	c.serf, err = serf.Create(makeConfig(addr, c.events, args))
	if err != nil {
		return nil, err
	}
	go c.eventHandler()
	if args.StartJoinAddrs == nil {
		return c, nil
	}
	_, err = c.serf.Join(args.StartJoinAddrs, true)
	return c, err
}

func makeConfig(addr *net.TCPAddr, eventCh chan serf.Event, args Args) *serf.Config {
	c := serf.DefaultConfig()
	c.Init()
	c.MemberlistConfig.BindAddr = addr.IP.String()
	c.MemberlistConfig.BindPort = addr.Port
	c.EventCh = eventCh
	c.Tags = args.Tags
	c.NodeName = args.NodeName
	return c
}

func (m *Membership) eventHandler() {
	for e := range m.events {
		//nolint:exhaustive //reason: only needed type
		switch e.EventType() {
		case serf.EventMemberJoin:
			//nolint:forcetypeassert //reason: explicit
			for _, member := range e.(serf.MemberEvent).Members {
				if !m.isLocal(member) {
					if err := m.handler.Join(member.Name, member.Tags["rpc_addr"]); err != nil {
						m.logError(err, "failed to join", member)
					}
				}
			}
		case serf.EventMemberLeave, serf.EventMemberFailed:
			//nolint:forcetypeassert //reason: for explicit
			for _, member := range e.(serf.MemberEvent).Members {
				if !m.isLocal(member) {
					if err := m.handler.Leave(member.Name); err != nil {
						m.logError(err, "failed to leave", member)
					}
				}
			}
		}
	}
}

func (m *Membership) isLocal(member serf.Member) bool {
	return m.serf.LocalMember().Name == member.Name
}

func (m *Membership) Members() []serf.Member {
	return m.serf.Members()
}

func (m *Membership) Leave() error {
	return m.serf.Leave()
}

func (m *Membership) logError(err error, msg string, member serf.Member) {
	log := m.logger.Error
	if errors.Is(err, raft.ErrNotLeader) {
		log = m.logger.Debug
	}
	log(msg, zap.Error(err), zap.String("name", member.Name), zap.String("rpc_addr", member.Tags["rpc_addr"]))
}
