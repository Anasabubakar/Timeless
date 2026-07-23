package handler

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/sponsoros/backend/internal/middleware"
)

type SearchHandler struct {
	db *gorm.DB
}

func NewSearchHandler(db *gorm.DB) *SearchHandler {
	return &SearchHandler{db: db}
}

type SearchResult struct {
	ID         uuid.UUID `json:"id"`
	Type       string    `json:"type"`
	Title      string    `json:"title"`
	Subtitle   string    `json:"subtitle,omitempty"`
	Rank       float64   `json:"rank"`
}

func (h *SearchHandler) Search(c fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	query := c.Query("q")
	if query == "" {
		return fiber.NewError(fiber.StatusBadRequest, "query parameter 'q' is required")
	}

	limit := c.QueryInt("limit", 20)
	if limit > 50 {
		limit = 50
	}

	entityFilter := c.Query("type")

	tsQuery := sanitizeSearchQuery(query)
	var results []SearchResult

	unions := []string{}

	if entityFilter == "" || entityFilter == "sponsor" {
		unions = append(unions, fmt.Sprintf(`
			SELECT s.id, 'sponsor' as type,
				COALESCE(c.name, '') as title,
				COALESCE(s.stage, '') as subtitle,
				ts_rank(to_tsvector('english', COALESCE(c.name, '') || ' ' || COALESCE(c.industry, '') || ' ' || COALESCE(s.notes, '')), plainto_tsquery('english', '%s')) as rank
			FROM sponsors s
			LEFT JOIN companies c ON s.company_id = c.id
			WHERE s.organization_id = '%s' AND s.deleted_at IS NULL
				AND to_tsvector('english', COALESCE(c.name, '') || ' ' || COALESCE(c.industry, '') || ' ' || COALESCE(s.notes, ''))
				@@ plainto_tsquery('english', '%s')
		`, tsQuery, orgID.String(), tsQuery))
	}

	if entityFilter == "" || entityFilter == "company" {
		unions = append(unions, fmt.Sprintf(`
			SELECT id, 'company' as type,
				name as title,
				COALESCE(industry, '') as subtitle,
				ts_rank(to_tsvector('english', name || ' ' || COALESCE(industry, '') || ' ' || COALESCE(description, '')), plainto_tsquery('english', '%s')) as rank
			FROM companies
			WHERE organization_id = '%s' AND deleted_at IS NULL
				AND to_tsvector('english', name || ' ' || COALESCE(industry, '') || ' ' || COALESCE(description, ''))
				@@ plainto_tsquery('english', '%s')
		`, tsQuery, orgID.String(), tsQuery))
	}

	if entityFilter == "" || entityFilter == "contact" {
		unions = append(unions, fmt.Sprintf(`
			SELECT id, 'contact' as type,
				first_name || ' ' || last_name as title,
				COALESCE(email, '') as subtitle,
				ts_rank(to_tsvector('english', first_name || ' ' || last_name || ' ' || COALESCE(email, '') || ' ' || COALESCE(job_title, '')), plainto_tsquery('english', '%s')) as rank
			FROM contacts
			WHERE organization_id = '%s' AND deleted_at IS NULL
				AND to_tsvector('english', first_name || ' ' || last_name || ' ' || COALESCE(email, '') || ' ' || COALESCE(job_title, ''))
				@@ plainto_tsquery('english', '%s')
		`, tsQuery, orgID.String(), tsQuery))
	}

	if entityFilter == "" || entityFilter == "campaign" {
		unions = append(unions, fmt.Sprintf(`
			SELECT id, 'campaign' as type,
				name as title,
				COALESCE(status, '') as subtitle,
				ts_rank(to_tsvector('english', name || ' ' || COALESCE(description, '')), plainto_tsquery('english', '%s')) as rank
			FROM campaigns
			WHERE organization_id = '%s' AND deleted_at IS NULL
				AND to_tsvector('english', name || ' ' || COALESCE(description, ''))
				@@ plainto_tsquery('english', '%s')
		`, tsQuery, orgID.String(), tsQuery))
	}

	if len(unions) == 0 {
		return c.JSON(fiber.Map{"data": []SearchResult{}, "total": 0})
	}

	sql := ""
	for i, u := range unions {
		if i > 0 {
			sql += " UNION ALL "
		}
		sql += u
	}
	sql += fmt.Sprintf(" ORDER BY rank DESC LIMIT %d", limit)

	if err := h.db.Raw(sql).Scan(&results).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "search failed")
	}

	if results == nil {
		results = []SearchResult{}
	}

	return c.JSON(fiber.Map{
		"data":  results,
		"total": len(results),
		"query": query,
	})
}

func sanitizeSearchQuery(q string) string {
	safe := make([]byte, 0, len(q))
	for i := 0; i < len(q); i++ {
		c := q[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == ' ' || c == '-' {
			safe = append(safe, c)
		}
	}
	return string(safe)
}
