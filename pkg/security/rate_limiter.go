package security

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultRateLimit    = 100
	DefaultRateInterval = time.Minute
)

var ErrRateLimitExceeded = errors.New("rate limit exceeded")

type RateLimiterConfig struct {
	Redis              *redis.Client
	Limit              int
	Internal           time.Duration
	SkipSuccessfulAuth bool
}

type RateLimiter struct {
	redis    *redis.Client
	limit    int
	internal time.Duration
	skipAuth bool
}

func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	if config.Redis == nil {
		panic("Redis client is required for rate limiting")
	}

	if config.Limit <= 0 {
		config.Limit = DefaultRateLimit
	}
	if config.Internal <= 0 {
		config.Internal = DefaultRateInterval
	}
	return &RateLimiter{
		redis:    config.Redis,
		limit:    config.Limit,
		internal: config.Internal,
		skipAuth: config.SkipSuccessfulAuth,
	}
}

func getIPAddress(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			clientIP := strings.TrimSpace(ips[0])
			return clientIP
		}
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func (rl *RateLimiter) getLimitKey(r *http.Request) string {
	ip := getIPAddress(r)
	path := r.URL.Path
	return fmt.Sprintf("ratelimit:%s:%s", ip, path)
}

func (rl *RateLimiter) CheckRateLimit(ctx context.Context, r *http.Request) (int, error) {
	key := rl.getLimitKey(r)
	now := time.Now().Unix()
	windowStart := now - int64(rl.internal.Seconds())

	// Remove old entries (outside the current window)
	err := rl.redis.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart, 10)).Err()
	if err != nil {
		log.Error().Err(err).Msg("Failed to remove old rate limit entries")
		// Allow the request to proceed if we can't communicate with Redis
		return 0, nil
	}

	// Count existing requests in the current window
	count, err := rl.redis.ZCard(ctx, key).Result()
	if err != nil {
		log.Error().Err(err).Msg("Failed to count rate limit entries")
		// Allow the request to proceed if we can't communicate with Redis
		return 0, nil
	}

	// Check if the rate limit has been exceeded
	if count >= int64(rl.limit) {
		return int(count), ErrRateLimitExceeded
	}

	// Add the current request to the sorted set with the current timestamp as score
	err = rl.redis.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: now,
	}).Err()
	if err != nil {
		log.Error().Err(err).Msg("Failed to add rate limit entry")
	}

	// Set expiration for the key to the rate limit interval + 1 minute
	err = rl.redis.Expire(ctx, key, rl.internal+time.Minute).Err()
	if err != nil {
		log.Error().Err(err).Msg("Failed to set rate limit key expiration")
	}

	return int(count) + 1, nil
}
