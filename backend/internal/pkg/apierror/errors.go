package apierror

import "net/http"

type APIError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Details    any    `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	return e.Message
}

func New(statusCode int, code, message string) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
	}
}

func BadRequest(message string) *APIError {
	return New(http.StatusBadRequest, "bad_request", message)
}

func Unauthorized(message string) *APIError {
	return New(http.StatusUnauthorized, "unauthorized", message)
}

func Forbidden(message string) *APIError {
	return New(http.StatusForbidden, "forbidden", message)
}

func NotFound(message string) *APIError {
	return New(http.StatusNotFound, "not_found", message)
}

func Conflict(message string) *APIError {
	return New(http.StatusConflict, "conflict", message)
}

func Internal(message string) *APIError {
	return New(http.StatusInternalServerError, "internal_error", message)
}

func Validation(details any) *APIError {
	return &APIError{
		StatusCode: http.StatusUnprocessableEntity,
		Code:       "validation_error",
		Message:    "Validation failed",
		Details:    details,
	}
}
