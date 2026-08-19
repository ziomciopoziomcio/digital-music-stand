package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

var unprotectedMethods = map[string]bool{
	"/digital_music_stand.user.UserService/RegisterUser":  true,
	"/digital_music_stand.user.UserService/LoginUser":     true,
	"/digital_music_stand.user.UserService/ResetPassword": true,
}

func NewAuthInterceptor(secret string, db *gorm.DB) grpc.UnaryServerInterceptor {
	jwtSecretKey := []byte(secret)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if unprotectedMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata in request")
		}

		authHeader := md["authorization"]
		if len(authHeader) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "missing authorization token")
		}

		tokenString := strings.TrimPrefix(authHeader[0], "Bearer ")

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected sign method")
			}
			return jwtSecretKey, nil
		})

		if err != nil || !token.Valid {
			return nil, status.Errorf(codes.Unauthenticated, "invalid or expired token: %v", err)
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token claims")
		}

		userIDFloat, ok := claims["sub"].(float64)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing or invalid sub claim")
		}
		userID := uint(userIDFloat)

		var user models.User
		if err := db.Select("id", "status").First(&user, userID).Error; err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "user account no longer exists")
		}

		if user.Status != "active" {
			return nil, status.Errorf(codes.PermissionDenied, "account status is '%s' - access denied", user.Status)
		}
		ctx = context.WithValue(ctx, userIDKey, userID)

		return handler(ctx, req)
	}
}
