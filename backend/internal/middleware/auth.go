package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"

	"github.com/timeless/backend/internal/config"
)

type AuthMiddleware struct {
	cfg *config.Config
}

func NewAuth(cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{cfg: cfg}
}

func (m *AuthMiddleware) Handle(c fiber.Ctx) error {
	auth := c.Get("Authorization")
	if auth == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing authorization header")
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid authorization format")
	}

	token, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid signing method")
		}
		return []byte(m.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token claims")
	}

	c.Locals("user_id", claims["sub"])
	c.Locals("org_id", claims["org_id"])
	c.Locals("email", claims["email"])

	return c.Next()
}
