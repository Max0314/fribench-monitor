import { useCallback, useEffect, useRef, useState } from "react";
import type {
  ConnectionState,
  HistoryPoint,
  HistoryResponse,
  RangeOption,
  Snapshot,
  SnapshotResponse,
} from "./types";

export const RANGE_OPTIONS: RangeOption[] = [
  { key: "15m", label: "15分钟", seconds: 15 * 60 },
  { key: "1h", label: "1小时", seconds: 60 * 60 },
  { key: "6h", label: "6小时", seconds: 6 * 60 * 60 },
  { key: "24h", label: "24小时", seconds: 24 * 60 * 60 },
];

interface MetricsState {
  snapshot?: Snapshot;
  points: HistoryPoint[];
  sampleIntervalSeconds: number;
  retentionSeconds: number;
  connection: ConnectionState;
  refresh: () => Promise<void>;
}

export function useMetrics(rangeSeconds: number): MetricsState {
  const [snapshot, setSnapshot] = useState<Snapshot>();
  const [points, setPoints] = useState<HistoryPoint[]>([]);
  const [sampleIntervalSeconds, setSampleIntervalSeconds] = useState(5);
  const [retentionSeconds, setRetentionSeconds] = useState(24 * 3600);
  const [connection, setConnection] = useState<ConnectionState>({
    status: "connecting",
  });
  const timerRef = useRef<number | undefined>(undefined);
  const mountedRef = useRef(true);

  const loadSnapshot = useCallback(async () => {
    try {
      const response = await fetch("/api/snapshot", {
        cache: "no-store",
        headers: { Accept: "application/json" },
      });
      if (!response.ok) {
        if(response.status===401)window.dispatchEvent(new Event('fribench:monitor-denied'));
        throw new Error(`snapshot ${response.status}`);
      }
      const payload = (await response.json()) as SnapshotResponse;
      if (!mountedRef.current) return;

      setSnapshot(payload.snapshot);
      setSampleIntervalSeconds(payload.sampleIntervalSeconds);
      setRetentionSeconds(payload.retentionSeconds);
      setConnection({
        status: "live",
        lastSuccessfulAt: new Date(payload.snapshot.timestamp),
      });
      setPoints((current) => appendPoint(current, payload.snapshot, rangeSeconds));
    } catch (error) {
      if (!mountedRef.current) return;
      setConnection((current) => ({
        status: current.lastSuccessfulAt ? "stale" : "offline",
        lastSuccessfulAt: current.lastSuccessfulAt,
        error: error instanceof Error ? error.message : "unknown error",
      }));
    }
  }, [rangeSeconds]);

  const scheduleNext = useCallback(() => {
    window.clearTimeout(timerRef.current);
    if (document.hidden) return;
    timerRef.current = window.setTimeout(async () => {
      await loadSnapshot();
      scheduleNext();
    }, Math.max(1000, sampleIntervalSeconds * 1000));
  }, [loadSnapshot, sampleIntervalSeconds]);

  useEffect(() => {
    mountedRef.current = true;
    void loadSnapshot().then(scheduleNext);

    const handleVisibility = () => {
      if (document.hidden) {
        window.clearTimeout(timerRef.current);
        return;
      }
      void loadSnapshot().then(scheduleNext);
    };
    document.addEventListener("visibilitychange", handleVisibility);

    return () => {
      mountedRef.current = false;
      window.clearTimeout(timerRef.current);
      document.removeEventListener("visibilitychange", handleVisibility);
    };
  }, [loadSnapshot, scheduleNext]);

  useEffect(() => {
    const controller = new AbortController();
    async function loadHistory() {
      try {
        const response = await fetch(`/api/history?range=${rangeSeconds}`, {
          cache: "no-store",
          headers: { Accept: "application/json" },
          signal: controller.signal,
        });
        if (!response.ok) throw new Error(`history ${response.status}`);
        const payload = (await response.json()) as HistoryResponse;
        if (!controller.signal.aborted) setPoints(payload.points);
      } catch (error) {
        if (error instanceof DOMException && error.name === "AbortError") return;
      }
    }
    void loadHistory();
    return () => controller.abort();
  }, [rangeSeconds]);

  return {
    snapshot,
    points,
    sampleIntervalSeconds,
    retentionSeconds,
    connection,
    refresh: loadSnapshot,
  };
}

function appendPoint(
  current: HistoryPoint[],
  snapshot: Snapshot,
  rangeSeconds: number,
): HistoryPoint[] {
  const timestamp = new Date(snapshot.timestamp).getTime();
  const cutoff = timestamp - rangeSeconds * 1000;
  const nextPoint: HistoryPoint = {
    timestamp: snapshot.timestamp,
    cpuPercent: snapshot.cpu.usagePercent,
    memoryPercent: snapshot.memory.usagePercent,
    diskPercent: snapshot.disk.usagePercent,
    swapPercent: snapshot.memory.swapPercent,
    load1: snapshot.cpu.load1,
    networkReceiveBytesPerSecond: snapshot.network.receiveBytesPerSecond,
    networkSendBytesPerSecond: snapshot.network.sendBytesPerSecond,
  };

  const filtered = current.filter((point) => {
    const pointTime = new Date(point.timestamp).getTime();
    return pointTime >= cutoff && pointTime !== timestamp;
  });
  return [...filtered, nextPoint].slice(-720);
}
