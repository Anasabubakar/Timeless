package middleware

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

// idempotencyRecord is what gets cached in Redis for a completed request.
type idempotencyRecord struct {
	Status int    `json:"status"`
	Body   []byte `json:"body"`
}

// Idempotency lets a client attach an "Idempotency-Key" header to a
// mutating, expensive, or side-effecting request (an AI call, a
// proposal generation, an integration sync) so that a network retry —
// or a client's own accidental double-submit — replays the original
// response instead of doing the work twice. Without a key, requests
// pass through unaffected; this is opt-in per-request, not per-route.
func Idempotency(rdb *redis.Client, ttl time.Duration) fiber.Handler {
	return func(c fiber.Ctx) error {
		key := c.Get("Idempotency-Key")
		if key == "" {
			return c.Next()
		}

		userID := GetUserID(c)
		redisKey := "idem:" + userID.String() + ":" + c.Method() + ":" + c.Route().Path + ":" + key
		lockKey := redisKey + ":lock"

		if cached, err := rdb.Get(c.Context(), redisKey).Result(); err == nil {
			var rec idempotencyRecord
			if json.Unmarshal([]byte(cached), &rec) == nil {
				c.Set("Idempotency-Replayed", "true")
				return c.Status(rec.Status).Send(rec.Body)
			}
		}

		// SetNX-style lock: a second request with the same key arriving
		// while the first is still in flight is rejected rather than
		// allowed to duplicate the side effect — the cached-response
		// path above only helps once the first request has finished.
		acquired, err := rdb.SetNX(c.Context(), lockKey, "1", 30*time.Second).Result()
		if err == nil && !acquired {
			return fiber.NewError(fiber.StatusConflict, "a request with this idempotency key is already in progress")
		}
		defer rdb.Del(c.Context(), lockKey)

		nextErr := c.Next()

		status := c.Response().StatusCode()
		if status < fiber.StatusInternalServerError {
			// Server errors aren't cached — the point of idempotency is
			// to let a client safely retry, and caching a 500 would
			// permanently replay a failure that a retry might succeed at.
			body := append([]byte(nil), c.Response().Body()...)
			rec := idempotencyRecord{Status: status, Body: body}
			if data, err := json.Marshal(rec); err == nil {
				rdb.Set(c.Context(), redisKey, data, ttl)
			}
		}

		return nextErr
	}
}
