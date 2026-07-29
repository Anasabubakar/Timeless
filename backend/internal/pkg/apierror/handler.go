package apierror

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v3"
)

// WriteError is the single place that turns whatever error a handler
// returned into an HTTP response, so every route — regardless of
// whether it returns an *APIError, a *fiber.Error, or a bare Go error —
// produces the same {code, message, details} shape. A bare error (the
// case that previously fell through to a generic 500 with no message,
// but could just as easily have been "return err" with a raw DB/SQL
// error string if a handler wasn't careful) is logged server-side in
// full and never has its message forwarded to the client.
func WriteError(c fiber.Ctx, err error) error {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return c.Status(apiErr.StatusCode).JSON(fiber.Map{
			"error":   true,
			"code":    apiErr.Code,
			"message": apiErr.Message,
			"details": apiErr.Details,
		})
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return c.Status(fiberErr.Code).JSON(fiber.Map{
			"error":   true,
			"code":    codeForStatus(fiberErr.Code),
			"message": fiberErr.Message,
		})
	}

	// Anything else is unexpected — log it with full detail server-side,
	// but the client only ever sees a generic message. This is the fix
	// for handlers that used to do `fiber.NewError(500, err.Error())`
	// and forward raw DB/internal error text straight to the response.
	log.Printf("apierror: unhandled error on %s %s: %v", c.Method(), c.Path(), err)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error":   true,
		"code":    "internal_error",
		"message": "Internal Server Error",
	})
}

func codeForStatus(status int) string {
	switch status {
	case fiber.StatusBadRequest:
		return "bad_request"
	case fiber.StatusUnauthorized:
		return "unauthorized"
	case fiber.StatusForbidden:
		return "forbidden"
	case fiber.StatusNotFound:
		return "not_found"
	case fiber.StatusConflict:
		return "conflict"
	case fiber.StatusTooManyRequests:
		return "rate_limited"
	case fiber.StatusUnprocessableEntity:
		return "validation_error"
	default:
		return "error"
	}
}
