package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

type Args struct {
	CertFile string
	KeyFile  string
	CAFile   string
	Server   bool
}

func SetupTLS(args Args) (*tls.Config, error) {
	if args.CertFile != "" && args.KeyFile != "" {
		c, err := tls.LoadX509KeyPair(args.CertFile, args.KeyFile)
		if err != nil {
			return nil, err
		}
		return &tls.Config{Certificates: []tls.Certificate{c}}, nil
	}
	if args.CAFile == "" {
		return &tls.Config{}, nil
	}
	b, err := os.ReadFile(args.CAFile)
	if err != nil {
		return nil, err
	}
	ca := x509.NewCertPool()
	if !ca.AppendCertsFromPEM(b) {
		return nil, fmt.Errorf("failed to parse root certificate: %q", args.CAFile)
	}
	if args.Server {
		return &tls.Config{ClientCAs: ca, ClientAuth: tls.RequireAndVerifyClientCert}, nil
	} else {
		return &tls.Config{RootCAs: ca}, nil
	}
}
