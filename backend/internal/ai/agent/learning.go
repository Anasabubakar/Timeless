package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type OutcomeType string

const (
	OutcomeSuccess  OutcomeType = "success"
	OutcomeFailure  OutcomeType = "failure"
	OutcomeNeutral  OutcomeType = "neutral"
	OutcomePositive OutcomeType = "positive_feedback"
	OutcomeNegative OutcomeType = "negative_feedback"
)

type AgentOutcome struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	AgentType      string         `gorm:"size:50;not null;index" json:"agent_type"`
	ConversationID *uuid.UUID     `gorm:"type:uuid" json:"conversation_id,omitempty"`
	Query          string         `gorm:"type:text;not null" json:"query"`
	Response       string         `gorm:"type:text" json:"response"`
	Outcome        OutcomeType    `gorm:"size:30;not null;index" json:"outcome"`
	Score          float64        `gorm:"default:0" json:"score"`
	Feedback       *string        `gorm:"type:text" json:"feedback,omitempty"`
	Metadata       datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`
	CreatedAt      time.Time      `json:"created_at"`
}

func (AgentOutcome) TableName() string {
	return "agent_outcomes"
}

type LearnedPreference struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrganizationID uuid.UUID `gorm:"type:uuid;not null;index" json:"organization_id"`
	AgentType      string    `gorm:"size:50;not null;index" json:"agent_type"`
	Category       string    `gorm:"size:100;not null" json:"category"`
	Preference     string    `gorm:"type:text;not null" json:"preference"`
	Confidence     float64   `gorm:"default:0.5" json:"confidence"`
	LearnedFrom    int       `gorm:"default:1" json:"learned_from"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (LearnedPreference) TableName() string {
	return "agent_learned_preferences"
}

type LearningService struct {
	db *gorm.DB
}

func NewLearningService(db *gorm.DB) *LearningService {
	return &LearningService{db: db}
}

func (s *LearningService) RecordOutcome(ctx context.Context, outcome *AgentOutcome) error {
	if outcome.ID == uuid.Nil {
		outcome.ID = uuid.New()
	}
	return s.db.WithContext(ctx).Create(outcome).Error
}

func (s *LearningService) GetRecentOutcomes(ctx context.Context, orgID uuid.UUID, agentType AgentType, limit int) ([]AgentOutcome, error) {
	var outcomes []AgentOutcome
	err := s.db.WithContext(ctx).
		Where("organization_id = ? AND agent_type = ?", orgID, string(agentType)).
		Order("created_at DESC").
		Limit(limit).
		Find(&outcomes).Error
	return outcomes, err
}

func (s *LearningService) GetSuccessPatterns(ctx context.Context, orgID uuid.UUID, agentType AgentType) ([]AgentOutcome, error) {
	var outcomes []AgentOutcome
	err := s.db.WithContext(ctx).
		Where("organization_id = ? AND agent_type = ? AND outcome IN ?",
			orgID, string(agentType), []OutcomeType{OutcomeSuccess, OutcomePositive}).
		Order("score DESC").
		Limit(5).
		Find(&outcomes).Error
	return outcomes, err
}

func (s *LearningService) StorePreference(ctx context.Context, pref *LearnedPreference) error {
	var existing LearnedPreference
	result := s.db.WithContext(ctx).Where(
		"organization_id = ? AND agent_type = ? AND category = ?",
		pref.OrganizationID, pref.AgentType, pref.Category,
	).First(&existing)

	if result.Error == nil {
		existing.Preference = pref.Preference
		existing.Confidence = min64(existing.Confidence+0.1, 1.0)
		existing.LearnedFrom++
		return s.db.WithContext(ctx).Save(&existing).Error
	}

	if pref.ID == uuid.Nil {
		pref.ID = uuid.New()
	}
	return s.db.WithContext(ctx).Create(pref).Error
}

func (s *LearningService) GetPreferences(ctx context.Context, orgID uuid.UUID, agentType AgentType) ([]LearnedPreference, error) {
	var prefs []LearnedPreference
	err := s.db.WithContext(ctx).
		Where("organization_id = ? AND agent_type = ?", orgID, string(agentType)).
		Where("confidence >= ?", 0.3).
		Order("confidence DESC").
		Limit(10).
		Find(&prefs).Error
	return prefs, err
}

func (s *LearningService) BuildContextInjection(ctx context.Context, orgID uuid.UUID, agentType AgentType) string {
	var parts []string

	prefs, err := s.GetPreferences(ctx, orgID, agentType)
	if err == nil && len(prefs) > 0 {
		parts = append(parts, "## Learned Preferences for this Organization")
		for _, p := range prefs {
			parts = append(parts, fmt.Sprintf("- [%s] %s (confidence: %.0f%%)", p.Category, p.Preference, p.Confidence*100))
		}
	}

	patterns, err := s.GetSuccessPatterns(ctx, orgID, agentType)
	if err == nil && len(patterns) > 0 {
		parts = append(parts, "\n## Past Successful Approaches")
		for _, p := range patterns {
			summary := p.Query
			if len(summary) > 100 {
				summary = summary[:100] + "..."
			}
			parts = append(parts, fmt.Sprintf("- Query: %s → Outcome: %s", summary, p.Outcome))
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return "\n\n---\n" + strings.Join(parts, "\n") + "\n---\n"
}

func (s *LearningService) LearnFromFeedback(ctx context.Context, orgID uuid.UUID, agentType AgentType, feedback string, isPositive bool) error {
	outcome := OutcomeNegative
	if isPositive {
		outcome = OutcomePositive
	}

	record := &AgentOutcome{
		OrganizationID: orgID,
		AgentType:      string(agentType),
		Outcome:        outcome,
		Feedback:       &feedback,
		Score:          0.5,
	}
	if isPositive {
		record.Score = 1.0
	}

	return s.RecordOutcome(ctx, record)
}

func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
