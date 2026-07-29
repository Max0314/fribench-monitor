package monitor

import (
	"testing"
	"time"

	"github.com/Max0314/fribench-monitor/internal/metrics"
)

func TestHistoryDownsamplesAndKeepsEndpoints(t *testing.T) {
	state := NewState(5*time.Second, time.Hour)
	now := time.Now()
	for index := 0; index < 1000; index++ {
		state.AddHistory(metrics.Point{
			Timestamp:  now.Add(time.Duration(index) * time.Second),
			CPUPercent: float64(index),
		})
	}

	points := state.History(now.Add(-time.Second), 100)
	if len(points) != 100 {
		t.Fatalf("downsampled points = %d, want 100", len(points))
	}
	if points[0].CPUPercent != 0 || points[len(points)-1].CPUPercent != 999 {
		t.Fatalf("downsample endpoints = %f, %f", points[0].CPUPercent, points[len(points)-1].CPUPercent)
	}
}

func TestHistoryDownsampleToSinglePointKeepsLatest(t *testing.T) {
	state := NewState(5*time.Second, time.Hour)
	now := time.Now()
	for index := 0; index < 3; index++ {
		state.AddHistory(metrics.Point{
			Timestamp:  now.Add(time.Duration(index) * time.Second),
			CPUPercent: float64(index),
		})
	}

	points := state.History(now.Add(-time.Second), 1)
	if len(points) != 1 || points[0].CPUPercent != 2 {
		t.Fatalf("single point = %#v, want latest point", points)
	}
}
