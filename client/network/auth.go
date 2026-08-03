package network

import (
	"context"
	"fmt"
	"time"

	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/userpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Authenticate(server, email, password string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(server, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", fmt.Errorf("connection failed")
	}
	defer conn.Close()

	client := userpb.NewUserServiceClient(conn)
	resp, err := client.LoginUser(ctx, &userpb.LoginUserRequest{Email: email, Password: password})
	if err != nil {
		return "", fmt.Errorf("login failed")
	}
	return resp.GetToken(), nil
}
