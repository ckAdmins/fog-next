package middleware

import (
	"context"
	"net/http"
	"strings"

	fogauth "github.com/nemvince/fog-next/internal/auth"
	"github.com/nemvince/fog-next/internal/config"
)

type contextKey string

const claimsKey contextKey = "claims"

// Authenticate parses and validates the Bearer JWT on each request.
// Requests without a valid token receive 401 Unauthorized.
func Authenticate(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractBearerToken(r)
			if tokenStr == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			claims, err := fogauth.ParseAccessToken(cfg.Auth, tokenStr)
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFrom retrieves the authenticated JWT claims from a request context.
func ClaimsFrom(ctx context.Context) *fogauth.Claims {
	c, _ := ctx.Value(claimsKey).(*fogauth.Claims)
	return c
}

func extractBearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	// Also support ?token= query param for legacy client compatibility.
	return r.URL.Query().Get("token")
}
