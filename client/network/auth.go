package network

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/userpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func getTransportCredentials(serverAddr string) credentials.TransportCredentials {
	if strings.HasSuffix(serverAddr, ":443") {
		return credentials.NewTLS(&tls.Config{})
	}
	return insecure.NewCredentials()
}

func Authenticate(server, email, password string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cleanServer := strings.TrimRight(strings.TrimPrefix(strings.TrimPrefix(server, "https://"), "http://"), "/")

	conn, err := grpc.NewClient(cleanServer, grpc.WithTransportCredentials(getTransportCredentials(cleanServer)))
	if err != nil {
		return "", "", fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	client := userpb.NewUserServiceClient(conn)
	resp, err := client.LoginUser(ctx, &userpb.LoginUserRequest{Email: email, Password: password})
	if err != nil {
		return "", "", fmt.Errorf("login failed: %w", err)
	}

	return resp.GetToken(), resp.GetRefreshToken(), nil
}

func RefreshSession(server, refreshToken string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cleanServer := strings.TrimRight(strings.TrimPrefix(strings.TrimPrefix(server, "https://"), "http://"), "/")

	conn, err := grpc.NewClient(cleanServer, grpc.WithTransportCredentials(getTransportCredentials(cleanServer)))
	if err != nil {
		return "", "", fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	client := userpb.NewUserServiceClient(conn)
	resp, err := client.RefreshToken(ctx, &userpb.RefreshTokenRequest{RefreshToken: refreshToken})
	if err != nil {
		return "", "", fmt.Errorf("refresh failed: %w", err)
	}
	return resp.GetToken(), resp.GetRefreshToken(), nil
}
