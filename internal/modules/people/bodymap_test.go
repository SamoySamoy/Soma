package people

import (
	"testing"
	"time"
)

func TestDaysUntilBirthday(t *testing.T) {
	t.Parallel()

	date := func(y int, m time.Month, d int) time.Time {
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}
	tests := []struct {
		name     string
		now      time.Time
		birthday time.Time
		want     int
	}{
		{"birthday today is zero", date(2026, 10, 6), date(1990, 10, 6), 0},
		{"three days ahead", date(2026, 10, 6), date(1990, 10, 9), 3},
		{"already passed this year wraps to next year", date(2026, 10, 6), date(1990, 10, 1), 360},
		{"crosses a year end", date(2026, 12, 30), date(1990, 1, 2), 3},
		{"leap-day birthday in a common year falls on March 1", date(2026, 2, 27), date(1992, 2, 29), 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := daysUntilBirthday(tt.now, tt.birthday); got != tt.want {
				t.Errorf("daysUntilBirthday() = %d, want %d", got, tt.want)
			}
		})
	}
}
