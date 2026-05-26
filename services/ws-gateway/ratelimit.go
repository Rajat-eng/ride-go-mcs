package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"ride-sharing/shared/env"

	"github.com/redis/go-redis/v9"
)

var luaRateLimit = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return current
`)

type RateLimiter struct {
	rdb *redis.Client
}

func NewRateLimiter(rdb *redis.Client) *RateLimiter {
	return &RateLimiter{rdb: rdb}
}

func (rl *RateLimiter) Limit(limit int, windowSecs int, keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}
			count, err := luaRateLimit.Run(r.Context(), rl.rdb, []string{key}, windowSecs).Int()
			if err != nil {
				log.Printf("rate limiter redis error: %v", err)
				next.ServeHTTP(w, r)
				return
			}
			if count > limit {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// WsConnectionGate enforces a per-user limit on simultaneous WebSocket connections.
// It returns (allowed bool, release func). The caller must call release() when the
// connection closes to decrement the counter.
func (rl *RateLimiter) WsConnectionGate(ctx context.Context, userID string, max int) (bool, func()) {
	key := fmt.Sprintf("ws:conns:%s", userID)
	count, err := luaRateLimit.Run(ctx, rl.rdb, []string{key}, 86400).Int()
	if err != nil {
		log.Printf("ws connection gate redis error: %v", err)
		return true, func() {} // fail open
	}
	if count > max {
		return false, func() {}
	}
	release := func() {
		rl.rdb.Decr(ctx, key)
	}
	return true, release
}

func userKey(prefix string) func(*http.Request) string {
	return func(r *http.Request) string {
		userID, _ := r.Context().Value(ctxKeyUserID).(string)
		if userID == "" {
			return ""
		}
		return fmt.Sprintf("rl:%s:%s", prefix, userID)
	}
}

func ipKey(prefix string) func(*http.Request) string {
	return func(r *http.Request) string {
		return fmt.Sprintf("rl:%s:%s", prefix, realIP(r))
	}
}

func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

var _ = env.GetString // suppress unused import if env is not referenced elsewhere
