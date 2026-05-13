package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// perIPLimiter holds the rate.Limiter for a single IP address and the time it
// was last seen (used to prune the map and prevent unbounded memory growth).
type perIPLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter is a per-IP rate limiter middleware backed by a token-bucket.
// It returns 429 Too Many Requests when the bucket for an IP is exhausted.
type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*perIPLimiter
	r       rate.Limit // tokens per second
	b       int        // bucket size (burst)
}

// NewRateLimiter creates a RateLimiter that allows r requests per second per
// IP with a burst of b. A background goroutine prunes stale entries every
// 5 minutes.
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*perIPLimiter),
		r:       r,
		b:       b,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if e, ok := rl.clients[ip]; ok {
		e.lastSeen = time.Now()
		return e.limiter
	}
	l := rate.NewLimiter(rl.r, rl.b)
	rl.clients[ip] = &perIPLimiter{limiter: l, lastSeen: time.Now()}
	return l
}

func (rl *RateLimiter) cleanup() {
	for range time.Tick(5 * time.Minute) {
		rl.mu.Lock()
		for ip, e := range rl.clients {
			if time.Since(e.lastSeen) > 10*time.Minute {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Handler wraps next and enforces the per-IP rate limit.
func (rl *RateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		if !rl.getLimiter(ip).Allow() {
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
