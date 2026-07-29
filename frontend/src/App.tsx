import { useMemo, useState } from "react";
import { RANGE_OPTIONS, useMetrics } from "./api";
import { Header } from "./components/Header";
import { MetricRail } from "./components/MetricRail";
import { TrendChart } from "./components/TrendChart";
import { ResourcePanel } from "./components/ResourcePanel";
import { HostInfo } from "./components/HostInfo";
import { ContainerTable } from "./components/ContainerTable";
import { alertCount, StatusFooter } from "./components/StatusFooter";
import type { RangeKey } from "./types";

function initialRange(): RangeKey {
  const candidate = new URLSearchParams(window.location.search).get("range");
  return RANGE_OPTIONS.some((option) => option.key === candidate)
    ? (candidate as RangeKey)
    : "1h";
}

export default function App() {
  const [rangeKey, setRangeKey] = useState<RangeKey>(initialRange);
  const range =
    RANGE_OPTIONS.find((option) => option.key === rangeKey) ?? RANGE_OPTIONS[1];
  const metrics = useMetrics(range.seconds);

  const summary = useMemo(() => {
    if (!metrics.snapshot) return "正在读取服务器指标。";
    const alerts = alertCount(metrics.snapshot);
    if (alerts > 0) return `检测到 ${alerts} 项资源或服务需要关注。`;
    return "运行平稳，资源余量充足。";
  }, [metrics.snapshot]);

  function changeRange(next: RangeKey) {
    setRangeKey(next);
    const url = new URL(window.location.href);
    if (next === "1h") {
      url.searchParams.delete("range");
    } else {
      url.searchParams.set("range", next);
    }
    window.history.replaceState({}, "", url);
  }

  return (
    <div className="app-shell">
      <div className="page-atmosphere" aria-hidden="true" />
      <Header
        connection={metrics.connection}
        snapshot={metrics.snapshot}
      />
      <main>
        <section className="page-intro" aria-labelledby="page-title">
          <div>
            <h1 id="page-title">服务器状态</h1>
            <p>{summary}</p>
          </div>
          {metrics.snapshot ? (
            <time
              className="page-intro__updated"
              dateTime={metrics.snapshot.timestamp}
            >
              更新于{" "}
              {new Date(metrics.snapshot.timestamp).toLocaleTimeString("zh-CN", {
                hour12: false,
              })}
            </time>
          ) : null}
        </section>

        {metrics.snapshot ? (
          <>
            <MetricRail snapshot={metrics.snapshot} />
            <div className="dashboard-grid">
              <TrendChart
                points={metrics.points}
                range={range}
                ranges={RANGE_OPTIONS}
                memoryTotalBytes={metrics.snapshot.memory.totalBytes}
                onRangeChange={changeRange}
              />
              <aside className="support-rail" aria-label="服务器资源与主机信息">
                <ResourcePanel snapshot={metrics.snapshot} />
                <HostInfo snapshot={metrics.snapshot} />
              </aside>
              <ContainerTable containers={metrics.snapshot.containers} />
            </div>
            <div className="host-info-mobile">
              <HostInfo snapshot={metrics.snapshot} />
            </div>
            <StatusFooter
              snapshot={metrics.snapshot}
              sampleIntervalSeconds={metrics.sampleIntervalSeconds}
              retentionSeconds={metrics.retentionSeconds}
            />
          </>
        ) : (
          <LoadingState status={metrics.connection.status} />
        )}
      </main>
    </div>
  );
}

function LoadingState({ status }: { status: string }) {
  return (
    <section className="loading-state" aria-live="polite">
      <span className="loading-state__line loading-state__line--wide" />
      <span className="loading-state__line" />
      <div className="loading-state__panel" />
      <p>{status === "offline" ? "暂时无法连接监控服务" : "正在读取实时指标…"}</p>
    </section>
  );
}
