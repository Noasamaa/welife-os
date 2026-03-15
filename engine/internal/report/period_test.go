package report

import "testing"

func TestResolvePeriodNormalizesDateOnlyBounds(t *testing.T) {
	period, err := ResolvePeriod("weekly", "2026-03-01", "2026-03-03")
	if err != nil {
		t.Fatalf("resolve period: %v", err)
	}

	if period.Start != "2026-03-01T00:00:00Z" {
		t.Fatalf("unexpected start: %s", period.Start)
	}
	if period.End != "2026-03-03T23:59:59Z" {
		t.Fatalf("unexpected end: %s", period.End)
	}
}

func TestResolvePeriodRejectsInvertedRange(t *testing.T) {
	if _, err := ResolvePeriod("weekly", "2026-03-03", "2026-03-01"); err == nil {
		t.Fatal("expected inverted range to fail")
	}
}
