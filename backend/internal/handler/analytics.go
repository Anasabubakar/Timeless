package handler

import (
	"encoding/csv"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/sponsoros/backend/internal/models"
)

type AnalyticsHandler struct {
	db *gorm.DB
}

func NewAnalyticsHandler(db *gorm.DB) *AnalyticsHandler {
	return &AnalyticsHandler{db: db}
}

// Dashboard returns top-level KPI stats for the org.
func (h *AnalyticsHandler) Dashboard(c fiber.Ctx) error {
	orgID := uuid.MustParse(c.Locals("org_id").(string))

	var totalSponsors int64
	h.db.Model(&models.Sponsor{}).Where("organization_id = ?", orgID).Count(&totalSponsors)

	var activeCampaigns int64
	h.db.Model(&models.Campaign{}).Where("organization_id = ? AND status = ?", orgID, "active").Count(&activeCampaigns)

	var pipelineValue float64
	h.db.Model(&models.Sponsor{}).
		Where("organization_id = ? AND stage NOT IN ?", orgID, []string{"closed_won", "closed_lost"}).
		Select("COALESCE(SUM(deal_value), 0)").Scan(&pipelineValue)

	var closedWon int64
	h.db.Model(&models.Sponsor{}).Where("organization_id = ? AND stage = ?", orgID, "closed_won").Count(&closedWon)

	var closedLost int64
	h.db.Model(&models.Sponsor{}).Where("organization_id = ? AND stage = ?", orgID, "closed_lost").Count(&closedLost)

	var conversionRate float64
	if totalSponsors > 0 {
		conversionRate = float64(closedWon) / float64(totalSponsors) * 100
	}

	var totalRevenue float64
	h.db.Model(&models.Sponsor{}).
		Where("organization_id = ? AND stage = ?", orgID, "closed_won").
		Select("COALESCE(SUM(deal_value), 0)").Scan(&totalRevenue)

	var totalContacts int64
	h.db.Model(&models.Contact{}).Where("organization_id = ?", orgID).Count(&totalContacts)

	var totalCompanies int64
	h.db.Model(&models.Company{}).Where("organization_id = ?", orgID).Count(&totalCompanies)

	var avgDealSize float64
	if closedWon > 0 {
		avgDealSize = totalRevenue / float64(closedWon)
	}

	// Average days from creation to close for won deals
	var avgDealVelocity float64
	h.db.Model(&models.Sponsor{}).
		Where("organization_id = ? AND stage = ?", orgID, "closed_won").
		Select("COALESCE(AVG(EXTRACT(EPOCH FROM (updated_at - created_at)) / 86400), 0)").
		Scan(&avgDealVelocity)

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"total_sponsors":    totalSponsors,
			"active_campaigns":  activeCampaigns,
			"pipeline_value":    pipelineValue,
			"conversion_rate":   roundTo1(conversionRate),
			"total_revenue":     totalRevenue,
			"total_contacts":    totalContacts,
			"total_companies":   totalCompanies,
			"closed_won":        closedWon,
			"closed_lost":       closedLost,
			"avg_deal_size":     math.Round(avgDealSize*100) / 100,
			"avg_deal_velocity": roundTo1(avgDealVelocity),
			"win_rate":          calcWinRate(closedWon, closedLost),
		},
	})
}

// Pipeline returns sponsor counts and values grouped by stage.
func (h *AnalyticsHandler) Pipeline(c fiber.Ctx) error {
	orgID := uuid.MustParse(c.Locals("org_id").(string))

	type StageCount struct {
		Stage string  `json:"stage"`
		Count int64   `json:"count"`
		Value float64 `json:"value"`
	}

	var stages []StageCount
	h.db.Model(&models.Sponsor{}).
		Where("organization_id = ?", orgID).
		Select("stage, COUNT(*) as count, COALESCE(SUM(deal_value), 0) as value").
		Group("stage").
		Scan(&stages)

	return c.JSON(fiber.Map{"data": stages})
}

// Activity returns the 20 most recent activities.
func (h *AnalyticsHandler) Activity(c fiber.Ctx) error {
	orgID := uuid.MustParse(c.Locals("org_id").(string))

	var activities []models.Activity
	h.db.Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Limit(20).
		Preload("User").
		Find(&activities)

	return c.JSON(fiber.Map{"data": activities})
}

// TimeSeries returns daily data points for the requested metric.
// Query params: metric (sponsors|revenue|pipeline_value|contacts|companies|deals_won), period (days, default 30).
func (h *AnalyticsHandler) TimeSeries(c fiber.Ctx) error {
	orgID := uuid.MustParse(c.Locals("org_id").(string))

	metric := c.Query("metric", "sponsors")
	period, _ := strconv.Atoi(c.Query("period", "30"))
	if period < 1 {
		period = 30
	}
	if period > 365 {
		period = 365
	}

	startDate := time.Now().AddDate(0, 0, -period)

	var points []tsDataPoint

	switch metric {
	case "sponsors":
		h.db.Model(&models.Sponsor{}).
			Where("organization_id = ? AND created_at >= ?", orgID, startDate).
			Select("DATE(created_at) as date, COUNT(*) as value").
			Group("DATE(created_at)").
			Order("date").
			Scan(&points)

	case "revenue":
		h.db.Model(&models.Sponsor{}).
			Where("organization_id = ? AND stage = ? AND updated_at >= ?", orgID, "closed_won", startDate).
			Select("DATE(updated_at) as date, COALESCE(SUM(deal_value), 0) as value").
			Group("DATE(updated_at)").
			Order("date").
			Scan(&points)

	case "pipeline_value":
		h.db.Model(&models.Sponsor{}).
			Where("organization_id = ? AND stage NOT IN ? AND created_at >= ?", orgID, []string{"closed_won", "closed_lost"}, startDate).
			Select("DATE(created_at) as date, COALESCE(SUM(deal_value), 0) as value").
			Group("DATE(created_at)").
			Order("date").
			Scan(&points)

	case "contacts":
		h.db.Model(&models.Contact{}).
			Where("organization_id = ? AND created_at >= ?", orgID, startDate).
			Select("DATE(created_at) as date, COUNT(*) as value").
			Group("DATE(created_at)").
			Order("date").
			Scan(&points)

	case "companies":
		h.db.Model(&models.Company{}).
			Where("organization_id = ? AND created_at >= ?", orgID, startDate).
			Select("DATE(created_at) as date, COUNT(*) as value").
			Group("DATE(created_at)").
			Order("date").
			Scan(&points)

	case "deals_won":
		h.db.Model(&models.Sponsor{}).
			Where("organization_id = ? AND stage = ? AND updated_at >= ?", orgID, "closed_won", startDate).
			Select("DATE(updated_at) as date, COUNT(*) as value").
			Group("DATE(updated_at)").
			Order("date").
			Scan(&points)

	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid metric. Use: sponsors, revenue, pipeline_value, contacts, companies, deals_won",
		})
	}

	// Fill gaps so the frontend gets a continuous series
	filled := fillDateGaps(points, startDate, time.Now())

	return c.JSON(fiber.Map{"data": filled, "metric": metric, "period": period})
}

// PipelineFunnel returns ordered stage data with percentages and average days in each stage.
func (h *AnalyticsHandler) PipelineFunnel(c fiber.Ctx) error {
	orgID := uuid.MustParse(c.Locals("org_id").(string))

	stages := models.DefaultPipelineStages

	type FunnelStage struct {
		Stage      string  `json:"stage"`
		Count      int64   `json:"count"`
		Value      float64 `json:"value"`
		Percentage float64 `json:"percentage"`
		AvgDays    float64 `json:"avg_days_in_stage"`
	}

	var totalCount int64
	h.db.Model(&models.Sponsor{}).Where("organization_id = ?", orgID).Count(&totalCount)

	result := make([]FunnelStage, 0, len(stages))
	for _, stage := range stages {
		var count int64
		var value float64
		h.db.Model(&models.Sponsor{}).
			Where("organization_id = ? AND stage = ?", orgID, stage).
			Select("COUNT(*) as count, COALESCE(SUM(deal_value), 0) as value").
			Row().Scan(&count, &value)

		pct := 0.0
		if totalCount > 0 {
			pct = roundTo1(float64(count) / float64(totalCount) * 100)
		}

		var avgDays float64
		h.db.Model(&models.Sponsor{}).
			Where("organization_id = ? AND stage = ?", orgID, stage).
			Select("COALESCE(AVG(EXTRACT(EPOCH FROM (COALESCE(updated_at, NOW()) - stage_entered_at)) / 86400), 0)").
			Scan(&avgDays)

		result = append(result, FunnelStage{
			Stage:      stage,
			Count:      count,
			Value:      value,
			Percentage: pct,
			AvgDays:    roundTo1(avgDays),
		})
	}

	return c.JSON(fiber.Map{"data": result, "total": totalCount})
}

// DealVelocity returns monthly deal velocity metrics for the past 12 months.
func (h *AnalyticsHandler) DealVelocity(c fiber.Ctx) error {
	orgID := uuid.MustParse(c.Locals("org_id").(string))

	type VelocityPoint struct {
		Month       string  `json:"month"`
		AvgDays     float64 `json:"avg_days"`
		DealsWon    int64   `json:"deals_won"`
		TotalValue  float64 `json:"total_value"`
		AvgDealSize float64 `json:"avg_deal_size"`
	}

	var points []VelocityPoint
	h.db.Model(&models.Sponsor{}).
		Where("organization_id = ? AND stage = ? AND updated_at >= ?", orgID, "closed_won", time.Now().AddDate(-1, 0, 0)).
		Select(`
			TO_CHAR(updated_at, 'YYYY-MM') as month,
			AVG(EXTRACT(EPOCH FROM (updated_at - created_at)) / 86400) as avg_days,
			COUNT(*) as deals_won,
			COALESCE(SUM(deal_value), 0) as total_value,
			COALESCE(AVG(deal_value), 0) as avg_deal_size
		`).
		Group("TO_CHAR(updated_at, 'YYYY-MM')").
		Order("month").
		Scan(&points)

	for i := range points {
		points[i].AvgDays = roundTo1(points[i].AvgDays)
		points[i].AvgDealSize = math.Round(points[i].AvgDealSize*100) / 100
	}

	return c.JSON(fiber.Map{"data": points})
}

// ExportSponsors streams a CSV download of all sponsors for the org.
func (h *AnalyticsHandler) ExportSponsors(c fiber.Ctx) error {
	orgID := uuid.MustParse(c.Locals("org_id").(string))

	var sponsors []models.Sponsor
	h.db.Where("organization_id = ?", orgID).
		Preload("Company").
		Preload("Campaign").
		Order("created_at DESC").
		Find(&sponsors)

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=sponsors_export_%s.csv", time.Now().Format("2006-01-02")))

	writer := csv.NewWriter(c.Response().BodyWriter())
	defer writer.Flush()

	headers := []string{"Company", "Campaign", "Stage", "Deal Value", "Tier", "Probability", "Expected Close", "Created At"}
	if err := writer.Write(headers); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to write CSV"})
	}

	for _, s := range sponsors {
		row := []string{
			s.Company.Name,
			s.Campaign.Name,
			s.Stage,
			ptrFloat(s.DealValue),
			ptrStr(s.Tier),
			ptrIntPct(s.Probability),
			ptrStr(s.ExpectedClose),
			s.CreatedAt.Format("2006-01-02"),
		}
		if err := writer.Write(row); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to write CSV row"})
		}
	}

	return nil
}

// ExportCampaigns streams a CSV download of all campaigns for the org.
func (h *AnalyticsHandler) ExportCampaigns(c fiber.Ctx) error {
	orgID := uuid.MustParse(c.Locals("org_id").(string))

	var campaigns []models.Campaign
	h.db.Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Find(&campaigns)

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=campaigns_export_%s.csv", time.Now().Format("2006-01-02")))

	writer := csv.NewWriter(c.Response().BodyWriter())
	defer writer.Flush()

	headers := []string{"Name", "Status", "Goal Amount", "Raised Amount", "Start Date", "End Date", "Created At"}
	if err := writer.Write(headers); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to write CSV"})
	}

	for _, cam := range campaigns {
		row := []string{
			cam.Name,
			cam.Status,
			ptrFloat(cam.GoalAmount),
			fmt.Sprintf("%.2f", cam.RaisedAmount),
			ptrStr(cam.StartDate),
			ptrStr(cam.EndDate),
			cam.CreatedAt.Format("2006-01-02"),
		}
		if err := writer.Write(row); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to write CSV row"})
		}
	}

	return nil
}

// --- helpers ---

func roundTo1(v float64) float64 {
	return math.Round(v*10) / 10
}

func calcWinRate(won, lost int64) float64 {
	total := won + lost
	if total == 0 {
		return 0
	}
	return roundTo1(float64(won) / float64(total) * 100)
}

func ptrFloat(v *float64) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%.2f", *v)
}

func ptrStr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func ptrIntPct(v *int) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%d%%", *v)
}

type tsDataPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

func fillDateGaps(points []tsDataPoint, start, end time.Time) []tsDataPoint {
	lookup := make(map[string]float64, len(points))
	for _, p := range points {
		lookup[p.Date] = p.Value
	}

	var filled []tsDataPoint
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		filled = append(filled, tsDataPoint{
			Date:  dateStr,
			Value: lookup[dateStr],
		})
	}

	return filled
}
