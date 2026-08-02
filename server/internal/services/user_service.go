package services

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/userpb"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/auth"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/models"
	"golang.org/x/crypto/bcrypt"
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

func (s *UserService) generateJWT(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(s.tokenDuration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *UserService) RegisterUser(ctx context.Context, req *userpb.RegisterUserRequest) (*userpb.RegisterUserResponse, error) {
	if req.GetEmail() == "" || req.GetPassword() == "" || req.GetName() == "" || req.GetSurname() == "" {
		return nil, fmt.Errorf("missing required fields")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword()), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %v", err)
	}

	newUser := models.User{
		PasswordHash: string(hashedPassword),
		Email:        req.GetEmail(),
		Name:         req.GetName(),
		Surname:      req.GetSurname(),
	}

	if err := s.db.Create(&newUser).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	return &userpb.RegisterUserResponse{
		Id:      uint32(newUser.ID),
		Message: "User registered successfully",
	}, nil
}

func (s *UserService) LoginUser(ctx context.Context, req *userpb.LoginUserRequest) (*userpb.LoginUserResponse, error) {
	if req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, fmt.Errorf("missing required fields")
	}

	var user models.User
	if err := s.db.Where("email = ?", req.GetEmail()).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		} //todo: error.is
		return nil, fmt.Errorf("failed to query user: %v", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.GetPassword())); err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	tokenString, err := s.generateJWT(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %v", err)
	}

	return &userpb.LoginUserResponse{
		Token: tokenString,
	}, nil
}

func (s *UserService) GetProfile(ctx context.Context, req *userpb.GetProfileRequest) (*userpb.GetProfileResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user id from context: %v", err)
	}

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to query user: %v", err)
	}

	return &userpb.GetProfileResponse{
		Id:      uint32(user.ID),
		Name:    user.Name,
		Surname: user.Surname,
		Email:   user.Email,
	}, nil
}
