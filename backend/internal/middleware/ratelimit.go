package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

type RateLimiter struct {
	requests map[string]*bucket
	mu       sync.RWMutex
	rate     int
	window   time.Duration
}

type bucket struct {
	count    int
	resetAt  time.Time
}

func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*bucket),
		rate:     rate,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) Handle(c fiber.Ctx) error {
	key := c.IP()
	if userID, ok := c.Locals("user_id").(string); ok && userID != "" {
		key = userID
	}

	rl.mu.Lock()
	b, exists := rl.requests[key]
	now := time.Now()

	if !exists || now.After(b.resetAt) {
		rl.requests[key] = &bucket{count: 1, resetAt: now.Add(rl.window)}
		rl.mu.Unlock()
		return c.Next()
	}

	if b.count >= rl.rate {
		rl.mu.Unlock()
		c.Set("Retry-After", b.resetAt.Format(time.RFC1123))
		return fiber.NewError(fiber.StatusTooManyRequests, "rate limit exceeded")
	}

	b.count++
	rl.mu.Unlock()
	return c.Next()
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for k, b := range rl.requests {
			if now.After(b.resetAt) {
				delete(rl.requests, k)
			}
		}
		rl.mu.Unlock()
	}
}
