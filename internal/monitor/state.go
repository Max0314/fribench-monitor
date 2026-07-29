package monitor

import (
	"sync"
	"time"

	"github.com/Max0314/fribench-monitor/internal/metrics"
)

type State struct {
	mu             sync.RWMutex
	snapshot       metrics.Snapshot
	hasSnapshot    bool
	history        *metrics.History
	sampleInterval time.Duration
	retention      time.Duration
}

func NewState(sampleInterval, retention time.Duration) *State {
	return &State{
		history:        metrics.NewHistory(retention),
		sampleInterval: sampleInterval,
		retention:      retention,
	}
}

func (state *State) Update(snapshot metrics.Snapshot) {
	state.mu.Lock()
	state.snapshot = snapshot
	state.hasSnapshot = true
	state.mu.Unlock()
	state.history.Add(metrics.PointFromSnapshot(snapshot))
}

func (state *State) AddHistory(point metrics.Point) {
	state.history.Add(point)
}

func (state *State) Snapshot() (metrics.Snapshot, bool) {
	state.mu.RLock()
	defer state.mu.RUnlock()
	return state.snapshot, state.hasSnapshot
}

func (state *State) History(since time.Time, maxPoints int) []metrics.Point {
	points := state.history.Since(since)
	return downsample(points, maxPoints)
}

func (state *State) SampleInterval() time.Duration {
	return state.sampleInterval
}

func (state *State) Retention() time.Duration {
	return state.retention
}

func downsample(points []metrics.Point, maxPoints int) []metrics.Point {
	if maxPoints <= 0 || len(points) <= maxPoints {
		return points
	}
	if maxPoints == 1 {
		return []metrics.Point{points[len(points)-1]}
	}

	result := make([]metrics.Point, 0, maxPoints)
	stride := float64(len(points)-1) / float64(maxPoints-1)
	lastIndex := -1
	for index := 0; index < maxPoints; index++ {
		sourceIndex := int(float64(index) * stride)
		if index == maxPoints-1 {
			sourceIndex = len(points) - 1
		}
		if sourceIndex == lastIndex {
			continue
		}
		result = append(result, points[sourceIndex])
		lastIndex = sourceIndex
	}
	return result
}
