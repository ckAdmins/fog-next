package auth_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nemvince/fog-next/internal/auth"
	"github.com/nemvince/fog-next/internal/config"
)

func testAuthConfig() config.AuthConfig {
	return config.AuthConfig{
		JWTSecret:          "test-secret-that-is-long-enough",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
}

func TestIssueAndParseAccessToken(t *testing.T) {
	cfg := testAuthConfig()
	userID := uuid.New()

	pair, err := auth.IssueTokenPair(cfg, userID, "alice", "admin")
	if err != nil {
		t.Fatalf("IssueTokenPair: %v", err)
	}
	if pair.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if pair.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}

	claims, err := auth.ParseAccessToken(cfg, pair.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("UserID: got %v, want %v", claims.UserID, userID)
	}
	if claims.Username != "alice" {
		t.Errorf("Username: got %q, want %q", claims.Username, "alice")
	}
	if claims.Role != "admin" {
		t.Errorf("Role: got %q, want %q", claims.Role, "admin")
	}
}

func TestParseAccessToken_InvalidSignature(t *testing.T) {
	cfg := testAuthConfig()
	pair, _ := auth.IssueTokenPair(cfg, uuid.New(), "bob", "readonly")

	wrongCfg := testAuthConfig()
	wrongCfg.JWTSecret = "wrong-secret"
	_, err := auth.ParseAccessToken(wrongCfg, pair.AccessToken)
	if err == nil {
		t.Fatal("expected error for wrong secret, got nil")
	}
}

func TestParseAccessToken_Expired(t *testing.T) {
	cfg := testAuthConfig()
	cfg.AccessTokenExpiry = -1 * time.Second // already expired

	pair, err := auth.IssueTokenPair(cfg, uuid.New(), "carol", "readonly")
	if err != nil {
		t.Fatalf("IssueTokenPair: %v", err)
	}

	_, err = auth.ParseAccessToken(cfg, pair.AccessToken)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestParseAccessToken_Malformed(t *testing.T) {
	cfg := testAuthConfig()
	_, err := auth.ParseAccessToken(cfg, "not.a.jwt")
	if err == nil {
		t.Fatal("expected error for malformed token, got nil")
	}
}

func TestIssueTokenPair_UniqueRefreshTokens(t *testing.T) {
	cfg := testAuthConfig()
	seen := make(map[string]bool)
	for range 10 {
		pair, err := auth.IssueTokenPair(cfg, uuid.New(), "user", "readonly")
		if err != nil {
			t.Fatalf("IssueTokenPair: %v", err)
		}
		if seen[pair.RefreshToken] {
			t.Fatal("IssueTokenPair produced a duplicate refresh token")
		}
		seen[pair.RefreshToken] = true
	}
}
