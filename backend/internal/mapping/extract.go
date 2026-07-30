package mapping

import (
	"github.com/timeless/backend/internal/models"
)

// CompanyToRecord builds a SyncableRecord from a Company. Field names are
// this package's own vocabulary for FieldMapping.Fields[].InternalField —
// stable regardless of the Go struct's own field names/json tags, so a
// mapping saved by a user keeps working even if the model is refactored.
func CompanyToRecord(c *models.Company) SyncableRecord {
	fields := map[string]interface{}{
		"name":   c.Name,
		"status": c.Status,
		"tags":   []string(c.Tags),
	}
	if c.Domain != nil {
		fields["domain"] = *c.Domain
	}
	if c.Website != nil {
		fields["website"] = *c.Website
	}
	if c.Description != nil {
		fields["description"] = *c.Description
	}
	if c.EmployeeCount != nil {
		fields["employee_count"] = *c.EmployeeCount
	}
	if c.AnnualRevenue != nil {
		fields["annual_revenue"] = *c.AnnualRevenue
	}
	if c.Headquarters != nil {
		fields["headquarters"] = *c.Headquarters
	}
	if c.LinkedinURL != nil {
		fields["linkedin_url"] = *c.LinkedinURL
	}
	if c.Phone != nil {
		fields["phone"] = *c.Phone
	}
	if c.Score != nil {
		fields["score"] = float64(*c.Score)
	}
	return SyncableRecord{EntityType: "company", EntityID: c.ID, Fields: fields}
}

// ContactToRecord builds a SyncableRecord from a Contact.
func ContactToRecord(c *models.Contact) SyncableRecord {
	fields := map[string]interface{}{
		"first_name": c.FirstName,
		"last_name":  c.LastName,
		"status":     c.Status,
		"tags":       []string(c.Tags),
	}
	if c.Email != nil {
		fields["email"] = *c.Email
	}
	if c.Phone != nil {
		fields["phone"] = *c.Phone
	}
	if c.Title != nil {
		fields["title"] = *c.Title
	}
	if c.Department != nil {
		fields["department"] = *c.Department
	}
	if c.LinkedinURL != nil {
		fields["linkedin_url"] = *c.LinkedinURL
	}
	if c.Notes != nil {
		fields["notes"] = *c.Notes
	}
	if c.LastContactedAt != nil {
		fields["last_contacted_at"] = *c.LastContactedAt
	}
	return SyncableRecord{EntityType: "contact", EntityID: c.ID, Fields: fields}
}

// SponsorToRecord builds a SyncableRecord from a Sponsor (the sponsorship
// pipeline/deal entity — this is what most orgs actually want to see as
// rows in a Notion CRM database).
func SponsorToRecord(s *models.Sponsor) SyncableRecord {
	fields := map[string]interface{}{
		"stage":            s.Stage,
		"stage_entered_at": s.StageEnteredAt,
	}
	if s.DealValue != nil {
		fields["deal_value"] = *s.DealValue
	}
	if s.Probability != nil {
		fields["probability"] = float64(*s.Probability)
	}
	if s.Tier != nil {
		fields["tier"] = *s.Tier
	}
	if s.ExpectedClose != nil {
		fields["expected_close"] = *s.ExpectedClose
	}
	if s.ActualClose != nil {
		fields["actual_close"] = *s.ActualClose
	}
	if s.LostReason != nil {
		fields["lost_reason"] = *s.LostReason
	}
	if s.Notes != nil {
		fields["notes"] = *s.Notes
	}
	if s.Company.Name != "" {
		fields["company_name"] = s.Company.Name
	}
	return SyncableRecord{EntityType: "sponsor", EntityID: s.ID, Fields: fields}
}
