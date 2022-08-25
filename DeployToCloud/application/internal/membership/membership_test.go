package membership_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/serf/serf"
	"github.com/stretchr/testify/require"
	"github.com/travisjeffery/go-dynaport"

	. "github.com/travisjeffery/proglog/internal/membership"
)

func TestMembership(t *testing.T) {
	ports := dynaport.Get(3)
	m, handler := setupMembership(t, ports[0], nil, 0)
	m, _ = setupMembership(t, ports[1], m, ports[0])
	m, _ = setupMembership(t, ports[2], m, ports[0])

	require.Eventually(t, func() bool {
		return len(handler.joins) == 2 &&
			len(m[0].Members()) == 3 &&
			len(handler.leaves) == 0
	}, 3*time.Second, 250*time.Millisecond)

	require.NoError(t, m[2].Leave())

	require.Eventually(t, func() bool {
		return len(handler.joins) == 2 &&
			len(m[0].Members()) == 3 &&
			serf.StatusLeft == m[0].Members()[2].Status &&
			len(handler.leaves) == 1
	}, 3*time.Second, 250*time.Millisecond)

	require.Equal(t, fmt.Sprintf("%d", 2), <-handler.leaves)
}

func setupMembership(t *testing.T, port int, members []*Membership, joinPort int) ([]*Membership, *handler) {
	t.Helper()
	addr := fmt.Sprintf("%s:%d", "127.0.0.1", port)
	id := len(members)
	tags := map[string]string{"rpc_addr": addr}
	args := Args{
		NodeName: fmt.Sprintf("%d", id),
		BindAddr: addr,
		RPCAddr:  addr,
		Tags:     tags,
	}
	h := &handler{}
	if len(members) == 0 {
		h.joins = make(chan map[string]string, 3)
		h.leaves = make(chan string, 3)
	} else {
		args.StartJoinAddrs = []string{
			fmt.Sprintf("%s:%d", "127.0.0.1", joinPort),
		}
	}
	m, err := NewMembership(h, args)
	require.NoError(t, err)
	members = append(members, m)
	return members, h
}

type handler struct {
	joins  chan map[string]string
	leaves chan string
}

func (h *handler) Join(id, addr string) error {
	if h.joins != nil {
		h.joins <- map[string]string{
			"id":   id,
			"addr": addr,
		}
	}
	return nil
}

func (h *handler) Leave(id string) error {
	if h.leaves != nil {
		h.leaves <- id
	}
	return nil
}
