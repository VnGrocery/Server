package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrUnauthorized = errors.New("missing or invalid authentication token")

type Principal struct {
	UserID string `json:"userId"`
	Email  string `json:"email,omitempty"`
}

type Verifier interface {
	Verify(ctx context.Context, token string) (Principal, error)
}

type Issuer interface {
	IssueToken(principal Principal, ttl time.Duration) (string, error)
}

type JWTService struct {
	secret []byte
	issuer string
}

func NewJWTService(secret, issuer string) *JWTService {
	return &JWTService{
		secret: []byte(secret),
		issuer: issuer,
	}
}

func (s *JWTService) Verify(ctx context.Context, token string) (Principal, error) {
	_ = ctx

	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrUnauthorized
		}
		return s.secret, nil
	}, jwt.WithIssuer(s.issuer))
	if err != nil {
		return Principal{}, ErrUnauthorized
	}
	if !parsed.Valid {
		return Principal{}, ErrUnauthorized
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return Principal{}, ErrUnauthorized
	}

	userID, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	if userID == "" {
		return Principal{}, ErrUnauthorized
	}

	return Principal{UserID: userID, Email: email}, nil
}

func (s *JWTService) IssueToken(principal Principal, ttl time.Duration) (string, error) {
	if principal.UserID == "" {
		return "", fmt.Errorf("userId is required")
	}
	if ttl <= 0 {
		return "", fmt.Errorf("ttl must be positive")
	}

	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"iss":   s.issuer,
		"sub":   principal.UserID,
		"email": principal.Email,
		"iat":   now.Unix(),
		"exp":   now.Add(ttl).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}
