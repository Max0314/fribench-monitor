import { useEffect, useMemo, useRef, useState, type FormEvent } from "react";
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
  const [access,setAccess]=useState<{signedIn:boolean;authorized:boolean}|null>(null);
  const [password,setPassword]=useState(''),[message,setMessage]=useState(''),[busy,setBusy]=useState(false);
  const generation=useRef(0);
  useEffect(()=>{
    const refresh=()=>{
      const own=++generation.current;setAccess(null);
      void fetch('/api/account',{cache:'no-store'}).then(async response=>{
        if(!response.ok)throw Error('account_unavailable');
        const value=await response.json();if(own===generation.current)setAccess(value);
      }).catch(()=>{if(own===generation.current)setMessage('暂时无法验证监控访问权限，请稍后刷新。');});
    };
    refresh();window.addEventListener('fribench:account-change',refresh);window.addEventListener('fribench:monitor-denied',refresh);
    return()=>{generation.current++;window.removeEventListener('fribench:account-change',refresh);window.removeEventListener('fribench:monitor-denied',refresh);};
  },[]);
  const link=async(event:FormEvent)=>{
    event.preventDefault();if(busy)return;setBusy(true);setMessage('');
    try {
      const response=await fetch('/api/account/link',{method:'POST',headers:{'content-type':'application/json'},body:JSON.stringify({password})});
      if(!response.ok)throw Error(response.status===429?'尝试过于频繁，请稍后再试。':'无法授权。请核对原管理员口令；监控不能改绑到其他账号。');
      setPassword('');window.dispatchEvent(new Event('fribench:monitor-denied'));
    }catch(error){setMessage(error instanceof Error?error.message:'无法完成授权。');}finally{setBusy(false);}
  };
  if(access?.authorized)return <Dashboard key={generation.current}/>;
  return <main className="monitor-access">
    <p>FRIBENCH MONITOR</p><h1>仅限主账号访问</h1>
    <p>登录不会自动授予服务器监控权限；普通注册账号不能查看指标。</p>
    {!access?<p role="status">{message||'正在识别已有登录态…'}</p>:!access.signedIn?<a href="/_fribench/auth/login?return_to=%2F">登录 Fribench 账号</a>:<form onSubmit={link}>
      <p>首次使用原管理员口令授权当前账号。授权后，同一账号无需再次输入此口令；该权限不授予其他账号。</p>
      <label htmlFor="operator-password">原管理员口令</label><input id="operator-password" type="password" autoComplete="off" maxLength={1024} value={password} onChange={event=>setPassword(event.target.value)}/>
      <button disabled={!password||busy}>{busy?'验证中':'验证并授权主账号'}</button>
    </form>}
    {access&&message?<p role="alert">{message}</p>:null}
  </main>;
}

function Dashboard() {
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
