import { MoreIcon } from "../icons";
import type { ConnectionState, Snapshot } from "../types";
import { formatClock } from "../format";

interface HeaderProps {
  connection: ConnectionState;
  snapshot?: Snapshot;
}

function portalURL(): string {
  const configured = import.meta.env.VITE_PORTAL_URL as string | undefined;
  if (configured) return configured;
  return "https://fribench.cn/";
}

export function Header({ connection, snapshot }: HeaderProps) {
  const statusLabel =
    connection.status === "live"
      ? "实时"
      : connection.status === "connecting"
        ? "连接中"
        : connection.status === "stale"
          ? "数据延迟"
          : "离线";

  return (
    <header className="app-header">
      <div className="brand">
        <span className="brand__wordmark">FRIBENCH</span>
        <span className="brand__section">服务器监控</span>
      </div>
      <div className="header-status">
        <span className={`live-state live-state--${connection.status}`}>
          <span className="live-state__dot" aria-hidden="true" />
          {statusLabel}
        </span>
        {snapshot ? (
          <span className="updated-at">
            更新于 {formatClock(snapshot.timestamp)}
            {snapshot.source === "demo" ? " · 演示数据" : ""}
          </span>
        ) : null}
        <a className="portal-link" href="/_fribench/auth/manage?return_to=%2F" data-fribench-action="manage">账号与安全</a>
        <a className="portal-link portal-link--desktop" href={portalURL()}>
          返回门户
        </a>
        <a
          className="portal-link portal-link--mobile"
          href={portalURL()}
          aria-label="返回 Fribench 门户"
        >
          <MoreIcon />
        </a>
      </div>
    </header>
  );
}
