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

// luaRateLimit atomically increments a fixed-window counter and sets its TTL
// on the first increment. Returns the current request count in this window.
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

// Limit returns a middleware that allows at most `limit` requests per
// `windowSecs` second window. keyFn derives the bucket key from the request.
// Fails open on Redis errors so a cache outage never blocks traffic.
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
				next.ServeHTTP(w, r) // fail open
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

// userKey returns a key function that buckets by authenticated user ID.
func userKey(prefix string) func(*http.Request) string {
	return func(r *http.Request) string {
		userID, _ := r.Context().Value(ctxKeyUserID).(string)
		if userID == "" {
			return ""
		}
		return fmt.Sprintf("rl:%s:%s", prefix, userID)
	}
}

// ipKey returns a key function that buckets by real client IP.
func ipKey(prefix string) func(*http.Request) string {
	return func(r *http.Request) string {
		return fmt.Sprintf("rl:%s:%s", prefix, realIP(r))
	}
}

// realIP extracts the original client IP, honouring X-Forwarded-For.
func realIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.SplitN(fwd, ",", 2)[0])
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

// WsConnectionGate checks whether a user may open another WS connection.
// Returns (allowed, releaseFunc). Call releaseFunc in the WS handler's defer.
func (rl *RateLimiter) WsConnectionGate(ctx context.Context, userID string, limit int) (bool, func()) {
	key := fmt.Sprintf("ws:conn:%s", userID)

	count, err := rl.rdb.Incr(ctx, key).Result()
	if err != nil {
		log.Printf("ws connection gate redis error: %v", err)
		return true, func() {} // fail open
	}

	if int(count) > limit {
		rl.rdb.Decr(ctx, key)
		return false, func() {}
	}

	return true, func() {
		rl.rdb.Decr(ctx, key)
	}
}

// stripeAllowedNets holds Stripe's published webhook sender IP ranges.
// Override with a comma-separated CIDR list in STRIPE_WEBHOOK_IPS.
var stripeAllowedNets = func() []*net.IPNet {
	raw := env.GetString("STRIPE_WEBHOOK_IPS",
		// Stripe's documented webhook IPs — update when Stripe publishes changes.
		// https://stripe.com/files/ips/ips_webhooks.txt
		"3.18.12.63/32,3.130.192.231/32,13.235.14.237/32,13.235.122.149/32,"+
			"18.211.135.69/32,35.154.171.200/32,52.15.183.38/32,54.88.130.119/32,"+
			"54.88.130.237/32,54.187.174.169/32,54.187.205.235/32,54.187.216.72/32",
	)

	var nets []*net.IPNet
	for _, cidr := range strings.Split(raw, ",") {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			log.Printf("invalid STRIPE_WEBHOOK_IPS entry %q: %v", cidr, err)
			continue
		}
		nets = append(nets, ipnet)
	}
	return nets
}()

// StripeIPWhitelist rejects webhook requests that do not originate from a
// known Stripe IP range. Bypass with STRIPE_WEBHOOK_IPS="" to disable.
func StripeIPWhitelist(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If no CIDRs are configured, allow all (useful in local dev).
		if len(stripeAllowedNets) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		ip := net.ParseIP(realIP(r))
		if ip == nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		for _, cidr := range stripeAllowedNets {
			if cidr.Contains(ip) {
				next.ServeHTTP(w, r)
				return
			}
		}

		log.Printf("stripe webhook blocked: IP %s not in whitelist", ip)
		http.Error(w, "forbidden", http.StatusForbidden)
	})
}
