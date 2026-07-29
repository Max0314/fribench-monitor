import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type PointerEvent,
} from "react";
import { formatAxisTime, formatBytes, formatPercent } from "../format";
import type { HistoryPoint, RangeKey, RangeOption } from "../types";

interface TrendChartProps {
  points: HistoryPoint[];
  range: RangeOption;
  ranges: RangeOption[];
  memoryTotalBytes: number;
  onRangeChange: (range: RangeKey) => void;
}

const desktopChart = {
  width: 820,
  height: 250,
  left: 42,
  right: 72,
  top: 24,
  bottom: 38,
};

const mobileChart = {
  width: 360,
  height: 286,
  left: 34,
  right: 62,
  top: 22,
  bottom: 38,
};

export function TrendChart({
  points,
  range,
  ranges,
  memoryTotalBytes,
  onRangeChange,
}: TrendChartProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const [selectedIndex, setSelectedIndex] = useState<number | null>(null);
  const [compact, setCompact] = useState(() =>
    window.matchMedia("(max-width: 860px)").matches,
  );
  const chart = compact ? mobileChart : desktopChart;
  const safePoints = points.length > 0 ? points : [emptyPoint()];
  const activeIndex = selectedIndex ?? safePoints.length - 1;
  const activePoint = safePoints[Math.max(0, activeIndex)];

  useEffect(() => {
    const media = window.matchMedia("(max-width: 860px)");
    const sync = () => setCompact(media.matches);
    media.addEventListener("change", sync);
    return () => media.removeEventListener("change", sync);
  }, []);

  const geometry = useMemo(() => {
    const plotWidth = chart.width - chart.left - chart.right;
    const plotHeight = chart.height - chart.top - chart.bottom;
    const coordinates = safePoints.map((point, index) => ({
      x:
        chart.left +
        (safePoints.length <= 1 ? plotWidth : (index / (safePoints.length - 1)) * plotWidth),
      cpuY: chart.top + plotHeight * (1 - clamp(point.cpuPercent) / 100),
      memoryY: chart.top + plotHeight * (1 - clamp(point.memoryPercent) / 100),
    }));
    return {
      coordinates,
      cpuPath: pathFrom(coordinates.map((point) => [point.x, point.cpuY])),
      memoryPath: pathFrom(
        coordinates.map((point) => [point.x, point.memoryY]),
      ),
      plotWidth,
      plotHeight,
    };
  }, [chart, safePoints]);

  const activeCoordinate = geometry.coordinates[Math.max(0, activeIndex)];
  const activeMemoryBytes =
    (memoryTotalBytes * activePoint.memoryPercent) / 100;
  const ticks = [0, 25, 50, 75, 100];
  const timeTicks = timeTickIndexes(safePoints.length, compact ? 4 : 5);

  function selectPoint(event: PointerEvent<SVGSVGElement>) {
    const bounds = svgRef.current?.getBoundingClientRect();
    if (!bounds || safePoints.length <= 1) return;
    const ratio = clamp(
      ((event.clientX - bounds.left) / bounds.width) * chart.width -
        chart.left,
      0,
      geometry.plotWidth,
    );
    const index = Math.round((ratio / geometry.plotWidth) * (safePoints.length - 1));
    setSelectedIndex(index);
  }

  return (
    <section className="trend-panel" aria-labelledby="trend-title">
      <div className="panel-heading trend-panel__heading">
        <div>
          <h2 id="trend-title">资源趋势</h2>
          <p>
            <span className="series-key series-key--cpu">CPU</span>
            <span className="series-key series-key--memory">内存</span>
          </p>
        </div>
        <div className="range-control" aria-label="趋势时间范围">
          {ranges.map((option) => (
            <button
              type="button"
              className={option.key === range.key ? "is-selected" : ""}
              aria-pressed={option.key === range.key}
              onClick={() => onRangeChange(option.key)}
              key={option.key}
            >
              {option.label}
            </button>
          ))}
        </div>
      </div>

      <div className="chart-shell">
        <svg
          ref={svgRef}
          className="trend-chart"
          viewBox={`0 0 ${chart.width} ${chart.height}`}
          role="img"
          aria-labelledby="chart-title chart-description"
          onPointerMove={selectPoint}
          onPointerDown={selectPoint}
          onPointerLeave={() => setSelectedIndex(null)}
        >
          <title id="chart-title">CPU 与内存使用率趋势</title>
          <desc id="chart-description">
            当前 CPU {formatPercent(activePoint.cpuPercent)}，内存{" "}
            {formatBytes(activeMemoryBytes)}。
          </desc>

          <text className="chart-axis-title" x={chart.left} y={13}>
            CPU (%)
          </text>
          <text
            className="chart-axis-title chart-axis-title--right"
            x={chart.width - 4}
            y={13}
          >
            内存 (GiB)
          </text>

          {ticks.map((tick) => {
            const y = chart.top + geometry.plotHeight * (1 - tick / 100);
            return (
              <g key={tick}>
                <line
                  className="chart-grid"
                  x1={chart.left}
                  x2={chart.width - chart.right}
                  y1={y}
                  y2={y}
                />
                <text className="chart-axis" x={chart.left - 10} y={y + 4}>
                  {tick}
                </text>
                <text
                  className="chart-axis chart-axis--right"
                  x={chart.width - 4}
                  y={y + 4}
                >
                  {formatMemoryTick((memoryTotalBytes * tick) / 100)}
                </text>
              </g>
            );
          })}

          {timeTicks.map((index) => {
            const point = geometry.coordinates[index];
            return (
              <text
                className="chart-axis chart-axis--time"
                x={point.x}
                y={chart.height - 9}
                key={`${safePoints[index].timestamp}-${index}`}
              >
                {formatAxisTime(safePoints[index].timestamp, range.seconds)}
              </text>
            );
          })}

          <path className="chart-line chart-line--memory" d={geometry.memoryPath} />
          <path className="chart-line chart-line--cpu" d={geometry.cpuPath} />

          {activeCoordinate ? (
            <g className="chart-focus">
              <line
                x1={activeCoordinate.x}
                x2={activeCoordinate.x}
                y1={chart.top}
                y2={chart.height - chart.bottom}
              />
              <circle
                className="chart-dot chart-dot--memory"
                cx={activeCoordinate.x}
                cy={activeCoordinate.memoryY}
                r="4"
              />
              <circle
                className="chart-dot chart-dot--cpu"
                cx={activeCoordinate.x}
                cy={activeCoordinate.cpuY}
                r="4"
              />
            </g>
          ) : null}

          <text
            className="chart-value chart-value--memory"
            x={chart.width - chart.right + 12}
            y={activeCoordinate?.memoryY ?? 40}
          >
            {formatBytes(activeMemoryBytes)}
          </text>
          <text
            className="chart-value chart-value--cpu"
            x={chart.width - chart.right + 12}
            y={(activeCoordinate?.cpuY ?? 40) + 4}
          >
            {formatPercent(activePoint.cpuPercent)}
          </text>

          <rect
            className="chart-hit-area"
            x={chart.left}
            y={chart.top}
            width={geometry.plotWidth}
            height={geometry.plotHeight}
          />
        </svg>
      </div>
      <div className="chart-readout" aria-live="polite">
        <time>{formatAxisTime(activePoint.timestamp, range.seconds)}</time>
        <span className="chart-readout__cpu">
          CPU {formatPercent(activePoint.cpuPercent)}
        </span>
        <span className="chart-readout__memory">
          内存 {formatBytes(activeMemoryBytes)}
        </span>
      </div>
    </section>
  );
}

function pathFrom(points: [number, number][]): string {
  return points
    .map(([x, y], index) => `${index === 0 ? "M" : "L"}${x.toFixed(2)},${y.toFixed(2)}`)
    .join(" ");
}

function timeTickIndexes(length: number, count: number): number[] {
  if (length <= 1) return [0];
  const indexes = new Set<number>();
  for (let index = 0; index < count; index++) {
    indexes.add(Math.round((index / (count - 1)) * (length - 1)));
  }
  return [...indexes];
}

function emptyPoint(): HistoryPoint {
  return {
    timestamp: new Date().toISOString(),
    cpuPercent: 0,
    memoryPercent: 0,
    diskPercent: 0,
    swapPercent: 0,
    load1: 0,
    networkReceiveBytesPerSecond: 0,
    networkSendBytesPerSecond: 0,
  };
}

function clamp(value: number, min = 0, max = 100): number {
  return Math.min(max, Math.max(min, value));
}

function formatMemoryTick(bytes: number): string {
  if (bytes <= 0) return "0";
  return (bytes / 1024 ** 3).toFixed(1);
}
