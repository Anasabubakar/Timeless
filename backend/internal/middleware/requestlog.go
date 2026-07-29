package middleware

import (
	"github.com/gofiber/fiber/v3/middleware/logger"
)

// RequestLogFormat deliberately omits the query string (there is no
// ${url}/${queries} tag here, only ${path}) — several routes take a
// bearer token as a query parameter (OAuth start, the WebSocket
// upgrade), since browsers can't attach an Authorization header to a
// top-level navigation or upgrade request. Logging the full URL would
// put those tokens in every request log line. Pulled out as its own
// constant (rather than left inline where it was previously
// duplicated-by-omission — only one of the two entrypoints had request
// logging at all) so this is one thing to get right, not one per
// entrypoint, and so it's something a test can pin down.
const RequestLogFormat = "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path}\n"

const RequestLogTimeFormat = "2006-01-02 15:04:05"

// RequestLogger builds the standard access-log middleware with the
// format above.
func RequestLogger() logger.Config {
	return logger.Config{
		Format:     RequestLogFormat,
		TimeFormat: RequestLogTimeFormat,
	}
}
