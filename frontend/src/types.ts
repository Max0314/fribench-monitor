export interface SnapshotResponse {
  snapshot: Snapshot;
  sampleIntervalSeconds: number;
  retentionSeconds: number;
}

export interface HistoryResponse {
  rangeSeconds: number;
  points: HistoryPoint[];
}

export interface Snapshot {
  timestamp: string;
  source: "live" | "demo";
  hostname: string;
  kernel: string;
  timezone: string;
  uptimeSeconds: number;
  cpu: CPU;
  memory: Memory;
  disk: Disk;
  network: Network;
  containers: ContainerSummary;
}

export interface CPU {
  usagePercent: number;
  logicalCores: number;
  load1: number;
  load5: number;
  load15: number;
  pressure: Pressure;
}

export interface Memory {
  totalBytes: number;
  usedBytes: number;
  availableBytes: number;
  usagePercent: number;
  swapTotalBytes: number;
  swapUsedBytes: number;
  swapPercent: number;
  pressure: Pressure;
}

export interface Disk {
  available: boolean;
  totalBytes: number;
  usedBytes: number;
  freeBytes: number;
  usagePercent: number;
  inodesTotal: number;
  inodesUsed: number;
  inodesPercent: number;
  pressure: Pressure;
}

export interface Network {
  receivedBytes: number;
  sentBytes: number;
  receiveBytesPerSecond: number;
  sendBytesPerSecond: number;
}

export interface Pressure {
  someAvg10: number;
  fullAvg10: number;
}

export interface ContainerSummary {
  available: boolean;
  running: number;
  healthy: number;
  items: ContainerMetric[];
  error?: string;
}

export interface ContainerMetric {
  id: string;
  name: string;
  image?: string;
  state: string;
  health: "healthy" | "unhealthy" | "running" | "unknown";
  status?: string;
  cpuPercent: number;
  memoryBytes: number;
  pids: number;
  uptimeSeconds: number;
  mainProcess?: string;
}

export interface HistoryPoint {
  timestamp: string;
  cpuPercent: number;
  memoryPercent: number;
  diskPercent: number;
  swapPercent: number;
  load1: number;
  networkReceiveBytesPerSecond: number;
  networkSendBytesPerSecond: number;
}

export type RangeKey = "15m" | "1h" | "6h" | "24h";

export interface RangeOption {
  key: RangeKey;
  label: string;
  seconds: number;
}

export interface ConnectionState {
  status: "connecting" | "live" | "stale" | "offline";
  lastSuccessfulAt?: Date;
  error?: string;
}
