import type { Snapshot } from "../types";

export function StatusFooter({
  snapshot,
  sampleIntervalSeconds,
  retentionSeconds,
}: {
  snapshot: Snapshot;
  sampleIntervalSeconds: number;
  retentionSeconds: number;
}) {
  const alerts = alertCount(snapshot);
  return (
    <footer className="status-footer">
      <span>数据保留 {Math.round(retentionSeconds / 3600)} 小时</span>
      <i aria-hidden="true" />
      <span>每 {sampleIntervalSeconds} 秒刷新</span>
      <i aria-hidden="true" />
      <span className={alerts > 0 ? "has-alerts" : "no-alerts"}>
        <b aria-hidden="true" />
        {alerts > 0 ? `${alerts} 项需关注` : "无告警"}
      </span>
    </footer>
  );
}

export function alertCount(snapshot: Snapshot): number {
  let alerts = 0;
  if (snapshot.cpu.usagePercent >= 85) alerts++;
  if (snapshot.memory.usagePercent >= 85) alerts++;
  if (snapshot.disk.available && snapshot.disk.usagePercent >= 85) alerts++;
  if (snapshot.disk.available && snapshot.disk.inodesPercent >= 85) alerts++;
  alerts += snapshot.containers.items.filter(
    (container) =>
      container.health === "unhealthy" || container.state !== "running",
  ).length;
  return alerts;
}
