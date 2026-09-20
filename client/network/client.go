package network

import (
	"context"
	"crypto/tls"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type TokenAuth struct {
	Token  string
	Secure bool
}

func (t TokenAuth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + t.Token,
	}, nil
}

func (t TokenAuth) RequireTransportSecurity() bool {
	return t.Secure
}

func NewGRPCClient(serverAddr, token string) (*grpc.ClientConn, error) {
	cleanAddr := sanitizeAddress(serverAddr)
	isSecure := strings.HasSuffix(cleanAddr, ":443")

	var creds credentials.TransportCredentials
	if isSecure {
		creds = credentials.NewTLS(&tls.Config{})
	} else {
		creds = insecure.NewCredentials()
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(TokenAuth{
			Token:  token,
			Secure: isSecure,
		}),
	}
	return grpc.NewClient(cleanAddr, opts...)
}

func sanitizeAddress(addr string) string {
	addr = strings.TrimPrefix(addr, "https://")
	addr = strings.TrimPrefix(addr, "http://")
	return strings.TrimRight(addr, "/")
}
