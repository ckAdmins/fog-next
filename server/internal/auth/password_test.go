package auth_test

import (
	"strings"
	"testing"

	"github.com/nemvince/fog-next/internal/auth"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := auth.HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	// Hash must look like a bcrypt hash
	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
		t.Errorf("unexpected hash prefix: %q", hash[:6])
	}

	if err := auth.CheckPassword(hash, "correct-horse-battery-staple"); err != nil {
		t.Errorf("CheckPassword should succeed: %v", err)
	}
}

func TestCheckPassword_WrongPassword(t *testing.T) {
	hash, err := auth.HashPassword("secret")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if err := auth.CheckPassword(hash, "wrong"); err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}

func TestHashPassword_Empty(t *testing.T) {
	hash, err := auth.HashPassword("")
	if err != nil {
		t.Fatalf("HashPassword empty string should not fail: %v", err)
	}
	if err := auth.CheckPassword(hash, ""); err != nil {
		t.Errorf("CheckPassword empty string should succeed: %v", err)
	}
	if err := auth.CheckPassword(hash, "notempty"); err == nil {
		t.Error("CheckPassword should fail when plain doesn't match empty original")
	}
}

func TestHashPassword_IsDifferentEachTime(t *testing.T) {
	h1, _ := auth.HashPassword("same")
	h2, _ := auth.HashPassword("same")
	if h1 == h2 {
		t.Error("bcrypt hashes of the same password should differ due to random salt")
	}
}
