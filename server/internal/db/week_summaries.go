package db

import (
	"time"

	"github.com/andrometry/server/internal/models"
)

// GetDailySummariesForWeek retrieves all daily summaries for a given ISO week.
func (db *DB) GetDailySummariesForWeek(userID string, year, week int) ([]models.DailySummary, error) {
	startDate, endDate := weekDateRange(year, week)

	rows, err := db.Query(`
		SELECT user_id, date, timezone, summary, version, updated_at
		FROM daily_summary
		WHERE user_id = ? AND date >= ? AND date <= ?
		ORDER BY date ASC
	`, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []models.DailySummary
	for rows.Next() {
		var s models.DailySummary
		if err := rows.Scan(&s.UserID, &s.Date, &s.Timezone, &s.Summary, &s.Version, &s.UpdatedAt); err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

// weekDateRange returns the start and end dates for an ISO week.
func weekDateRange(year, week int) (string, string) {
	start := isoWeekStart(year, week)
	end := start.AddDate(0, 0, 6)
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}

// isoWeekStart returns the Monday of the given ISO week.
func isoWeekStart(year, week int) time.Time {
	// January 4 is always in week 1. Find the Monday of week 1.
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, time.UTC)
	weekday := int(jan4.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7
	}
	week1Monday := jan4.AddDate(0, 0, -weekday+1) // Monday of week 1
	return week1Monday.AddDate(0, 0, (week-1)*7)
}
