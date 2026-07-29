import type { Snapshot } from "../types";
import { formatUptime } from "../format";

export function HostInfo({ snapshot }: { snapshot: Snapshot }) {
  const details = [
    ["主机名", snapshot.hostname || "—"],
    ["操作系统", snapshot.kernel ? `Linux ${snapshot.kernel}` : "Linux"],
    ["CPU", `${snapshot.cpu.logicalCores || "—"} 核`],
    ["时区", snapshot.timezone || "—"],
    ["启动时间", formatUptime(snapshot.uptimeSeconds)],
  ];

  return (
    <section className="host-info" aria-labelledby="host-title">
      <h2 id="host-title">主机信息</h2>
      <dl>
        {details.map(([label, value]) => (
          <div key={label}>
            <dt>{label}</dt>
            <dd>{value}</dd>
          </div>
        ))}
      </dl>
    </section>
  );
}
