import {
  CpuIcon,
  DiskIcon,
  LoadIcon,
  MemoryIcon,
  UptimeIcon,
} from "../icons";
import { formatBytes, formatPercent, formatUptime } from "../format";
import type { Snapshot } from "../types";

export function MetricRail({ snapshot }: { snapshot: Snapshot }) {
  const metrics = [
    {
      key: "cpu",
      label: "CPU",
      value: formatPercent(snapshot.cpu.usagePercent),
      detail: `${snapshot.cpu.logicalCores} 核`,
      icon: CpuIcon,
      tone: "blue",
    },
    {
      key: "memory",
      label: "内存",
      value: formatBytes(snapshot.memory.usedBytes),
      detail: `/ ${formatBytes(snapshot.memory.totalBytes)}`,
      icon: MemoryIcon,
      tone: "violet",
    },
    {
      key: "disk",
      label: "磁盘",
      value: snapshot.disk.available
        ? formatBytes(snapshot.disk.usedBytes, 0)
        : "—",
      detail: snapshot.disk.available
        ? `/ ${formatBytes(snapshot.disk.totalBytes, 0)}`
        : "暂无数据",
      icon: DiskIcon,
      tone: "blue",
    },
    {
      key: "load",
      label: "负载",
      value: snapshot.cpu.load1.toFixed(2),
      detail: "1 分钟",
      icon: LoadIcon,
      tone: "blue",
    },
    {
      key: "uptime",
      label: "在线",
      value: formatUptime(snapshot.uptimeSeconds, true),
      detail: "",
      icon: UptimeIcon,
      tone: "blue",
    },
  ];

  return (
    <div className="metric-rail-shell">
      <div className="metric-rail" aria-label="资源摘要">
        {metrics.map((metric) => {
          const Icon = metric.icon;
          return (
            <div className="metric" key={metric.key}>
              <Icon className="metric__icon" />
              <span className="metric__copy">
                <span className="metric__label">{metric.label}</span>
                <span className={`metric__value metric__value--${metric.tone}`}>
                  {metric.value}
                  {metric.detail ? (
                    <span className="metric__detail">{metric.detail}</span>
                  ) : null}
                </span>
              </span>
            </div>
          );
        })}
      </div>
      <span className="metric-rail__fade" aria-hidden="true" />
    </div>
  );
}
