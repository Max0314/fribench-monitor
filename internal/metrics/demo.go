package metrics

import (
	"context"
	"math"
	"strconv"
	"sync"
	"time"
)

const gib = uint64(1024 * 1024 * 1024)

type DemoCollector struct {
	mu      sync.Mutex
	started time.Time
	step    int
}

func NewDemoCollector() *DemoCollector {
	return &DemoCollector{started: time.Now().Add(-(6*24*time.Hour + 4*time.Hour + 12*time.Minute))}
}

func (collector *DemoCollector) Collect(context.Context) (Snapshot, error) {
	collector.mu.Lock()
	defer collector.mu.Unlock()

	now := time.Now()
	wave := math.Sin(float64(collector.step) / 3)
	secondary := math.Sin(float64(collector.step) / 7)
	collector.step++

	totalMemory := 15 * gib / 2
	usedMemory := uint64(float64(gib) * (1.18 + wave*0.025))
	totalDisk := uint64(118 * gib)
	usedDisk := uint64(23 * gib)
	swapTotal := 19 * gib / 10
	swapUsed := uint64(268 * 1024)

	containers := demoContainers(now.Sub(collector.started).Seconds(), wave)

	return Snapshot{
		Timestamp: now,
		Source:    "demo",
		Hostname:  "fribench",
		Kernel:    "6.8.0-136-generic",
		Timezone:  "Asia/Shanghai",
		Uptime:    now.Sub(collector.started).Seconds(),
		CPU: CPU{
			UsagePercent: 3.1 + wave*0.75 + secondary*0.35,
			LogicalCores: 4,
			Load1:        0.11 + wave*0.018,
			Load5:        0.17,
			Load15:       0.12,
			Pressure:     Pressure{SomeAvg10: 0.08},
		},
		Memory: Memory{
			TotalBytes:     totalMemory,
			UsedBytes:      usedMemory,
			AvailableBytes: totalMemory - usedMemory,
			UsagePercent:   percent(usedMemory, totalMemory),
			SwapTotalBytes: swapTotal,
			SwapUsedBytes:  swapUsed,
			SwapPercent:    percent(swapUsed, swapTotal),
		},
		Disk: Disk{
			Available:     true,
			TotalBytes:    totalDisk,
			UsedBytes:     usedDisk,
			FreeBytes:     totalDisk - usedDisk,
			UsagePercent:  percent(usedDisk, totalDisk),
			InodesTotal:   7_500_000,
			InodesUsed:    519_000,
			InodesPercent: percent(519_000, 7_500_000),
		},
		Network: Network{
			ReceivedBytes: 8*gib + uint64(collector.step*32_000),
			SentBytes:     3*gib + uint64(collector.step*12_000),
			ReceiveBPS:    8_000 + (wave+1)*4_000,
			SendBPS:       2_400 + (secondary+1)*1_200,
		},
		Containers: ContainerSummary{
			Available: true,
			Running:   len(containers),
			Healthy:   len(containers),
			Items:     containers,
		},
	}, nil
}

func DemoPoints(now time.Time, duration, interval time.Duration) []Point {
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	count := int(duration / interval)
	points := make([]Point, 0, count)

	for index := count; index > 0; index-- {
		timestamp := now.Add(-time.Duration(index) * interval)
		phase := float64(count-index) / 7
		points = append(points, Point{
			Timestamp:         timestamp,
			CPUPercent:        3.1 + math.Sin(phase)*0.7 + math.Sin(phase/3)*0.35,
			MemoryPercent:     15.8 + math.Sin(phase/5)*0.24,
			DiskPercent:       19.5,
			SwapPercent:       0.02,
			Load1:             0.11 + math.Sin(phase/4)*0.02,
			NetworkReceiveBPS: 8_000 + (math.Sin(phase)+1)*4_000,
			NetworkSendBPS:    2_400 + (math.Cos(phase)+1)*1_200,
		})
	}
	return points
}

func demoContainers(uptime float64, wave float64) []Container {
	items := []struct {
		name    string
		memory  uint64
		cpu     float64
		process string
	}{
		{"pick-meal-well", 128 * 1024 * 1024, 0.18, "next-server"},
		{"fribench-platform-postgres", 66 * 1024 * 1024, 0.04, "postgres"},
		{"fribench-platform-redis", 32 * 1024 * 1024, 0.22, "redis-server"},
		{"pick-meal-well-postgres", 24 * 1024 * 1024, 0.03, "postgres"},
		{"codex-continuity", 6 * 1024 * 1024, 0.04, "continuity-server"},
		{"fribench-portal", 5 * 1024 * 1024, 0.01, "nginx"},
		{"shadow-encore", 5 * 1024 * 1024, 0.01, "nginx"},
		{"twilight-leap", 5 * 1024 * 1024, 0.01, "nginx"},
		{"cloudtrace", 5 * 1024 * 1024, 0.01, "nginx"},
		{"void-lancer", 5 * 1024 * 1024, 0.01, "nginx"},
		{"pick-meal-well-redis", 4 * 1024 * 1024, 0.08, "redis-server"},
	}

	result := make([]Container, 0, len(items))
	for index, item := range items {
		result = append(result, Container{
			ID:            "demo-" + strconv.Itoa(index+1),
			Name:          item.name,
			State:         "running",
			Health:        "healthy",
			CPUPercent:    math.Max(0, item.cpu+wave*0.015),
			MemoryBytes:   item.memory,
			PIDs:          6,
			UptimeSeconds: uptime - float64(index*1800),
			MainProcess:   item.process,
		})
	}
	return result
}
