package config

import (
	"time"

	"go.uber.org/zap/zapcore"
)

const (
	Production = "production"
	Local      = "local"
)

type Env struct {
	Environment    string        `env:"ENVIRONMENT,required"`
	LogLevel       zapcore.Level `envconfig:"LOG_LEVEL" default:"INFO"`
	DataDir        string        `env:"DATA_DIR,required"`
	NodeName       string        `env:"HOST_NAME,required"`
	RpcPort        int           `env:"RPC_PORT,default=8400"`
	BindAddr       string        `env:"HOST_NAME,default=127.0.0.1:8401"`
	StartJoinAddrs []string      `env:"START_JOIN_ADDRS"`

	AclModelFile  string `env:"ACL_MODEL_FILE"`
	AclPolicyFile string `env:"ACL_POLICY_FILE"`

	ServerTLSCertFile string `env:"SERVER_TLS_CERT_FILE"`
	ServerTLSKeyFile  string `env:"SERVER_TLS_KEY_FILE"`
	ServerTLSCaFile   string `env:"SERVER_TLS_CA_FILE"`

	PeerTLSCertFile string `env:"PEER_TLS_CERT_FILE"`
	PeerTLSKeyFile  string `env:"PEER_TLS_KEY_FILE"`
	PeerTLSCaFile   string `env:"PEER_TLS_CA_FILE"`

	MaxStoreBytes uint64 `env:"MAX_STORE_BYTES"`
	MaxIndexBytes uint64 `env:"MAX_INDEX_BYTES"`
	InitialOffset uint64 `env:"INITIAL_OFFSET,default=1"`

	BootstrapTimeout   time.Duration `env:"BOOTSTRAP_TIMEOUT,default=3s"`
	HeartbeatTimeout   time.Duration `env:"HEARTBEAT_TIMEOUT"`
	ElectionTimeout    time.Duration `env:"ELECTION_TIMEOUT"`
	LeaderLeaseTimeout time.Duration `env:"LEADER_LEASE_TIMEOUT"`
	CommitTimeout      time.Duration `env:"COMMIT_TIMEOUT"`
}
