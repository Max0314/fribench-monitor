package metrics

import "time"

type Snapshot struct {
	Timestamp  time.Time        `json:"timestamp"`
	Source     string           `json:"source"`
	Hostname   string           `json:"hostname"`
	Kernel     string           `json:"kernel"`
	Timezone   string           `json:"timezone"`
	Uptime     float64          `json:"uptimeSeconds"`
	CPU        CPU              `json:"cpu"`
	Memory     Memory           `json:"memory"`
	Disk       Disk             `json:"disk"`
	Network    Network          `json:"network"`
	Containers ContainerSummary `json:"containers"`
}

type CPU struct {
	UsagePercent float64  `json:"usagePercent"`
	LogicalCores int      `json:"logicalCores"`
	Load1        float64  `json:"load1"`
	Load5        float64  `json:"load5"`
	Load15       float64  `json:"load15"`
	Pressure     Pressure `json:"pressure"`
}

type Memory struct {
	TotalBytes     uint64   `json:"totalBytes"`
	UsedBytes      uint64   `json:"usedBytes"`
	AvailableBytes uint64   `json:"availableBytes"`
	UsagePercent   float64  `json:"usagePercent"`
	SwapTotalBytes uint64   `json:"swapTotalBytes"`
	SwapUsedBytes  uint64   `json:"swapUsedBytes"`
	SwapPercent    float64  `json:"swapPercent"`
	Pressure       Pressure `json:"pressure"`
}

type Disk struct {
	Available     bool     `json:"available"`
	TotalBytes    uint64   `json:"totalBytes"`
	UsedBytes     uint64   `json:"usedBytes"`
	FreeBytes     uint64   `json:"freeBytes"`
	UsagePercent  float64  `json:"usagePercent"`
	InodesTotal   uint64   `json:"inodesTotal"`
	InodesUsed    uint64   `json:"inodesUsed"`
	InodesPercent float64  `json:"inodesPercent"`
	Pressure      Pressure `json:"pressure"`
}

type Network struct {
	ReceivedBytes uint64  `json:"receivedBytes"`
	SentBytes     uint64  `json:"sentBytes"`
	ReceiveBPS    float64 `json:"receiveBytesPerSecond"`
	SendBPS       float64 `json:"sendBytesPerSecond"`
}

type Pressure struct {
	SomeAvg10 float64 `json:"someAvg10"`
	FullAvg10 float64 `json:"fullAvg10"`
}

type ContainerSummary struct {
	Available bool        `json:"available"`
	Running   int         `json:"running"`
	Healthy   int         `json:"healthy"`
	Items     []Container `json:"items"`
	Error     string      `json:"error,omitempty"`
}

type Container struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Image         string  `json:"image,omitempty"`
	State         string  `json:"state"`
	Health        string  `json:"health"`
	Status        string  `json:"status,omitempty"`
	CPUPercent    float64 `json:"cpuPercent"`
	MemoryBytes   uint64  `json:"memoryBytes"`
	PIDs          int     `json:"pids"`
	UptimeSeconds float64 `json:"uptimeSeconds"`
	MainProcess   string  `json:"mainProcess,omitempty"`
}

type Point struct {
	Timestamp         time.Time `json:"timestamp"`
	CPUPercent        float64   `json:"cpuPercent"`
	MemoryPercent     float64   `json:"memoryPercent"`
	DiskPercent       float64   `json:"diskPercent"`
	SwapPercent       float64   `json:"swapPercent"`
	Load1             float64   `json:"load1"`
	NetworkReceiveBPS float64   `json:"networkReceiveBytesPerSecond"`
	NetworkSendBPS    float64   `json:"networkSendBytesPerSecond"`
}

func PointFromSnapshot(snapshot Snapshot) Point {
	return Point{
		Timestamp:         snapshot.Timestamp,
		CPUPercent:        snapshot.CPU.UsagePercent,
		MemoryPercent:     snapshot.Memory.UsagePercent,
		DiskPercent:       snapshot.Disk.UsagePercent,
		SwapPercent:       snapshot.Memory.SwapPercent,
		Load1:             snapshot.CPU.Load1,
		NetworkReceiveBPS: snapshot.Network.ReceiveBPS,
		NetworkSendBPS:    snapshot.Network.SendBPS,
	}
}
