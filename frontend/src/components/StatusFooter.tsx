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
      <i aria-hidden="true" />
      <span className="filing-links">
        <a href="https://beian.miit.gov.cn/" target="_blank" rel="noreferrer">
          渝ICP备2026016967号-1
        </a>
        <a
          href="https://beian.mps.gov.cn/#/query/webSearch?code=50010502504754"
          target="_blank"
          rel="noreferrer"
        >
          <img
            src="https://beian.mps.gov.cn/web/assets/logo01.6189a29f.png"
            alt=""
            width="16"
            height="17"
          />
          渝公网安备 50010502504754号
        </a>
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
