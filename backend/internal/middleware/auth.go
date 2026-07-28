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

	return m.authenticate(c, parts[1])
}

// HandleWS authenticates the WebSocket upgrade route. Browsers can't attach
// an Authorization header to a WebSocket handshake, so the frontend sends
// the token as a query param instead (`/ws?token=...`) — this is the WS
// equivalent of Handle, reading from the query string rather than the
// header. Without this, every WebSocket connection was rejected with 401
// immediately after opening, and the client reconnected every 3 seconds
// forever.
func (m *AuthMiddleware) HandleWS(c fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing token query param")
	}
	return m.authenticate(c, token)
}

func (m *AuthMiddleware) authenticate(c fiber.Ctx, tokenString string) error {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
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
