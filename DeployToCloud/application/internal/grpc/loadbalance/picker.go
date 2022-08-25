package loadbalance

import (
	"strings"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
)

func init() {
	balancer.Register(base.NewBalancerBuilder(Name, &Picker{}, base.Config{}))
}

type Picker struct {
	mu        sync.RWMutex
	leader    balancer.SubConn
	followers []balancer.SubConn
	current   uint64
	logger    *zap.Logger
}

var (
	_ base.PickerBuilder = (*Picker)(nil)
	_ balancer.Picker    = (*Picker)(nil)
)

func (p *Picker) Build(buildInfo base.PickerBuildInfo) balancer.Picker {
	p.logger = zap.L().Named("picker")
	p.mu.Lock()
	defer p.mu.Unlock()
	followers := make([]balancer.SubConn, 0, len(buildInfo.ReadySCs))
	for sc, scInfo := range buildInfo.ReadySCs {
		isLeader, ok := scInfo.Address.Attributes.Value("is_leader").(bool)
		if !ok {
			p.logger.Warn("not found attributes is_leader",
				zap.String("address", scInfo.Address.Addr),
				zap.String("server_name", scInfo.Address.ServerName))
			followers = append(followers, sc)
			continue
		}
		if isLeader {
			p.leader = sc
			continue
		}
		followers = append(followers, sc)
	}
	p.followers = followers
	return p
}

func (p *Picker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var result balancer.PickResult
	if strings.Contains(info.FullMethodName, "Produce") || len(p.followers) == 0 {
		result.SubConn = p.leader
	} else if strings.Contains(info.FullMethodName, "Consume") {
		result.SubConn = p.nextFollower()
	}
	if result.SubConn == nil {
		return result, balancer.ErrNoSubConnAvailable
	}
	return result, nil
}

func (p *Picker) nextFollower() balancer.SubConn {
	cur := atomic.AddUint64(&p.current, uint64(1))
	idx := int(cur % uint64(len(p.followers)))
	return p.followers[idx]
}
