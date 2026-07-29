import { DiskIcon, InodeIcon, NetworkIcon, SwapIcon } from "../icons";
import { formatBytes, formatCount, formatPercent, formatRate } from "../format";
import type { Snapshot } from "../types";

export function ResourcePanel({ snapshot }: { snapshot: Snapshot }) {
  const resources = [
    {
      key: "disk",
      label: "磁盘使用",
      percent: snapshot.disk.usagePercent,
      detail: snapshot.disk.available
        ? `${formatBytes(snapshot.disk.usedBytes, 0)} / ${formatBytes(snapshot.disk.totalBytes, 0)}`
        : "暂无数据",
      icon: DiskIcon,
    },
    {
      key: "inode",
      label: "inode 使用",
      percent: snapshot.disk.inodesPercent,
      detail:
        snapshot.disk.available && snapshot.disk.inodesTotal > 0
          ? `${formatCount(snapshot.disk.inodesUsed)} / ${formatCount(snapshot.disk.inodesTotal)}`
          : "暂无数据",
      icon: InodeIcon,
    },
    {
      key: "swap",
      label: "交换空间",
      percent: snapshot.memory.swapPercent,
      detail: `${formatBytes(snapshot.memory.swapUsedBytes)} / ${formatBytes(snapshot.memory.swapTotalBytes)}`,
      icon: SwapIcon,
    },
  ];

  return (
    <section className="support-panel resource-panel" aria-labelledby="resource-title">
      <div className="panel-heading">
        <div>
          <h2 id="resource-title">
            <span className="resource-title--desktop">磁盘与交换空间</span>
            <span className="resource-title--mobile">当前资源</span>
          </h2>
        </div>
      </div>

      <div className="resource-list">
        {resources.map((resource) => {
          const Icon = resource.icon;
          return (
            <div className="resource-row" key={resource.key}>
              <Icon className="resource-row__icon" />
              <div className="resource-row__content">
                <div className="resource-row__title">
                  <span>{resource.label}</span>
                  <strong>{formatPercent(resource.percent)}</strong>
                </div>
                <div className="progress" aria-hidden="true">
                  <span
                    style={{ width: `${Math.min(100, Math.max(0, resource.percent))}%` }}
                  />
                </div>
                <span className="resource-row__detail">{resource.detail}</span>
              </div>
            </div>
          );
        })}
      </div>

      <div className="network-summary">
        <NetworkIcon />
        <span>
          <small>网络吞吐</small>
          <strong>
            ↓ {formatRate(snapshot.network.receiveBytesPerSecond)}
            <span>↑ {formatRate(snapshot.network.sendBytesPerSecond)}</span>
          </strong>
        </span>
      </div>
    </section>
  );
}
