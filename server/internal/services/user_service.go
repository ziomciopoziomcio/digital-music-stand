package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/smtp"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/userpb"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/auth"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/models"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type UserService struct {
	userpb.UnimplementedUserServiceServer
	db            *gorm.DB
	jwtSecret     []byte
	tokenDuration time.Duration
}

func NewUserService(db *gorm.DB, secret string, durationHours int) *UserService {
	return &UserService{
		db:            db,
		jwtSecret:     []byte(secret),
		tokenDuration: time.Duration(durationHours) * time.Hour,
	}
}

func (s *UserService) generateTokens(userID uint, role string) (string, string, error) {
	accessClaims := jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"exp":  time.Now().Add(s.tokenDuration).Unix(),
		"type": "access",
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	refreshClaims := jwt.MapClaims{
		"sub":  userID,
		"exp":  time.Now().Add(time.Hour * 24 * 30).Unix(),
		"type": "refresh",
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	return accessTokenString, refreshTokenString, nil
}

func (s *UserService) RegisterUser(ctx context.Context, req *userpb.RegisterUserRequest) (*userpb.RegisterUserResponse, error) {
	if req.GetEmail() == "" || req.GetPassword() == "" || req.GetName() == "" || req.GetSurname() == "" {
		return nil, fmt.Errorf("missing required fields")
	}

	var existing models.User
	if err := s.db.Where("email = ?", req.GetEmail()).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("user with this email already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword()), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %v", err)
	}

	var totalUsers int64
	s.db.Model(&models.User{}).Count(&totalUsers)

	status := "pending"
	role := "user"
	if totalUsers == 0 {
		status = "active"
		role = "admin"
	}

	user := models.User{
		Email:        req.GetEmail(),
		PasswordHash: string(hashedPassword),
		Name:         req.GetName(),
		Surname:      req.GetSurname(),
		Status:       status,
		Role:         role,
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	msg := "Registration successful. Your account is awaiting administrator approval."
	if role == "admin" {
		msg = "First admin account created successfully."
	}

	return &userpb.RegisterUserResponse{
		Id:      uint32(user.ID),
		Message: msg,
	}, nil
}

func (s *UserService) LoginUser(ctx context.Context, req *userpb.LoginUserRequest) (*userpb.LoginUserResponse, error) {
	if req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, fmt.Errorf("missing required fields")
	}

	var user models.User
	if err := s.db.Where("email = ?", req.GetEmail()).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("invalid email or password")
		}
		return nil, fmt.Errorf("failed to query user: %v", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.GetPassword())); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	if user.Status == "pending" {
		return nil, fmt.Errorf("account is awaiting administrator approval")
	}
	if user.Status == "banned" {
		return nil, fmt.Errorf("account has been suspended")
	}
	if user.Status != "active" {
		return nil, fmt.Errorf("account is inactive")
	}

	tokenString, refreshTokenString, err := s.generateTokens(user.ID, user.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %v", err)
	}

	return &userpb.LoginUserResponse{
		Token:        tokenString,
		RefreshToken: refreshTokenString,
	}, nil
}

func (s *UserService) GetProfile(ctx context.Context, req *userpb.GetProfileRequest) (*userpb.GetProfileResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user id from context: %v", err)
	}

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}

	return &userpb.GetProfileResponse{
		Id:      uint32(user.ID),
		Email:   user.Email,
		Name:    user.Name,
		Surname: user.Surname,
	}, nil
}

func (s *UserService) GetDB() *gorm.DB {
	return s.db
}

func generateTempPassword(length int) string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, length)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func (s *UserService) ChangePassword(ctx context.Context, req *userpb.ChangePasswordRequest) (*userpb.ChangePasswordResponse, error) {
	var user models.User
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid old password")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to process new password")
	}

	s.db.Model(&user).Update("password_hash", string(hashedPassword))

	return &userpb.ChangePasswordResponse{Message: "Password changed successfully"}, nil
}

func (s *UserService) ResetPassword(ctx context.Context, req *userpb.ResetPasswordRequest) (*userpb.ResetPasswordResponse, error) {
	var user models.User
	if err := s.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return &userpb.ResetPasswordResponse{Message: "If the email exists, a recovery password has been sent."}, nil
	}

	tempPassword := generateTempPassword(8)
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)

	s.db.Model(&user).Update("password_hash", string(hashedPassword))

	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")

	if smtpHost != "" && smtpUser != "" && smtpPass != "" {
		auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

		to := []string{user.Email}

		msg := []byte("To: " + user.Email + "\r\n" +
			"Subject: Digital Music Stand - Password Reset\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
			"Hello,\r\n\n" +
			"A password reset was requested for your Digital Music Stand account.\r\n" +
			"Your temporary password is: " + tempPassword + "\r\n\n" +
			"Please log in to the application using this password and immediately change it in the Profile tab.\r\n")

		go func(recipient string, message []byte) {
			err := smtp.SendMail(smtpHost+":"+smtpPort, auth, smtpUser, to, message)
			if err != nil {
				log.Printf("[SMTP ERROR] Fail to send mail to %s: %v\n", recipient, err)
			} else {
				log.Printf("[SMTP SUCCESS] Sent new password to %s\n", recipient)
			}
		}(user.Email, msg)
	} else {
		log.Println("[SMTP WARNING] No SMTP configuration found. Skipping email sending.")
		log.Printf("[DEBUG] Password for %s is: %s\n", user.Email, tempPassword)
	}

	return &userpb.ResetPasswordResponse{Message: "If the email exists, a recovery password has been sent."}, nil
}

func (s *UserService) RefreshToken(ctx context.Context, req *userpb.RefreshTokenRequest) (*userpb.RefreshTokenResponse, error) {
	token, err := jwt.Parse(req.GetRefreshToken(), func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected sign method")
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, status.Errorf(codes.Unauthenticated, "invalid or expired refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token claims")
	}

	if tokenType, ok := claims["type"].(string); !ok || tokenType != "refresh" {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token type")
	}

	userIDFloat, ok := claims["sub"].(float64)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "missing sub claim")
	}
	userID := uint(userIDFloat)

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not found")
	}
	if user.Status != "active" {
		return nil, status.Errorf(codes.PermissionDenied, "account is inactive")
	}

	newAccess, newRefresh, err := s.generateTokens(user.ID, user.Role)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate tokens")
	}

	return &userpb.RefreshTokenResponse{
		Token:        newAccess,
		RefreshToken: newRefresh,
	}, nil
}
