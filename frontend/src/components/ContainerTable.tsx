import {
  Fragment,
  useMemo,
  useState,
  type KeyboardEvent,
} from "react";
import { ChevronIcon, ContainerIcon } from "../icons";
import { formatBytes, formatPercent, formatUptime } from "../format";
import type { ContainerMetric, ContainerSummary } from "../types";

type SortKey = "name" | "health" | "cpu" | "memory" | "uptime";

export function ContainerTable({ containers }: { containers: ContainerSummary }) {
  const [showAll, setShowAll] = useState(false);
  const [sortKey, setSortKey] = useState<SortKey>("memory");
  const [ascending, setAscending] = useState(false);
  const [expandedID, setExpandedID] = useState<string>();

  const sorted = useMemo(() => {
    const result = [...containers.items];
    result.sort((left, right) => {
      const comparison = compare(left, right, sortKey);
      return ascending ? comparison : -comparison;
    });
    return result;
  }, [containers.items, sortKey, ascending]);

  const visible = showAll ? sorted : sorted.slice(0, 7);

  function changeSort(next: SortKey) {
    if (next === sortKey) {
      setAscending((value) => !value);
      return;
    }
    setSortKey(next);
    setAscending(next === "name" || next === "health");
  }

  function toggleRow(container: ContainerMetric) {
    setExpandedID((current) => (current === container.id ? undefined : container.id));
  }

  function handleRowKey(
    event: KeyboardEvent<HTMLTableRowElement>,
    container: ContainerMetric,
  ) {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      toggleRow(container);
    }
  }

  return (
    <section className="container-panel" aria-labelledby="containers-title">
      <div className="container-panel__heading">
        <h2 id="containers-title">
          运行中的容器 <span>{containers.running}</span>
        </h2>
        {containers.available && sorted.length > 7 ? (
          <button
            className="container-heading-action"
            type="button"
            onClick={() => setShowAll((value) => !value)}
          >
            {showAll ? "收起容器列表" : `查看全部 ${sorted.length} 个`}
            <ChevronIcon className={showAll ? "is-expanded" : ""} />
          </button>
        ) : !containers.available && containers.error ? (
          <span className="container-panel__error">Docker 明细暂不可用</span>
        ) : null}
      </div>

      {containers.available && sorted.length > 0 ? (
        <>
          <div className="container-table-shell">
            <table className="container-table">
              <thead>
                <tr>
                  <SortableHeader
                    label="服务"
                    sort="name"
                    active={sortKey}
                    ascending={ascending}
                    onChange={changeSort}
                  />
                  <SortableHeader
                    label="状态"
                    sort="health"
                    active={sortKey}
                    ascending={ascending}
                    onChange={changeSort}
                  />
                  <SortableHeader
                    label="CPU"
                    sort="cpu"
                    active={sortKey}
                    ascending={ascending}
                    onChange={changeSort}
                  />
                  <SortableHeader
                    label="内存"
                    sort="memory"
                    active={sortKey}
                    ascending={ascending}
                    onChange={changeSort}
                  />
                  <SortableHeader
                    label="运行时间"
                    sort="uptime"
                    active={sortKey}
                    ascending={ascending}
                    onChange={changeSort}
                  />
                  <th aria-label="展开详情" />
                </tr>
              </thead>
              <tbody className={showAll ? "is-expanded" : ""}>
                {visible.map((container, index) => {
                  const expanded = expandedID === container.id;
                  const mobileHidden = !showAll && index >= 4;
                  return (
                    <Fragment key={container.id}>
                      <tr
                        className={
                          mobileHidden
                            ? "container-row container-row--mobile-hidden"
                            : "container-row"
                        }
                        tabIndex={0}
                        aria-expanded={expanded}
                        onClick={() => toggleRow(container)}
                        onKeyDown={(event) => handleRowKey(event, container)}
                      >
                        <td>
                          <span className="service-name">
                            <ContainerIcon />
                            <span>{container.name}</span>
                          </span>
                        </td>
                        <td>
                          <span className={`health health--${container.health}`}>
                            <span aria-hidden="true" />
                            {healthLabel(container)}
                          </span>
                        </td>
                        <td>{formatPercent(container.cpuPercent, 2)}</td>
                        <td>{formatBytes(container.memoryBytes)}</td>
                        <td>{formatUptime(container.uptimeSeconds, true)}</td>
                        <td>
                          <ChevronIcon
                            className={expanded ? "chevron is-expanded" : "chevron"}
                          />
                        </td>
                      </tr>
                      {expanded ? (
                        <tr
                          className={
                            mobileHidden
                              ? "container-detail container-row--mobile-hidden"
                              : "container-detail"
                          }
                        >
                          <td colSpan={6}>
                            <span>
                              镜像 <strong>{container.image || "未读取"}</strong>
                            </span>
                            <span>
                              主进程 <strong>{container.mainProcess || "—"}</strong>
                            </span>
                            <span>
                              进程数 <strong>{container.pids}</strong>
                            </span>
                            <span className="mobile-uptime">
                              运行时间{" "}
                              <strong>{formatUptime(container.uptimeSeconds)}</strong>
                            </span>
                          </td>
                        </tr>
                      ) : null}
                    </Fragment>
                  );
                })}
              </tbody>
            </table>
          </div>
          {sorted.length > 7 ? (
            <button
              className="show-all show-all--mobile"
              type="button"
              onClick={() => setShowAll((value) => !value)}
            >
              {showAll ? "收起容器列表" : `查看全部 ${sorted.length} 个`}
              <ChevronIcon className={showAll ? "is-expanded" : ""} />
            </button>
          ) : null}
        </>
      ) : (
        <div className="container-empty">
          <ContainerIcon />
          <div>
            <strong>容器明细暂不可用</strong>
            <p>
              {containers.error ||
                "未启用 Docker 指标；主机资源采集不受影响。"}
            </p>
          </div>
        </div>
      )}
    </section>
  );
}

function SortableHeader({
  label,
  sort,
  active,
  ascending,
  onChange,
}: {
  label: string;
  sort: SortKey;
  active: SortKey;
  ascending: boolean;
  onChange: (sort: SortKey) => void;
}) {
  return (
    <th aria-sort={active === sort ? (ascending ? "ascending" : "descending") : "none"}>
      <button type="button" onClick={() => onChange(sort)}>
        {label}
        <span
          className={
            active === sort
              ? ascending
                ? "sort-icon is-active is-ascending"
                : "sort-icon is-active"
              : "sort-icon"
          }
          aria-hidden="true"
        />
      </button>
    </th>
  );
}

function compare(
  left: ContainerMetric,
  right: ContainerMetric,
  key: SortKey,
): number {
  switch (key) {
    case "name":
      return left.name.localeCompare(right.name, "zh-CN");
    case "health":
      return left.health.localeCompare(right.health);
    case "cpu":
      return left.cpuPercent - right.cpuPercent;
    case "memory":
      return left.memoryBytes - right.memoryBytes;
    case "uptime":
      return left.uptimeSeconds - right.uptimeSeconds;
  }
}

function healthLabel(container: ContainerMetric): string {
  if (container.health === "unhealthy") return "异常";
  if (container.state !== "running") return "已停止";
  return "运行中";
}
