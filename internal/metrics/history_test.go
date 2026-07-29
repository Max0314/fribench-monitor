package metrics

import (
	"testing"
	"time"
)

func TestHistoryPrunesAndReturnsCopies(t *testing.T) {
	history := NewHistory(10 * time.Minute)
	now := time.Now()
	history.Add(Point{Timestamp: now.Add(-11 * time.Minute), CPUPercent: 1})
	history.Add(Point{Timestamp: now.Add(-5 * time.Minute), CPUPercent: 2})
	history.Add(Point{Timestamp: now, CPUPercent: 3})

	points := history.Since(now.Add(-30 * time.Minute))
	if len(points) != 2 {
		t.Fatalf("history length = %d, want 2", len(points))
	}

	points[0].CPUPercent = 99
	fresh := history.Since(now.Add(-30 * time.Minute))
	if fresh[0].CPUPercent != 2 {
		t.Fatalf("history result aliases internal data")
	}
}
