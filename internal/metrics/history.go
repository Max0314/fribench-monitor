package metrics

import (
	"sort"
	"sync"
	"time"
)

type History struct {
	mu        sync.RWMutex
	retention time.Duration
	points    []Point
}

func NewHistory(retention time.Duration) *History {
	return &History{retention: retention}
}

func (history *History) Add(point Point) {
	history.mu.Lock()
	defer history.mu.Unlock()

	history.points = append(history.points, point)
	history.pruneLocked(point.Timestamp.Add(-history.retention))
}

func (history *History) Since(since time.Time) []Point {
	history.mu.RLock()
	defer history.mu.RUnlock()

	index := sort.Search(len(history.points), func(index int) bool {
		return !history.points[index].Timestamp.Before(since)
	})

	result := make([]Point, len(history.points)-index)
	copy(result, history.points[index:])
	return result
}

func (history *History) Len() int {
	history.mu.RLock()
	defer history.mu.RUnlock()
	return len(history.points)
}

func (history *History) pruneLocked(cutoff time.Time) {
	index := sort.Search(len(history.points), func(index int) bool {
		return !history.points[index].Timestamp.Before(cutoff)
	})
	if index == 0 {
		return
	}

	copy(history.points, history.points[index:])
	history.points = history.points[:len(history.points)-index]
}
