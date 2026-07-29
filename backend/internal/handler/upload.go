package handler

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/storage"
)

var allowedMimeTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"image/webp":      true,
	"image/svg+xml":   true,
	"application/pdf": true,
	"text/csv":        true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":       true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
}

const maxFileSize = 10 << 20 // 10MB

type UploadHandler struct {
	store   storage.Storage
	scanner storage.Scanner
}

// NewUploadHandler defaults to storage.NoopScanner{} when scanner is
// nil, so existing call sites (and tests) that don't care about
// malware scanning don't need to change.
func NewUploadHandler(store storage.Storage, scanner storage.Scanner) *UploadHandler {
	if scanner == nil {
		scanner = storage.NoopScanner{}
	}
	return &UploadHandler{store: store, scanner: scanner}
}

func (h *UploadHandler) Upload(c fiber.Ctx) error {
	if h.store == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "file storage not configured")
	}

	orgID := middleware.GetOrgID(c)

	// Reject on the declared Content-Length before fasthttp buffers the
	// whole multipart body into memory/temp files parsing it — a request
	// that's obviously oversized shouldn't cost a full parse to reject.
	// (fiber.Config.BodyLimit already caps this at the connection level;
	// this just fails faster and with a clearer message.)
	if n := c.Request().Header.ContentLength(); n > maxFileSize {
		return fiber.NewError(fiber.StatusRequestEntityTooLarge, "file too large (max 10MB)")
	}

	file, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "file is required")
	}

	if file.Size > maxFileSize {
		return fiber.NewError(fiber.StatusBadRequest, "file too large (max 10MB)")
	}

	contentType := file.Header.Get("Content-Type")
	if !allowedMimeTypes[contentType] {
		return fiber.NewError(fiber.StatusBadRequest, "unsupported file type: "+contentType)
	}

	folder := c.FormValue("folder", "uploads")
	allowedFolders := map[string]bool{
		"uploads": true, "avatars": true, "logos": true,
		"proposals": true, "documents": true, "imports": true,
	}
	if !allowedFolders[folder] {
		return fiber.NewError(fiber.StatusBadRequest, "invalid folder")
	}

	src, err := file.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to read file")
	}
	defer src.Close()

	content, err := io.ReadAll(src)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to read file")
	}

	clean, reason, err := h.scanner.Scan(c.Context(), file.Filename, content)
	if err != nil {
		// Scan failure (scanner unreachable/timed out), not a positive
		// detection: treat as untrusted rather than letting the file
		// through on a technicality.
		return fiber.NewError(fiber.StatusServiceUnavailable, "file could not be scanned, try again")
	}
	if !clean {
		return fiber.NewError(fiber.StatusBadRequest, "file rejected: "+reason)
	}

	sanitized := sanitizeFilename(file.Filename)
	key := storage.GenerateKey(orgID.String(), folder, sanitized)

	result, err := h.store.Upload(c.Context(), key, bytes.NewReader(content), file.Size, contentType)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "upload failed")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": result,
	})
}

func (h *UploadHandler) Delete(c fiber.Ctx) error {
	if h.store == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "file storage not configured")
	}

	key := c.Query("key")
	if key == "" {
		return fiber.NewError(fiber.StatusBadRequest, "key parameter is required")
	}

	orgID := middleware.GetOrgID(c)
	if !strings.HasPrefix(key, fmt.Sprintf("orgs/%s/", orgID.String())) {
		return fiber.NewError(fiber.StatusForbidden, "access denied to this file")
	}

	if err := h.store.Delete(c.Context(), key); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "delete failed")
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

func sanitizeFilename(name string) string {
	ext := path.Ext(name)
	base := strings.TrimSuffix(name, ext)
	base = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, base)
	if len(base) > 100 {
		base = base[:100]
	}
	return base + ext
}
