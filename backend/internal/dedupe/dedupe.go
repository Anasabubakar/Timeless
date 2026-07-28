// Package dedupe merges companies that different integrations (or the same
// one, re-run with slightly different casing/URL formatting) created as
// separate rows for what's really one organization. It's a standalone
// package (rather than living in `service`, which imports `worker`) so both
// the HTTP-triggered maintenance endpoint and the background sync worker
// can call it without an import cycle.
//
// Ingestion already prevents new duplicates going forward (see normalize +
// findOrCreateCompany in the worker package) — this handles cleanup of
// whatever already exists.
package dedupe

import (
	"context"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/models"
	"github.com/timeless/backend/internal/normalize"
)

type MergeSummary struct {
	GroupsFound     int `json:"groups_found"`
	CompaniesMerged int `json:"companies_merged"`
}

// MergeDuplicateCompanies groups the org's companies by normalized domain
// (falling back to normalized name), keeps the most complete record in
// each group as primary, reassigns every related row (contacts, decision
// makers, sponsors, pain points) to it, unions tags, and soft-deletes the
// rest — no data is dropped, just consolidated.
func MergeDuplicateCompanies(ctx context.Context, db *gorm.DB, orgID uuid.UUID) (*MergeSummary, error) {
	var companies []models.Company
	if err := db.WithContext(ctx).Where("organization_id = ?", orgID).Order("created_at ASC").Find(&companies).Error; err != nil {
		return nil, err
	}

	groups := make(map[string][]models.Company)
	for _, c := range companies {
		key := dedupeKey(c)
		groups[key] = append(groups[key], c)
	}

	summary := &MergeSummary{}
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		summary.GroupsFound++

		primary := pickPrimaryCompany(group)
		for _, dup := range group {
			if dup.ID == primary.ID {
				continue
			}
			if err := mergeCompanyInto(ctx, db, &primary, &dup); err != nil {
				return summary, err
			}
			summary.CompaniesMerged++
		}
	}
	return summary, nil
}

func dedupeKey(c models.Company) string {
	if c.Domain != nil && *c.Domain != "" {
		if d := normalize.Domain(*c.Domain); d != "" {
			return "domain:" + d
		}
	}
	return "name:" + normalize.CompanyName(c.Name)
}

// pickPrimaryCompany keeps the record with the most filled-in fields, so
// merging doesn't discard richer data just because it happened to be the
// second row created.
func pickPrimaryCompany(group []models.Company) models.Company {
	best := group[0]
	bestScore := completenessScore(best)
	for _, c := range group[1:] {
		if score := completenessScore(c); score > bestScore {
			best, bestScore = c, score
		}
	}
	return best
}

func completenessScore(c models.Company) int {
	score := 0
	if c.Domain != nil && *c.Domain != "" {
		score++
	}
	if c.Website != nil && *c.Website != "" {
		score++
	}
	if c.Description != nil && *c.Description != "" {
		score++
	}
	if c.EmployeeCount != nil && *c.EmployeeCount != "" {
		score++
	}
	if c.LinkedinURL != nil && *c.LinkedinURL != "" {
		score++
	}
	if len(c.EnrichmentData) > len("{}") {
		score++
	}
	return score
}

func mergeCompanyInto(ctx context.Context, db *gorm.DB, primary, dup *models.Company) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, model := range []interface{}{&models.Contact{}, &models.DecisionMaker{}, &models.Sponsor{}, &models.PainPoint{}} {
			if err := tx.Model(model).Where("company_id = ?", dup.ID).Update("company_id", primary.ID).Error; err != nil {
				return err
			}
		}

		updates := map[string]interface{}{"tags": pq.StringArray(mergeStringSlices(primary.Tags, dup.Tags))}
		if (primary.Domain == nil || *primary.Domain == "") && dup.Domain != nil && *dup.Domain != "" {
			updates["domain"] = *dup.Domain
		}
		if err := tx.Model(&models.Company{}).Where("id = ?", primary.ID).Updates(updates).Error; err != nil {
			return err
		}

		return tx.Where("id = ?", dup.ID).Delete(&models.Company{}).Error
	})
}

func mergeStringSlices(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, s := range append(append([]string{}, a...), b...) {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
