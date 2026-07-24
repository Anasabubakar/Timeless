package handler

import (
	"fmt"
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
	store storage.Storage
}

func NewUploadHandler(store storage.Storage) *UploadHandler {
	return &UploadHandler{store: store}
}

func (h *UploadHandler) Upload(c fiber.Ctx) error {
	if h.store == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "file storage not configured")
	}

	orgID := middleware.GetOrgID(c)

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

	sanitized := sanitizeFilename(file.Filename)
	key := storage.GenerateKey(orgID.String(), folder, sanitized)

	result, err := h.store.Upload(c.Context(), key, src, file.Size, contentType)
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
