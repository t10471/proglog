package tls

import (
	"crypto/tls"
)

type Config struct {
	ServerTLSConfig *tls.Config
	PeerTLSConfig   *tls.Config
}
