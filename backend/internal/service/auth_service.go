package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"

	"github.com/npinot/vibe/backend/internal/config"
	"github.com/npinot/vibe/backend/internal/model"
	"github.com/npinot/vibe/backend/internal/repository"
)

type AuthService interface {
	GetAuthorizationURL(state string) (string, error)
	ExchangeCodeForToken(ctx context.Context, code string) (*model.User, string, error)
	GenerateJWT(user *model.User) (string, error)
}

type authService struct {
	cfg          *config.Config
	provider     *oidc.Provider
	oauth2Config *oauth2.Config
	verifier     *oidc.IDTokenVerifier
	userRepo     repository.UserRepository
}

func NewAuthService(cfg *config.Config, userRepo repository.UserRepository) (AuthService, error) {
	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, cfg.OIDCIssuer)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	oauth2Config := &oauth2.Config{
		ClientID:     cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret,
		RedirectURL:  cfg.OIDCRedirectURI,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.OIDCClientID})

	return &authService{
		cfg:          cfg,
		provider:     provider,
		oauth2Config: oauth2Config,
		verifier:     verifier,
		userRepo:     userRepo,
	}, nil
}

func (s *authService) GetAuthorizationURL(state string) (string, error) {
	if state == "" {
		stateBytes := make([]byte, 32)
		if _, err := rand.Read(stateBytes); err != nil {
			return "", fmt.Errorf("failed to generate state: %w", err)
		}
		state = base64.URLEncoding.EncodeToString(stateBytes)
	}

	return s.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

func (s *authService) ExchangeCodeForToken(ctx context.Context, code string) (*model.User, string, error) {
	oauth2Token, err := s.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, "", fmt.Errorf("failed to exchange code: %w", err)
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return nil, "", fmt.Errorf("no id_token in token response")
	}

	idToken, err := s.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to verify ID token: %w", err)
	}

	var claims struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		EmailVerified bool   `json:"email_verified"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return nil, "", fmt.Errorf("failed to parse claims: %w", err)
	}

	user, err := s.userRepo.CreateOrUpdateFromOIDC(ctx, claims.Sub, claims.Email, claims.Name, claims.Picture)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create or update user: %w", err)
	}

	jwtToken, err := s.GenerateJWT(user)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	return user, jwtToken, nil
}

func (s *authService) GenerateJWT(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID.String(),
		"sub":     user.OIDCSubject,
		"email":   user.Email,
		"name":    user.Name,
		"exp":     time.Now().Add(time.Duration(s.cfg.JWTExpiry) * time.Second).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}
