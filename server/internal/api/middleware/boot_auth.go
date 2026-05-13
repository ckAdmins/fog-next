package middleware

import (
	"context"
	"net/http"

	fogauth "github.com/nemvince/fog-next/internal/auth"
	"github.com/nemvince/fog-next/internal/config"
)

const bootClaimsKey contextKey = "boot_claims"

// AuthenticateBoot validates a boot token (issued at handshake) and injects
// the BootClaims into the request context.  It does NOT accept standard user
// access tokens — use Authenticate for those.
func AuthenticateBoot(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractBearerToken(r)
			if tokenStr == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}
			claims, err := fogauth.ParseBootToken(cfg.Auth, tokenStr)
			if err != nil {
				http.Error(w, `{"error":"invalid or expired boot token"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), bootClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// BootClaimsFrom retrieves the BootClaims from a request context set by
// AuthenticateBoot.  Returns nil if the middleware was not applied.
func BootClaimsFrom(ctx context.Context) *fogauth.BootClaims {
	c, _ := ctx.Value(bootClaimsKey).(*fogauth.BootClaims)
	return c
}

// extractBearerToken is already defined in auth.go (same package).
// The function is shared via the package — no need to redefine it here.
// (See auth.go for the implementation.)
func init() {
	// Compile-time assertion: both keys are distinct.
	_ = contextKey("boot_claims") != contextKey("claims")
}
