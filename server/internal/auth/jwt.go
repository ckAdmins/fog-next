package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/nemvince/fog-next/internal/config"
)

// Claims is the payload stored inside a JWT access token.
type Claims struct {
	UserID   uuid.UUID `json:"uid"`
	Username string    `json:"sub"`
	Role     string    `json:"role"`
	jwt.RegisteredClaims
}

// TokenPair holds a freshly issued access and refresh token.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// IssueTokenPair creates a signed JWT access token and an opaque refresh token.
func IssueTokenPair(cfg config.AuthConfig, userID uuid.UUID, username, role string) (*TokenPair, error) {
	now := time.Now()
	expiresAt := now.Add(cfg.AccessTokenExpiry)

	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Issuer:    "fog-next",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("signing access token: %w", err)
	}

	rt, err := generateToken(32)
	if err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  signed,
		RefreshToken: rt,
		ExpiresAt:    expiresAt,
	}, nil
}

// ParseAccessToken validates and parses a JWT access token string.
func ParseAccessToken(cfg config.AuthConfig, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

// generateToken returns a cryptographically random URL-safe token of byteLen bytes.
func generateToken(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateAPIToken creates a new long-lived API token for scripting use.
func GenerateAPIToken() (string, error) {
	return generateToken(32)
}
