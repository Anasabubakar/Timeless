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

// ApplyToCompany merges externally-sourced field values onto an
// already-loaded Company — the inbound-pull counterpart to
// CompanyToRecord. Only fields present in the map are touched (an absent
// key means "the external mapping doesn't cover this field," not "clear
// it"), and only recognized field names are applied — an unrecognized key
// is ignored rather than erroring, since FromExternal only ever emits
// keys the field mapping was explicitly configured with.
func ApplyToCompany(c *models.Company, fields map[string]interface{}) {
	if v, ok := fields["name"].(string); ok && v != "" {
		c.Name = v
	}
	if v, ok := fields["status"].(string); ok && v != "" {
		c.Status = v
	}
	if v, ok := fields["domain"].(string); ok {
		c.Domain = &v
	}
	if v, ok := fields["website"].(string); ok {
		c.Website = &v
	}
	if v, ok := fields["description"].(string); ok {
		c.Description = &v
	}
	if v, ok := fields["employee_count"].(string); ok {
		c.EmployeeCount = &v
	}
	if v, ok := fields["annual_revenue"].(string); ok {
		c.AnnualRevenue = &v
	}
	if v, ok := fields["headquarters"].(string); ok {
		c.Headquarters = &v
	}
	if v, ok := fields["linkedin_url"].(string); ok {
		c.LinkedinURL = &v
	}
	if v, ok := fields["phone"].(string); ok {
		c.Phone = &v
	}
}

// ApplyToContact merges externally-sourced field values onto an
// already-loaded Contact; see ApplyToCompany.
func ApplyToContact(c *models.Contact, fields map[string]interface{}) {
	if v, ok := fields["first_name"].(string); ok && v != "" {
		c.FirstName = v
	}
	if v, ok := fields["last_name"].(string); ok && v != "" {
		c.LastName = v
	}
	if v, ok := fields["status"].(string); ok && v != "" {
		c.Status = v
	}
	if v, ok := fields["email"].(string); ok {
		c.Email = &v
	}
	if v, ok := fields["phone"].(string); ok {
		c.Phone = &v
	}
	if v, ok := fields["title"].(string); ok {
		c.Title = &v
	}
	if v, ok := fields["department"].(string); ok {
		c.Department = &v
	}
	if v, ok := fields["linkedin_url"].(string); ok {
		c.LinkedinURL = &v
	}
	if v, ok := fields["notes"].(string); ok {
		c.Notes = &v
	}
}

// ApplyToSponsor merges externally-sourced field values onto an
// already-loaded Sponsor; see ApplyToCompany. Stage intentionally goes
// through the same plain-field assignment as everything else here — the
// position/kanban-ordering side effects SponsorService.UpdateStage
// applies are a Timeless-UI concern, not something an external system's
// stage change should trigger.
func ApplyToSponsor(s *models.Sponsor, fields map[string]interface{}) {
	if v, ok := fields["stage"].(string); ok && v != "" {
		s.Stage = v
	}
	if v, ok := fields["tier"].(string); ok {
		s.Tier = &v
	}
	if v, ok := fields["lost_reason"].(string); ok {
		s.LostReason = &v
	}
	if v, ok := fields["notes"].(string); ok {
		s.Notes = &v
	}
	if v, ok := fields["expected_close"].(string); ok {
		s.ExpectedClose = &v
	}
	if v, ok := fields["actual_close"].(string); ok {
		s.ActualClose = &v
	}
	if v, ok := fields["deal_value"].(float64); ok {
		s.DealValue = &v
	}
	if v, ok := fields["probability"].(float64); ok {
		p := int(v)
		s.Probability = &p
	}
}
