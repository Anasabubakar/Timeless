// Package reqbind decodes and validates JSON request bodies in one step.
// Fiber's own c.Bind().JSON() (used throughout the handler package)
// silently accepts unknown fields and never runs the `validate:"..."`
// struct tags already present on several DTOs — they were purely
// decorative. JSON here does both: rejects any field the target struct
// doesn't declare (closing the mass-assignment gap where a client could
// smuggle e.g. "organization_id" or "is_admin" into a body and have it
// silently ignored... or worse, accepted, depending on the struct), and
// enforces validation tags, returning a uniform apierror.APIError either
// way instead of each handler hand-rolling its own checks.
package reqbind

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	"github.com/timeless/backend/internal/pkg/apierror"
)

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	// Report the JSON field name in validation errors (e.g. "email"),
	// not the Go struct field name (e.g. "Email") — the client sent
	// JSON, so errors should speak JSON.
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})
	return v
}

// JSON decodes c's body into dst (which must be a pointer), rejecting
// any field not present in dst's JSON tags, then runs struct-tag
// validation. Returns nil on success; on failure, returns an
// *apierror.APIError ready to write back to the client.
func JSON(c fiber.Ctx, dst any) *apierror.APIError {
	body := c.Body()
	if len(bytes.TrimSpace(body)) == 0 {
		return apierror.BadRequest("request body is required")
	}

	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return apierror.BadRequest("invalid request body: " + decodeErrorMessage(err))
	}

	if err := validate.Struct(dst); err != nil {
		var verrs validator.ValidationErrors
		if errors.As(err, &verrs) {
			return apierror.Validation(fieldErrors(verrs))
		}
		// A non-ValidationErrors error here means the validator itself
		// couldn't run (e.g. dst isn't a struct pointer) — a programming
		// error in the handler, not a client-supplied problem.
		return apierror.Internal("validation could not be performed")
	}

	return nil
}

// fieldErrors turns validator.ValidationErrors into a client-friendly
// field -> message map for apierror.Validation's Details.
func fieldErrors(verrs validator.ValidationErrors) map[string]string {
	details := make(map[string]string, len(verrs))
	for _, fe := range verrs {
		details[fe.Field()] = validationMessage(fe)
	}
	return details
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return fmt.Sprintf("must be at least %s characters/items", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters/items", fe.Param())
	case "url":
		return "must be a valid URL"
	case "oneof":
		return "must be one of: " + fe.Param()
	default:
		return "is invalid (" + fe.Tag() + ")"
	}
}

// decodeErrorMessage keeps json.Decoder's error text (which already
// includes the offending field name for unknown-field/type-mismatch
// errors) without leaking Go type names or stack-shaped detail beyond
// what's useful to a client fixing their request.
func decodeErrorMessage(err error) string {
	var unmarshalErr *json.UnmarshalTypeError
	if errors.As(err, &unmarshalErr) {
		return fmt.Sprintf("field %q must be a %s", unmarshalErr.Field, unmarshalErr.Type.String())
	}
	msg := err.Error()
	if strings.HasPrefix(msg, "json: unknown field ") {
		return "unexpected field " + strings.TrimPrefix(msg, "json: unknown field ")
	}
	return "malformed JSON"
}
