package event

import (
	"testing"
	"time"
)

func TestResolveDashboardRange(t *testing.T) {
	now := time.Date(2026, 4, 8, 15, 30, 45, 0, time.UTC)
	todayStart := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC).UnixMilli()

	t.Run("today", func(t *testing.T) {
		fromMs, toMs := resolveDashboardRange(now, todayRangeDays)

		if fromMs != todayStart {
			t.Fatalf("fromMs = %d, want %d", fromMs, todayStart)
		}

		if toMs != todayStart {
			t.Fatalf("toMs = %d, want %d", toMs, todayStart)
		}
	})

	t.Run("seven days includes today", func(t *testing.T) {
		fromMs, toMs := resolveDashboardRange(now, 7)
		wantFrom := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC).UnixMilli()

		if fromMs != wantFrom {
			t.Fatalf("fromMs = %d, want %d", fromMs, wantFrom)
		}

		if toMs != todayStart {
			t.Fatalf("toMs = %d, want %d", toMs, todayStart)
		}
	})

	t.Run("all time ends today", func(t *testing.T) {
		fromMs, toMs := resolveDashboardRange(now, allTimeRangeDays)

		if fromMs != 0 {
			t.Fatalf("fromMs = %d, want 0", fromMs)
		}

		if toMs != todayStart {
			t.Fatalf("toMs = %d, want %d", toMs, todayStart)
		}
	})
}
