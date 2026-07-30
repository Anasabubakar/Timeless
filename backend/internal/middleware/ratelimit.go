package middleware

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RateLimitConfig describes one named limiter: a sustained ceiling over
// Window, plus an optional tighter Burst ceiling over BurstWindow so a
// client can't spend its entire sustained budget in the first second.
// Both checks must pass. KeyFunc decides what's being limited together
// (an IP, a user, an org, an attempted email address, ...).
type RateLimitConfig struct {
	Name        string
	Limit       int
	Window      time.Duration
	Burst       int // 0 disables the burst check
	BurstWindow time.Duration
	KeyFunc     func(c fiber.Ctx) string
}

// RedisRateLimiter implements fixed-window counters in Redis (INCR +
// EXPIRE on first hit), so limits are shared across every instance of
// the API instead of each process tracking its own in-memory counters —
// the previous implementation (an in-memory map) gave every horizontally
// scaled instance its own independent budget, which isn't a real limit
// at all once there's more than one instance.
type RedisRateLimiter struct {
	rdb *redis.Client
	db  *gorm.DB // optional, for audit logging violations; nil-safe
}

func NewRedisRateLimiter(rdb *redis.Client, db *gorm.DB) *RedisRateLimiter {
	return &RedisRateLimiter{rdb: rdb, db: db}
}

// Limit builds the fiber.Handler for a given config.
func (rl *RedisRateLimiter) Limit(cfg RateLimitConfig) fiber.Handler {
	return func(c fiber.Ctx) error {
		key := cfg.KeyFunc(c)
		if key == "" {
			return c.Next()
		}

		sustainedCount, sustainedTTL, err := rl.hit(c.Context(), "rl:"+cfg.Name+":s:"+key, cfg.Window)
		if err != nil {
			// Fail open: a Redis outage should degrade abuse protection,
			// not take the whole API down with it. Logged loudly so it's
			// visible in monitoring rather than silently permissive.
			log.Printf("ratelimit: redis error for %s (failing open): %v", cfg.Name, err)
			return c.Next()
		}
		if sustainedCount > int64(cfg.Limit) {
			return rl.deny(c, cfg.Name, key, sustainedTTL)
		}

		if cfg.Burst > 0 {
			burstCount, burstTTL, err := rl.hit(c.Context(), "rl:"+cfg.Name+":b:"+key, cfg.BurstWindow)
			if err != nil {
				log.Printf("ratelimit: redis error for %s burst check (failing open): %v", cfg.Name, err)
				return c.Next()
			}
			if burstCount > int64(cfg.Burst) {
				return rl.deny(c, cfg.Name, key, burstTTL)
			}
		}

		return c.Next()
	}
}

// hit atomically increments the counter at key and, only on the first
// hit in a window, sets its expiry — so the window resets cleanly
// instead of sliding forward on every request.
func (rl *RedisRateLimiter) hit(ctx context.Context, key string, window time.Duration) (count int64, ttl time.Duration, err error) {
	count, err = rl.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, 0, err
	}
	if count == 1 {
		rl.rdb.Expire(ctx, key, window)
		return count, window, nil
	}
	ttl, err = rl.rdb.TTL(ctx, key).Result()
	if err != nil || ttl < 0 {
		ttl = window
	}
	return count, ttl, nil
}

func (rl *RedisRateLimiter) deny(c fiber.Ctx, name, key string, retryAfter time.Duration) error {
	c.Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
	rl.logViolation(c, name, key)
	return fiber.NewError(fiber.StatusTooManyRequests, "rate limit exceeded, try again later")
}

// logViolation records a rate-limit violation as a security event,
// mirroring RBACMiddleware.logDenial — both are "something tried to do
// more than it should have been allowed to" events an admin needs
// visibility into.
func (rl *RedisRateLimiter) logViolation(c fiber.Ctx, limiterName, key string) {
	orgID := GetOrgID(c)
	userID := GetUserID(c)

	var userIDPtr *uuid.UUID
	if userID != uuid.Nil {
		userIDPtr = &userID
	}

	LogSecurityEvent(rl.db, orgID, userIDPtr, "rate_limit", "rate_limit_violation", "rate limited: "+limiterName, c.IP(), map[string]string{
		"limiter": limiterName,
		"key":     key,
		"method":  c.Method(),
		"path":    c.Path(),
	})
}

// Key derivation helpers, composed into RateLimitConfig.KeyFunc.

// ByIP limits per client IP — the right default for unauthenticated
// endpoints where there's no other identity yet.
func ByIP(c fiber.Ctx) string {
	return c.IP()
}

// ByUser limits per authenticated user, falling back to IP for
// unauthenticated requests (a route that mixes public/authenticated
// traffic shouldn't let logged-out callers share one global bucket).
func ByUser(c fiber.Ctx) string {
	if userID, ok := c.Locals("user_id").(string); ok && userID != "" {
		return "user:" + userID
	}
	return "ip:" + c.IP()
}

// ByOrg limits per organization — used for cost-sensitive shared
// resources (AI calls) where the budget should be per-tenant, not
// per-user, so N users in one org can't each get a full user-sized quota.
func ByOrg(c fiber.Ctx) string {
	if orgID, ok := c.Locals("org_id").(string); ok && orgID != "" {
		return "org:" + orgID
	}
	return "ip:" + c.IP()
}

// ByBodyEmail limits per the "email" field in the JSON request body —
// for public, unauthenticated endpoints (login, forgot-password,
// resend-verification) where an IP-only limit can be dodged by
// distributing attempts across many IPs against one target account.
// fasthttp buffers the whole body in memory, so peeking at it here
// doesn't consume it — the handler still reads it normally afterward.
// Falls back to IP if the body doesn't parse or has no email, so a
// malformed request still gets *some* limit applied.
func ByBodyEmail(c fiber.Ctx) string {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(c.Body(), &body); err == nil && body.Email != "" {
		return "email:" + strings.ToLower(strings.TrimSpace(body.Email))
	}
	return "ip:" + c.IP()
}

// ByBodyOrgSlug limits per the "org_slug" field in the JSON request body
// — for POST /auth/join, so an attacker distributing organization-
// password guesses across many IPs still hits a ceiling tied to the one
// organization they're actually targeting, the same reasoning as
// ByBodyEmail for account-targeted attacks.
func ByBodyOrgSlug(c fiber.Ctx) string {
	var body struct {
		OrgSlug string `json:"org_slug"`
	}
	if err := json.Unmarshal(c.Body(), &body); err == nil && body.OrgSlug != "" {
		return "org_slug:" + strings.ToLower(strings.TrimSpace(body.OrgSlug))
	}
	return "ip:" + c.IP()
}
