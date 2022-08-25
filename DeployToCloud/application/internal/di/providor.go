package di

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/soheilhy/cmux"

	"github.com/travisjeffery/proglog/internal/config"
	"github.com/travisjeffery/proglog/internal/grpc/auth"
	"github.com/travisjeffery/proglog/internal/log"
	"github.com/travisjeffery/proglog/internal/membership"
	"github.com/travisjeffery/proglog/internal/raft"
	innertls "github.com/travisjeffery/proglog/internal/tls"
)

func ProvideSegmentConfig(cfg *config.Env) log.Config {
	return log.Config{
		MaxStoreBytes: cfg.MaxStoreBytes,
		MaxIndexBytes: cfg.MaxIndexBytes,
		InitialOffset: cfg.InitialOffset,
	}
}

func ProvideACLArgs(cfg *config.Env) auth.Args {
	return auth.Args{
		ModelFile:  cfg.AclModelFile,
		PolicyFile: cfg.AclPolicyFile,
	}
}

func isBootstrap(nodeName string) bool {
	ss := strings.Split(nodeName, "-")
	return ss[len(ss)-1] == "0"
}

func ProvideRaftArgs(cfg *config.Env) raft.Args {
	return raft.Args{
		DataDir:            cfg.DataDir,
		NodeName:           cfg.NodeName,
		IsBootstrap:        isBootstrap(cfg.NodeName),
		BindAddr:           cfg.BindAddr,
		BootstrapTimeout:   cfg.BootstrapTimeout,
		HeartbeatTimeout:   cfg.HeartbeatTimeout,
		ElectionTimeout:    cfg.ElectionTimeout,
		LeaderLeaseTimeout: cfg.LeaderLeaseTimeout,
		CommitTimeout:      cfg.CommitTimeout,
	}
}

func ProvideMembershipArgs(cfg *config.Env) (membership.Args, error) {
	var as []string
	if !isBootstrap(cfg.NodeName) {
		as = cfg.StartJoinAddrs
	}
	host, _, err := net.SplitHostPort(cfg.BindAddr)
	if err != nil {
		return membership.Args{}, err
	}
	rpcAddr := fmt.Sprintf("%s:%d", host, cfg.RpcPort)
	return membership.Args{
		NodeName:       cfg.NodeName,
		Tags:           map[string]string{"rpc_addr": rpcAddr},
		BindAddr:       cfg.BindAddr,
		RPCAddr:        rpcAddr,
		StartJoinAddrs: as,
	}, nil
}

func ProvideInnerTLSConfig(cfg *config.Env) (innertls.Config, error) {
	var err error
	var serverTLSConfig *tls.Config
	if cfg.ServerTLSCertFile != "" && cfg.ServerTLSKeyFile != "" {
		serverTLSConfig, err = innertls.SetupTLS(innertls.Args{
			CertFile: cfg.ServerTLSCertFile,
			KeyFile:  cfg.ServerTLSKeyFile,
			CAFile:   cfg.ServerTLSCaFile,
			Server:   true,
		})
		if err != nil {
			return innertls.Config{}, err
		}
	}

	var peerTLSConfig *tls.Config
	if cfg.PeerTLSCertFile != "" && cfg.PeerTLSKeyFile != "" {
		peerTLSConfig, err = innertls.SetupTLS(innertls.Args{
			CertFile: cfg.PeerTLSCertFile,
			KeyFile:  cfg.PeerTLSKeyFile,
			CAFile:   cfg.PeerTLSCaFile,
		})
		if err != nil {
			return innertls.Config{}, err
		}
	}

	return innertls.Config{ServerTLSConfig: serverTLSConfig, PeerTLSConfig: peerTLSConfig}, nil
}

func ProvideTLSConfig(cfg innertls.Config) *tls.Config {
	return cfg.ServerTLSConfig
}

func ProvideLog(cfg log.Config) (*log.Log, error) {
	logDir := filepath.Join(cfg.DataDir, "log")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, err
	}
	cfg.DataDir = logDir
	return log.NewLog(cfg)
}

func ProvideMux(cfg *config.Env) (cmux.CMux, error) {
	addr, err := net.ResolveTCPAddr("tcp", cfg.BindAddr)
	if err != nil {
		return nil, err
	}
	rpcAddr := fmt.Sprintf("%s:%d", addr.IP.String(), cfg.RpcPort)
	ln, err := net.Listen("tcp", rpcAddr)
	if err != nil {
		return nil, err
	}
	return cmux.New(ln), nil
}

func ProvideLogStore(cfg *config.Env, l *log.Log) (*raft.LogStore, error) {
	logConfig := l.Config
	logConfig.InitialOffset = 1
	logDir := filepath.Join(cfg.DataDir, "raft", "log")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, err
	}
	return raft.NewLogStore(logConfig)
}
