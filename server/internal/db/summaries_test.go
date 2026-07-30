package db

import (
	"testing"
	"time"
)

func TestSegmentDurationMinutes(t *testing.T) {
	tests := []struct {
		name     string
		start    string
		end      string
		expected int
	}{
		{
			name:     "30 minute segment",
			start:    "2026-07-28T10:00:00Z",
			end:      "2026-07-28T10:30:00Z",
			expected: 30,
		},
		{
			name:     "1 hour segment",
			start:    "2026-07-28T09:00:00Z",
			end:      "2026-07-28T10:00:00Z",
			expected: 60,
		},
	{
		name:     "negative duration",
		start:    "2026-07-28T10:30:00Z",
		end:      "2026-07-28T10:00:00Z",
		expected: -30,
	},
	{
		name:     "timezone offset",
		start:    "2026-07-28T10:00:00+05:30",
		end:      "2026-07-28T11:00:00+05:30",
		expected: 60,
	},
	{
		name:     "cross timezone",
		start:    "2026-07-28T10:00:00+05:30",
		end:      "2026-07-28T04:30:00Z",
		expected: 0,
	},
		{
			name:     "invalid timestamp",
			start:    "invalid",
			end:      "2026-07-28T10:00:00Z",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := segmentDurationMinutes(tt.start, tt.end)
			if result != tt.expected {
				t.Errorf("segmentDurationMinutes(%q, %q) = %d, want %d", tt.start, tt.end, result, tt.expected)
			}
		})
	}
}

func TestParseWeek(t *testing.T) {
	tests := []struct {
		name      string
		week      string
		wantYear  int
		wantWeek  int
	}{
		{
			name:     "valid week",
			week:     "2026-W31",
			wantYear: 2026,
			wantWeek: 31,
		},
		{
			name:     "week 1",
			week:     "2026-W01",
			wantYear: 2026,
			wantWeek: 1,
		},
		{
			name:     "invalid format",
			week:     "invalid",
			wantYear: 0,
			wantWeek: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			year, week := parseWeek(tt.week)
			if year != tt.wantYear || week != tt.wantWeek {
				t.Errorf("parseWeek(%q) = (%d, %d), want (%d, %d)", tt.week, year, week, tt.wantYear, tt.wantWeek)
			}
		})
	}
}

func TestIsoWeekStart(t *testing.T) {
	tests := []struct {
		name     string
		year     int
		week     int
		expected time.Time
	}{
		{
			name:     "week 1 of 2026",
			year:     2026,
			week:     1,
			expected: time.Date(2025, 12, 29, 0, 0, 0, 0, time.UTC), // Monday of week 1
		},
		{
			name:     "week 31 of 2026",
			year:     2026,
			week:     31,
			expected: time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC), // Monday of week 31
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isoWeekStart(tt.year, tt.week)
			if !result.Equal(tt.expected) {
				t.Errorf("isoWeekStart(%d, %d) = %v, want %v", tt.year, tt.week, result, tt.expected)
			}
		})
	}
}

func TestWeekDateRange(t *testing.T) {
	start, end := weekDateRange(2026, 31)
	if start != "2026-07-27" {
		t.Errorf("weekDateRange start = %q, want %q", start, "2026-07-27")
	}
	if end != "2026-08-02" {
		t.Errorf("weekDateRange end = %q, want %q", end, "2026-08-02")
	}
}

func TestDailySummaryData_Fields(t *testing.T) {
	summary := &DailySummaryData{
		Date:                  "2026-07-28",
		Timezone:              "UTC",
		DeepWorkMinutes:       120,
		ShallowWorkMinutes:    60,
		MeetingMinutes:        30,
		CommunicationMinutes:  15,
		LearningMinutes:       45,
		AdminMinutes:          20,
		PlanningMinutes:       25,
		BreakMinutes:          10,
		LeisureMinutes:        30,
		SocialMinutes:         15,
		DistractionMinutes:    20,
		ExerciseMinutes:       0,
		CommuteMinutes:        0,
		SleepMinutes:          0,
		UnknownMinutes:        10,
		TotalTrackedMinutes:   400,
		SegmentCount:          12,
		AvgConfidence:         0.85,
		AvgDistractionPerHour: 3.0,
		AlignmentScore:        0.75,
		ActivityBreakdown:     map[string]int{"deep_work": 120, "shallow_work": 60},
		FocusSessions:         3,
		DistractionSegments:   2,
	}

	if summary.Date != "2026-07-28" {
		t.Errorf("Date = %q, want %q", summary.Date, "2026-07-28")
	}
	if summary.DeepWorkMinutes != 120 {
		t.Errorf("DeepWorkMinutes = %d, want %d", summary.DeepWorkMinutes, 120)
	}
	if summary.AlignmentScore != 0.75 {
		t.Errorf("AlignmentScore = %f, want %f", summary.AlignmentScore, 0.75)
	}
}

func TestWeeklySummaryData_Fields(t *testing.T) {
	summary := &WeeklySummaryData{
		Week:                "2026-W31",
		Timezone:            "UTC",
		Days:                5,
		DeepWorkMinutes:     600,
		MeetingMinutes:      150,
		DistractionMinutes:  100,
		TotalTrackedMinutes: 2000,
		AvgDailyDeepWork:    120.0,
		AvgDailyMeetings:    30.0,
		AvgDailyDistraction: 20.0,
		AlignmentScore:      0.70,
		ActivityBreakdown:   map[string]int{"deep_work": 600, "meeting": 150},
	}

	if summary.Week != "2026-W31" {
		t.Errorf("Week = %q, want %q", summary.Week, "2026-W31")
	}
	if summary.Days != 5 {
		t.Errorf("Days = %d, want %d", summary.Days, 5)
	}
	if summary.AlignmentScore != 0.70 {
		t.Errorf("AlignmentScore = %f, want %f", summary.AlignmentScore, 0.70)
	}
}
