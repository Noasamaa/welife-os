package report

import (
	"fmt"
	"time"
)

// ResolvePeriod combines defaults with optional request overrides and normalizes
// the final time range to RFC3339 so downstream storage filters behave predictably.
func ResolvePeriod(reportType string, startInput string, endInput string) (ReportPeriod, error) {
	period := DefaultPeriod(reportType)
	if startInput != "" {
		period.Start = startInput
	}
	if endInput != "" {
		period.End = endInput
	}

	start, err := ParseFlexibleTime(period.Start, false)
	if err != nil {
		return ReportPeriod{}, fmt.Errorf("invalid period_start: %w", err)
	}
	end, err := ParseFlexibleTime(period.End, true)
	if err != nil {
		return ReportPeriod{}, fmt.Errorf("invalid period_end: %w", err)
	}
	if end.Before(start) {
		return ReportPeriod{}, fmt.Errorf("period_end must be after period_start")
	}

	return ReportPeriod{
		Start: start.Format(time.RFC3339),
		End:   end.Format(time.RFC3339),
	}, nil
}

// ParseFlexibleTime accepts RFC3339, SQLite timestamps, and date-only strings.
// Date-only end bounds are expanded to the last second of that day.
func ParseFlexibleTime(value string, endOfDay bool) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		parsed, err := time.Parse(format, value)
		if err != nil {
			continue
		}
		if format == "2006-01-02" && endOfDay {
			return parsed.Add(23*time.Hour + 59*time.Minute + 59*time.Second), nil
		}
		return parsed, nil
	}

	return time.Time{}, fmt.Errorf("unsupported timestamp %q", value)
}
