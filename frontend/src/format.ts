const binaryUnits = ["B", "KiB", "MiB", "GiB", "TiB"];

export function formatBytes(value: number, decimals = 1): string {
  if (!Number.isFinite(value) || value <= 0) return "0 B";
  const unit = Math.min(
    Math.floor(Math.log(value) / Math.log(1024)),
    binaryUnits.length - 1,
  );
  const scaled = value / 1024 ** unit;
  const digits = scaled >= 100 || unit === 0 ? 0 : decimals;
  return `${scaled.toFixed(digits)} ${binaryUnits[unit]}`;
}

export function formatRate(value: number): string {
  return `${formatBytes(value)}/s`;
}

export function formatPercent(value: number, maxDecimals = 1): string {
  if (!Number.isFinite(value)) return "—";
  const digits = Math.abs(value) >= 10 ? 0 : maxDecimals;
  return `${value.toFixed(digits)}%`;
}

export function formatUptime(totalSeconds: number, compact = false): string {
  if (!Number.isFinite(totalSeconds) || totalSeconds <= 0) return "刚刚启动";
  const days = Math.floor(totalSeconds / 86400);
  const hours = Math.floor((totalSeconds % 86400) / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);

  if (compact) {
    if (days > 0) return `${days}天 ${hours}小时`;
    if (hours > 0) return `${hours}小时 ${minutes}分钟`;
    return `${Math.max(1, minutes)}分钟`;
  }

  return [days > 0 ? `${days} 天` : "", `${hours} 小时`, `${minutes} 分钟`]
    .filter(Boolean)
    .join(" ");
}

export function formatClock(timestamp: string | Date): string {
  const value = timestamp instanceof Date ? timestamp : new Date(timestamp);
  return new Intl.DateTimeFormat("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  }).format(value);
}

export function formatAxisTime(timestamp: string, rangeSeconds: number): string {
  const date = new Date(timestamp);
  if (rangeSeconds >= 24 * 3600) {
    return new Intl.DateTimeFormat("zh-CN", {
      month: "numeric",
      day: "numeric",
      hour: "2-digit",
      hour12: false,
    }).format(date);
  }
  return new Intl.DateTimeFormat("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(date);
}

export function formatCount(value: number): string {
  return new Intl.NumberFormat("zh-CN", { notation: "compact" }).format(value);
}
