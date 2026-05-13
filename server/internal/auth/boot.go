package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/nemvince/fog-next/internal/config"
)

// BootClaims is the payload stored in a boot token issued at handshake.
// The token scopes the session to a specific task and host; it carries no
// user identity and is never accepted by the standard auth middleware.
type BootClaims struct {
	TaskID uuid.UUID `json:"tid"`
	HostID uuid.UUID `json:"hid"`
	Action string    `json:"act"`
	jwt.RegisteredClaims
}

// IssueBootToken creates a short-lived signed JWT boot token.
// TTL defaults to 2 hours if BootTokenExpiry is zero.
func IssueBootToken(cfg config.AuthConfig, taskID, hostID uuid.UUID, action string) (string, error) {
	ttl := cfg.BootTokenExpiry
	if ttl == 0 {
		ttl = 2 * time.Hour
	}
	now := time.Now()
	claims := BootClaims{
		TaskID: taskID,
		HostID: hostID,
		Action: action,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Issuer:    "fog-next-boot",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("signing boot token: %w", err)
	}
	return signed, nil
}

// ParseBootToken validates and parses a boot JWT string.
func ParseBootToken(cfg config.AuthConfig, tokenStr string) (*BootClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &BootClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing boot token: %w", err)
	}
	claims, ok := token.Claims.(*BootClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid boot token claims")
	}
	if claims.Issuer != "fog-next-boot" {
		return nil, fmt.Errorf("token issuer mismatch")
	}
	return claims, nil
}
