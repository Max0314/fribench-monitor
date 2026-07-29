package metrics

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Max0314/fribench-monitor/internal/dockerapi"
)

type Source interface {
	Collect(context.Context) (Snapshot, error)
}

type CollectorOptions struct {
	Hostname      string
	ProcRoot      string
	CgroupRoot    string
	RootFSProbe   string
	Timezone      string
	DockerEnabled bool
	DockerSocket  string
}

type LinuxCollector struct {
	mu sync.Mutex

	options CollectorOptions
	docker  *dockerapi.Client

	lastCPU             cpuTimes
	lastNetwork         networkTotals
	lastCollectedAt     time.Time
	lastContainerCPU    map[string]uint64
	dockerContainers    map[string]dockerapi.Container
	lastDockerRefreshAt time.Time
	lastDockerError     error
}

type cpuTimes struct {
	total uint64
	idle  uint64
}

type networkTotals struct {
	received uint64
	sent     uint64
}

func NewLinuxCollector(options CollectorOptions) *LinuxCollector {
	collector := &LinuxCollector{
		options:          options,
		lastContainerCPU: make(map[string]uint64),
		dockerContainers: make(map[string]dockerapi.Container),
	}
	if options.DockerEnabled {
		collector.docker = dockerapi.New(options.DockerSocket)
	}
	return collector
}

func (collector *LinuxCollector) Collect(ctx context.Context) (Snapshot, error) {
	collector.mu.Lock()
	defer collector.mu.Unlock()

	now := time.Now()
	snapshot := Snapshot{
		Timestamp: now,
		Source:    "live",
		Timezone:  collector.options.Timezone,
	}

	var criticalErrors []error

	snapshot.Hostname = collector.options.Hostname
	if snapshot.Hostname == "" {
		snapshot.Hostname, _ = readTrimmed(filepath.Join(collector.options.ProcRoot, "sys/kernel/hostname"))
	}
	snapshot.Kernel, _ = readTrimmed(filepath.Join(collector.options.ProcRoot, "sys/kernel/osrelease"))
	snapshot.Uptime, _ = readUptime(filepath.Join(collector.options.ProcRoot, "uptime"))

	currentCPU, err := readCPU(filepath.Join(collector.options.ProcRoot, "stat"))
	if err != nil {
		criticalErrors = append(criticalErrors, fmt.Errorf("cpu: %w", err))
	} else {
		snapshot.CPU.UsagePercent = cpuPercent(collector.lastCPU, currentCPU)
		collector.lastCPU = currentCPU
	}

	snapshot.CPU.LogicalCores, _ = readLogicalCores(filepath.Join(collector.options.ProcRoot, "cpuinfo"))
	snapshot.CPU.Load1, snapshot.CPU.Load5, snapshot.CPU.Load15, _ = readLoad(
		filepath.Join(collector.options.ProcRoot, "loadavg"),
	)
	snapshot.CPU.Pressure, _ = readPressure(filepath.Join(collector.options.ProcRoot, "pressure/cpu"))

	snapshot.Memory, err = readMemory(filepath.Join(collector.options.ProcRoot, "meminfo"))
	if err != nil {
		criticalErrors = append(criticalErrors, fmt.Errorf("memory: %w", err))
	}
	snapshot.Memory.Pressure, _ = readPressure(filepath.Join(collector.options.ProcRoot, "pressure/memory"))

	snapshot.Disk, err = diskUsage(collector.options.RootFSProbe)
	if err == nil {
		snapshot.Disk.Available = true
	}
	snapshot.Disk.Pressure, _ = readPressure(filepath.Join(collector.options.ProcRoot, "pressure/io"))

	networkRoot := filepath.Join(collector.options.ProcRoot, "1/net")
	networkPath := filepath.Join(networkRoot, "dev")
	if _, statErr := os.Stat(networkPath); statErr != nil {
		networkRoot = filepath.Join(collector.options.ProcRoot, "net")
		networkPath = filepath.Join(networkRoot, "dev")
	}
	defaultInterface, _ := readDefaultInterface(filepath.Join(networkRoot, "route"))
	currentNetwork, err := readNetwork(networkPath, defaultInterface)
	if err == nil {
		snapshot.Network.ReceivedBytes = currentNetwork.received
		snapshot.Network.SentBytes = currentNetwork.sent
		if !collector.lastCollectedAt.IsZero() {
			elapsed := now.Sub(collector.lastCollectedAt).Seconds()
			if elapsed > 0 {
				snapshot.Network.ReceiveBPS = rate(collector.lastNetwork.received, currentNetwork.received, elapsed)
				snapshot.Network.SendBPS = rate(collector.lastNetwork.sent, currentNetwork.sent, elapsed)
			}
		}
		collector.lastNetwork = currentNetwork
	}

	snapshot.Containers = collector.collectContainers(ctx, now)
	collector.lastCollectedAt = now

	if len(criticalErrors) > 0 {
		return snapshot, errors.Join(criticalErrors...)
	}
	return snapshot, nil
}

func (collector *LinuxCollector) collectContainers(ctx context.Context, now time.Time) ContainerSummary {
	if !collector.options.DockerEnabled {
		return ContainerSummary{Available: false}
	}

	if collector.docker != nil &&
		(collector.lastDockerRefreshAt.IsZero() || now.Sub(collector.lastDockerRefreshAt) >= 30*time.Second) {
		containers, err := collector.docker.Containers(ctx)
		collector.lastDockerRefreshAt = now
		collector.lastDockerError = err
		if err == nil {
			collector.dockerContainers = make(map[string]dockerapi.Container, len(containers))
			for _, container := range containers {
				collector.dockerContainers[container.ID] = container
			}
		}
	}

	scopeRoot := filepath.Join(collector.options.CgroupRoot, "system.slice")
	entries, err := os.ReadDir(scopeRoot)
	if err != nil {
		return ContainerSummary{
			Available: false,
			Error:     conciseError(errors.Join(collector.lastDockerError, err)),
		}
	}

	elapsed := now.Sub(collector.lastCollectedAt).Seconds()
	items := make([]Container, 0, len(entries))
	activeIDs := make(map[string]struct{})

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "docker-") || !strings.HasSuffix(entry.Name(), ".scope") {
			continue
		}

		id := strings.TrimSuffix(strings.TrimPrefix(entry.Name(), "docker-"), ".scope")
		activeIDs[id] = struct{}{}
		scope := filepath.Join(scopeRoot, entry.Name())

		memoryBytes, _ := readUint(filepath.Join(scope, "memory.current"))
		pidsValue, _ := readUint(filepath.Join(scope, "pids.current"))
		cpuUsage, _ := readCPUUsage(filepath.Join(scope, "cpu.stat"))
		mainProcess := readMainProcess(collector.options.ProcRoot, filepath.Join(scope, "cgroup.procs"))

		container := Container{
			ID:          shortID(id),
			Name:        shortID(id),
			State:       "running",
			Health:      "running",
			MemoryBytes: memoryBytes,
			PIDs:        int(pidsValue),
			MainProcess: mainProcess,
		}

		if previous, ok := collector.lastContainerCPU[id]; ok && elapsed > 0 && cpuUsage >= previous {
			container.CPUPercent = float64(cpuUsage-previous) / (elapsed * 10_000)
		}
		collector.lastContainerCPU[id] = cpuUsage

		if metadata, ok := collector.dockerContainers[id]; ok {
			container.Name = metadata.Name
			container.Image = metadata.Image
			container.State = metadata.State
			container.Health = metadata.Health
			container.Status = metadata.Status
			if !metadata.Created.IsZero() {
				container.UptimeSeconds = math.Max(0, now.Sub(metadata.Created).Seconds())
			}
		}

		items = append(items, container)
	}

	for id := range collector.lastContainerCPU {
		if _, ok := activeIDs[id]; !ok {
			delete(collector.lastContainerCPU, id)
		}
	}

	sort.Slice(items, func(left, right int) bool {
		leftHealthy := items[left].Health == "healthy" || items[left].Health == "running"
		rightHealthy := items[right].Health == "healthy" || items[right].Health == "running"
		if leftHealthy != rightHealthy {
			return !leftHealthy
		}
		if items[left].MemoryBytes != items[right].MemoryBytes {
			return items[left].MemoryBytes > items[right].MemoryBytes
		}
		return items[left].Name < items[right].Name
	})

	summary := ContainerSummary{
		Available: len(items) > 0,
		Running:   len(items),
		Items:     items,
	}
	for _, item := range items {
		if item.Health == "healthy" || item.Health == "running" {
			summary.Healthy++
		}
	}
	if collector.lastDockerError != nil {
		summary.Error = conciseError(collector.lastDockerError)
	}
	return summary
}

func readCPU(path string) (cpuTimes, error) {
	file, err := os.Open(path)
	if err != nil {
		return cpuTimes{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return cpuTimes{}, errors.New("missing aggregate CPU row")
	}

	fields := strings.Fields(scanner.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuTimes{}, errors.New("invalid aggregate CPU row")
	}

	values := make([]uint64, 0, len(fields)-1)
	for _, field := range fields[1:] {
		value, parseErr := strconv.ParseUint(field, 10, 64)
		if parseErr != nil {
			return cpuTimes{}, parseErr
		}
		values = append(values, value)
	}

	var total uint64
	for _, value := range values {
		total += value
	}

	idle := values[3]
	if len(values) > 4 {
		idle += values[4]
	}
	return cpuTimes{total: total, idle: idle}, scanner.Err()
}

func cpuPercent(previous, current cpuTimes) float64 {
	if previous.total == 0 || current.total <= previous.total {
		return 0
	}

	totalDelta := current.total - previous.total
	idleDelta := uint64(0)
	if current.idle >= previous.idle {
		idleDelta = current.idle - previous.idle
	}
	return clampPercent(float64(totalDelta-idleDelta) / float64(totalDelta) * 100)
}

func readMemory(path string) (Memory, error) {
	file, err := os.Open(path)
	if err != nil {
		return Memory{}, err
	}
	defer file.Close()

	values := make(map[string]uint64)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		value, parseErr := strconv.ParseUint(fields[1], 10, 64)
		if parseErr != nil {
			continue
		}
		values[key] = value * 1024
	}
	if err := scanner.Err(); err != nil {
		return Memory{}, err
	}

	total := values["MemTotal"]
	available := values["MemAvailable"]
	if total == 0 {
		return Memory{}, errors.New("MemTotal is missing")
	}
	if available > total {
		available = total
	}

	swapTotal := values["SwapTotal"]
	swapFree := values["SwapFree"]
	if swapFree > swapTotal {
		swapFree = swapTotal
	}

	return Memory{
		TotalBytes:     total,
		UsedBytes:      total - available,
		AvailableBytes: available,
		UsagePercent:   percent(total-available, total),
		SwapTotalBytes: swapTotal,
		SwapUsedBytes:  swapTotal - swapFree,
		SwapPercent:    percent(swapTotal-swapFree, swapTotal),
	}, nil
}

func readLoad(path string) (float64, float64, float64, error) {
	value, err := readTrimmed(path)
	if err != nil {
		return 0, 0, 0, err
	}
	fields := strings.Fields(value)
	if len(fields) < 3 {
		return 0, 0, 0, errors.New("invalid loadavg")
	}

	load1, err1 := strconv.ParseFloat(fields[0], 64)
	load5, err5 := strconv.ParseFloat(fields[1], 64)
	load15, err15 := strconv.ParseFloat(fields[2], 64)
	return load1, load5, load15, errors.Join(err1, err5, err15)
}

func readUptime(path string) (float64, error) {
	value, err := readTrimmed(path)
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return 0, errors.New("invalid uptime")
	}
	return strconv.ParseFloat(fields[0], 64)
}

func readLogicalCores(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "processor") {
			count++
		}
	}
	return count, scanner.Err()
}

func readPressure(path string) (Pressure, error) {
	file, err := os.Open(path)
	if err != nil {
		return Pressure{}, err
	}
	defer file.Close()

	var pressure Pressure
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		var target *float64
		switch fields[0] {
		case "some":
			target = &pressure.SomeAvg10
		case "full":
			target = &pressure.FullAvg10
		default:
			continue
		}
		for _, field := range fields[1:] {
			if !strings.HasPrefix(field, "avg10=") {
				continue
			}
			value, parseErr := strconv.ParseFloat(strings.TrimPrefix(field, "avg10="), 64)
			if parseErr == nil {
				*target = value
			}
		}
	}
	return pressure, scanner.Err()
}

func readNetwork(path, targetInterface string) (networkTotals, error) {
	file, err := os.Open(path)
	if err != nil {
		return networkTotals{}, err
	}
	defer file.Close()

	var totals networkTotals
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		iface := strings.TrimSpace(parts[0])
		if iface == "lo" || (targetInterface != "" && iface != targetInterface) {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) < 9 {
			continue
		}
		received, receivedErr := strconv.ParseUint(fields[0], 10, 64)
		sent, sentErr := strconv.ParseUint(fields[8], 10, 64)
		if receivedErr == nil {
			totals.received += received
		}
		if sentErr == nil {
			totals.sent += sent
		}
	}
	return totals, scanner.Err()
}

func readDefaultInterface(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] != "Iface" && fields[1] == "00000000" {
			return fields[0], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", errors.New("default network interface not found")
}

func readCPUUsage(path string) (uint64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && fields[0] == "usage_usec" {
			return strconv.ParseUint(fields[1], 10, 64)
		}
	}
	return 0, scanner.Err()
}

func readMainProcess(procRoot, cgroupProcsPath string) string {
	value, err := readTrimmed(cgroupProcsPath)
	if err != nil {
		return ""
	}
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	process, _ := readTrimmed(filepath.Join(procRoot, fields[0], "comm"))
	return process
}

func readUint(path string) (uint64, error) {
	value, err := readTrimmed(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.Fields(value)[0], 10, 64)
}

func readTrimmed(path string) (string, error) {
	value, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(value)), nil
}

func percent(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return clampPercent(float64(used) / float64(total) * 100)
}

func rate(previous, current uint64, elapsed float64) float64 {
	if elapsed <= 0 || current < previous {
		return 0
	}
	return float64(current-previous) / elapsed
}

func clampPercent(value float64) float64 {
	return math.Max(0, math.Min(100, value))
}

func shortID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}

func conciseError(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	if len(message) <= 160 {
		return message
	}
	return message[:157] + "..."
}
