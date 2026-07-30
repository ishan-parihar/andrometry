package db

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/andrometry/server/internal/models"
)

// SummaryGenerator generates daily and weekly summaries from timeline segments.
type SummaryGenerator struct {
	db *DB
}

// NewSummaryGenerator creates a new SummaryGenerator.
func NewSummaryGenerator(db *DB) *SummaryGenerator {
	return &SummaryGenerator{db: db}
}

// DailySummaryData represents a computed daily summary.
type DailySummaryData struct {
	Date                  string             `json:"date"`
	Timezone              string             `json:"timezone"`
	DeepWorkMinutes       int                `json:"deep_work_minutes"`
	ShallowWorkMinutes    int                `json:"shallow_work_minutes"`
	MeetingMinutes        int                `json:"meeting_minutes"`
	CommunicationMinutes  int                `json:"communication_minutes"`
	LearningMinutes       int                `json:"learning_minutes"`
	AdminMinutes          int                `json:"admin_minutes"`
	PlanningMinutes       int                `json:"planning_minutes"`
	BreakMinutes          int                `json:"break_minutes"`
	LeisureMinutes        int                `json:"leisure_minutes"`
	SocialMinutes         int                `json:"social_minutes"`
	DistractionMinutes    int                `json:"distraction_minutes"`
	ExerciseMinutes       int                `json:"exercise_minutes"`
	CommuteMinutes        int                `json:"commute_minutes"`
	SleepMinutes          int                `json:"sleep_minutes"`
	UnknownMinutes        int                `json:"unknown_minutes"`
	TotalTrackedMinutes   int                `json:"total_tracked_minutes"`
	SegmentCount          int                `json:"segment_count"`
	AvgConfidence         float64            `json:"avg_confidence"`
	AvgDistractionPerHour float64            `json:"avg_distraction_per_hour"`
	AlignmentScore        float64            `json:"alignment_score"`
	ActivityBreakdown     map[string]int     `json:"activity_breakdown"`
	FocusSessions         int                `json:"focus_sessions"`
	DistractionSegments   int                `json:"distraction_segments"`
}

// WeeklySummaryData represents a computed weekly summary.
type WeeklySummaryData struct {
	Week                  string             `json:"week"`
	Timezone              string             `json:"timezone"`
	Days                  int                `json:"days"`
	DeepWorkMinutes       int                `json:"deep_work_minutes"`
	MeetingMinutes        int                `json:"meeting_minutes"`
	DistractionMinutes    int                `json:"distraction_minutes"`
	TotalTrackedMinutes   int                `json:"total_tracked_minutes"`
	AvgDailyDeepWork      float64            `json:"avg_daily_deep_work"`
	AvgDailyMeetings      float64            `json:"avg_daily_meetings"`
	AvgDailyDistraction   float64            `json:"avg_daily_distraction"`
	AlignmentScore        float64            `json:"alignment_score"`
	ActivityBreakdown     map[string]int     `json:"activity_breakdown"`
}

// GenerateDailySummary computes a daily summary from segments for a given date and user.
func (sg *SummaryGenerator) GenerateDailySummary(userID, date, timezone string) error {
	// Query segments for the day
	from := date + "T00:00:00Z"
	to := date + "T23:59:59Z"

	segments, err := sg.db.GetTimeline(userID, from, to, 10000, "")
	if err != nil {
		return fmt.Errorf("query segments: %w", err)
	}

	if len(segments) == 0 {
		return nil // No segments to summarize
	}

	// Compute breakdown
	summary := &DailySummaryData{
		Date:              date,
		Timezone:          timezone,
		ActivityBreakdown: make(map[string]int),
		SegmentCount:      len(segments),
	}

	totalConfidence := 0.0
	for _, seg := range segments {
		minutes := segmentDurationMinutes(seg.StartTime, seg.EndTime)
		if minutes <= 0 {
			continue
		}

		totalConfidence += seg.Confidence
		summary.ActivityBreakdown[seg.Label] += minutes
		summary.TotalTrackedMinutes += minutes

		switch seg.Label {
		case "deep_work":
			summary.DeepWorkMinutes += minutes
			summary.FocusSessions++
		case "shallow_work":
			summary.ShallowWorkMinutes += minutes
		case "meeting":
			summary.MeetingMinutes += minutes
		case "communication":
			summary.CommunicationMinutes += minutes
		case "learning":
			summary.LearningMinutes += minutes
		case "admin":
			summary.AdminMinutes += minutes
		case "planning":
			summary.PlanningMinutes += minutes
		case "break":
			summary.BreakMinutes += minutes
		case "leisure":
			summary.LeisureMinutes += minutes
		case "social":
			summary.SocialMinutes += minutes
		case "distraction":
			summary.DistractionMinutes += minutes
			summary.DistractionSegments++
		case "exercise":
			summary.ExerciseMinutes += minutes
		case "commute":
			summary.CommuteMinutes += minutes
		case "sleep":
			summary.SleepMinutes += minutes
		default:
			summary.UnknownMinutes += minutes
		}
	}

	if len(segments) > 0 {
		summary.AvgConfidence = totalConfidence / float64(len(segments))
	}

	// Alignment score: (deep_work + shallow_work + meeting + learning + planning) / total
	productive := summary.DeepWorkMinutes + summary.ShallowWorkMinutes + summary.MeetingMinutes + summary.LearningMinutes + summary.PlanningMinutes
	if summary.TotalTrackedMinutes > 0 {
		summary.AlignmentScore = float64(productive) / float64(summary.TotalTrackedMinutes)
	}

	// Distraction per hour
	if summary.TotalTrackedMinutes > 0 {
		hours := float64(summary.TotalTrackedMinutes) / 60.0
		summary.AvgDistractionPerHour = float64(summary.DistractionMinutes) / hours
	}

	// Store the summary
	summaryBytes, _ := json.Marshal(summary)
	return sg.db.UpsertDailySummary(&models.DailySummary{
		UserID:   userID,
		Date:     date,
		Timezone: timezone,
		Summary:  string(summaryBytes),
		Version:  1,
	})
}

// GenerateWeeklySummary computes a weekly summary for a given ISO week and user.
func (sg *SummaryGenerator) GenerateWeeklySummary(userID, week, timezone string) error {
	// Parse the week string to get date range
	year, w := parseWeek(week)
	if year == 0 || w == 0 {
		return fmt.Errorf("invalid week format: %s", week)
	}

	// Get all daily summaries for this week
	dailySummaries, err := sg.db.GetDailySummariesForWeek(userID, year, w)
	if err != nil {
		return fmt.Errorf("query daily summaries: %w", err)
	}

	if len(dailySummaries) == 0 {
		return nil // No daily summaries to aggregate
	}

	summary := &WeeklySummaryData{
		Week:                week,
		Timezone:            timezone,
		Days:                len(dailySummaries),
		ActivityBreakdown:   make(map[string]int),
	}

	for _, ds := range dailySummaries {
		var daily DailySummaryData
		if err := json.Unmarshal([]byte(ds.Summary), &daily); err != nil {
			continue
		}

		summary.DeepWorkMinutes += daily.DeepWorkMinutes
		summary.MeetingMinutes += daily.MeetingMinutes
		summary.DistractionMinutes += daily.DistractionMinutes
		summary.TotalTrackedMinutes += daily.TotalTrackedMinutes

		for label, mins := range daily.ActivityBreakdown {
			summary.ActivityBreakdown[label] += mins
		}
	}

	if len(dailySummaries) > 0 {
		days := float64(len(dailySummaries))
		summary.AvgDailyDeepWork = float64(summary.DeepWorkMinutes) / days
		summary.AvgDailyMeetings = float64(summary.MeetingMinutes) / days
		summary.AvgDailyDistraction = float64(summary.DistractionMinutes) / days
	}

	if summary.TotalTrackedMinutes > 0 {
		productive := summary.DeepWorkMinutes + summary.MeetingMinutes
		summary.AlignmentScore = float64(productive) / float64(summary.TotalTrackedMinutes)
	}

	summaryBytes, _ := json.Marshal(summary)
	return sg.db.UpsertWeeklySummary(&models.WeeklySummary{
		UserID:   userID,
		Week:     week,
		Timezone: timezone,
		Summary:  string(summaryBytes),
		Version:  1,
	})
}

// segmentDurationMinutes computes the duration in minutes between two timestamps.
func segmentDurationMinutes(start, end string) int {
	t1, err1 := time.Parse(time.RFC3339, start)
	t2, err2 := time.Parse(time.RFC3339, end)
	if err1 != nil || err2 != nil {
		return 0
	}
	return int(t2.Sub(t1).Minutes())
}

// parseWeek parses an ISO week string like "2026-W31" into year and week number.
func parseWeek(week string) (int, int) {
	var year, w int
	_, err := fmt.Sscanf(week, "%d-W%d", &year, &w)
	if err != nil {
		return 0, 0
	}
	return year, w
}
