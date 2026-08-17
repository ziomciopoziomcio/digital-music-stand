package network

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TokenAuth struct {
	Token string
}

func (t TokenAuth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + t.Token,
	}, nil
}

func (t TokenAuth) RequireTransportSecurity() bool {
	return false // todo: use TLS
}

func NewGRPCClient(serverAddr, token string) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(TokenAuth{Token: token}),
	}
	return grpc.NewClient(serverAddr, opts...)
}
