package reqbind

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/timeless/backend/internal/pkg/apierror"
)

type testInput struct {
	Email string `json:"email" validate:"required,email"`
	Name  string `json:"name" validate:"required,min=2"`
}

func bind(t *testing.T, body string) (testInput, *apierror.APIError) {
	t.Helper()
	app := fiber.New()
	var got testInput
	var apiErr *apierror.APIError
	app.Post("/x", func(c fiber.Ctx) error {
		apiErr = JSON(c, &got)
		return c.SendString("ok")
	})
	req := httptest.NewRequest("POST", "/x", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if _, err := app.Test(req); err != nil {
		t.Fatal(err)
	}
	return got, apiErr
}

func TestJSON_ValidBody(t *testing.T) {
	got, apiErr := bind(t, `{"email":"a@b.com","name":"Ada"}`)
	if apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}
	if got.Email != "a@b.com" || got.Name != "Ada" {
		t.Fatalf("got %+v", got)
	}
}

func TestJSON_RejectsUnknownFields(t *testing.T) {
	_, apiErr := bind(t, `{"email":"a@b.com","name":"Ada","is_admin":true}`)
	if apiErr == nil {
		t.Fatal("expected an error for an unknown field, got none")
	}
	if apiErr.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", apiErr.StatusCode, fiber.StatusBadRequest)
	}
}

func TestJSON_RequiresRequiredFields(t *testing.T) {
	_, apiErr := bind(t, `{"email":"a@b.com"}`)
	if apiErr == nil {
		t.Fatal("expected a validation error for missing required field")
	}
}

func TestJSON_ValidatesEmailFormat(t *testing.T) {
	_, apiErr := bind(t, `{"email":"not-an-email","name":"Ada"}`)
	if apiErr == nil {
		t.Fatal("expected a validation error for a malformed email")
	}
}

func TestJSON_RejectsEmptyBody(t *testing.T) {
	_, apiErr := bind(t, ``)
	if apiErr == nil {
		t.Fatal("expected an error for an empty body")
	}
}

func TestJSON_RejectsMalformedJSON(t *testing.T) {
	_, apiErr := bind(t, `{not json`)
	if apiErr == nil {
		t.Fatal("expected an error for malformed JSON")
	}
}
