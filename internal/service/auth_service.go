package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"ai-ba/internal/domain/models"
	"ai-ba/internal/repository"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"
)

type AuthService struct {
	users          *repository.UserRepo
	jwtSecret      string
	tokenTTL       time.Duration
	googleClientID string
}

func NewAuthService(ur *repository.UserRepo) *AuthService {
	secret := os.Getenv("JWT_SECRET")
	ttl := 24 * time.Hour
	return &AuthService{
		users:          ur,
		jwtSecret:      secret,
		tokenTTL:       ttl,
		googleClientID: os.Getenv("GOOGLE_CLIENT_ID"),
	}
}

func (s *AuthService) Register(email, password, name string) (string, error) {
	// check exists
	if _, err := s.users.FindByEmail(email); err == nil {
		return "", errors.New("user already exists")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	u := &models.User{
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
		Provider:     "local",
	}
	if err := s.users.Create(u); err != nil {
		return "", err
	}
	// generate token
	return s.GenerateToken(u)
}

func (s *AuthService) Login(email, password string) (string, error) {
	u, err := s.users.FindByEmail(email)
	if err != nil {
		return "", errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}
	return s.GenerateToken(u)
}

func (s *AuthService) LoginWithGoogle(ctx context.Context, idToken string) (string, error) {
	if s.googleClientID == "" {
		return "", errors.New("google auth not configured")
	}
	payload, err := idtoken.Validate(ctx, idToken, s.googleClientID)
	if err != nil {
		return "", err
	}
	email := ""
	if e, ok := payload.Claims["email"].(string); ok {
		email = e
	}
	sub := payload.Subject
	name := ""
	if n, ok := payload.Claims["name"].(string); ok {
		name = n
	}

	// Upsert user by provider
	u := &models.User{
		Email:      email,
		Name:       name,
		Provider:   "google",
		ProviderID: sub,
	}
	user, err := s.users.UpsertByProvider("google", sub, u)
	if err != nil {
		return "", err
	}
	return s.GenerateToken(user)
}

func (s *AuthService) GenerateToken(u *models.User) (string, error) {
	if s.jwtSecret == "" {
		// no jwt configured: return empty token but it's not ideal
		fmt.Println("jwt not")
		return "", errors.New("jwt not configured")
	}
	claims := jwt.MapClaims{
		"sub":   u.ID,
		"email": u.Email,
		"name":  u.Name,
		"exp":   time.Now().Add(s.tokenTTL).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
