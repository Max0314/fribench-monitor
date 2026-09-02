# Fribench Monitor

Fribench Monitor 是面向个人 Linux 云服务器的轻量只读监控看板。它延续
Fribench 门户的浅色视觉语言，同时把主机资源、短期趋势和 Docker 容器状态放在一个
桌面与手机都可读的页面中。

## 当前范围

- CPU 使用率、负载、CPU 压力和逻辑核心数
- 内存、Swap、内存压力
- 根文件系统容量、inode 与 I/O 压力
- 网络累计流量与实时吞吐
- 主机名、内核版本、运行时间和时区
- Docker 容器状态、CPU、内存和运行时长
- 24 小时内存环形历史；重启即清空，不写数据库或采样文件
- 实时、延迟、离线和演示数据状态
- 15 分钟、1 小时、6 小时和 24 小时趋势切换

生产运行时是一个不带 Node.js 的 Go 二进制。前端在镜像构建阶段生成并嵌入二进制，
浏览器不加载第三方图表库，趋势图由 React 和 SVG 绘制。

## 本地开发

要求 Go 1.24+、Node.js 22+。

```bash
npm --prefix frontend install
npm --prefix frontend run dev
```

另开一个终端启动演示 API：

```bash
set MONITOR_DEMO=true
go run ./cmd/fribench-monitor
```

Vite 默认把 `/api` 和 `/healthz` 代理到 `http://127.0.0.1:8080`。

## 本地生产构建

```bash
npm --prefix frontend run build
go build -o bin/fribench-monitor.exe ./cmd/fribench-monitor
```

然后运行：

```bash
set MONITOR_DEMO=true
bin\fribench-monitor.exe
```

访问 `http://127.0.0.1:8080`。

## Docker Compose

```bash
docker compose up -d --build
curl http://127.0.0.1:26001/healthz
```

Compose 默认：

- 只绑定 `127.0.0.1:26001`
- 从 `/etc/fribench/fribench-monitor/password` 只读挂载生产认证密码
- 除 `/healthz` 外，页面和 API 都要求用户 `monitor` 通过 HTTP Basic Authentication
- 每 5 秒采样一次，内存保留 24 小时
- 只读挂载宿主机 `/proc` 与 cgroup
- 使用仓库内空探针目录读取根文件系统容量，不挂载宿主机根目录
- 以非 root 用户运行，丢弃全部 Linux capabilities
- 内存上限 64 MiB、CPU 上限 0.20 核

Docker 容器名称需要读取 Docker API。Compose 将 Docker socket 只读挂载，并通过
`DOCKER_GID` 把运行用户加入宿主机 Docker 组。只读挂载不能把 Docker API 变成真正的
只读接口；任何能访问 socket 的进程理论上都拥有高权限。因此：

1. 看板代码只实现 `GET /containers/json`，不实现任何容器写操作。
2. 服务必须保持只监听回环地址。
3. 公网入口必须经过 Nginx 与认证保护。
4. 若不需要容器名称，可设置 `MONITOR_DOCKER_ENABLED=false` 并移除 socket 挂载。

服务器上的 Docker 组 GID 可通过以下只读命令获取：

```bash
getent group docker
```

启动时设置：

```bash
DOCKER_GID=实际GID docker compose up -d --build
```

Fribench 服务器当前 Docker 组 GID 为 `988`，Compose 将它作为默认值；迁移到其他宿主机
时必须重新核对并通过 `DOCKER_GID` 覆盖。`deploy/fribench-monitor.sudoers` 只授权
`cpl` 对本项目固定 Compose 文件执行构建、启动、状态、日志、重启和停止，不授权任意
Docker 命令。

## 访问保护

应用支持 HTTP Basic Authentication。密码不写入 Compose 或仓库；生产环境把密码
保存为服务器最小权限文件，然后通过只读挂载和以下变量启用：

- `MONITOR_AUTH_USERNAME`
- `FRIBENCH_MONITOR_AUTH_PASSWORD_FILE`：宿主机密码文件，默认
  `/etc/fribench/fribench-monitor/password`
- `FRIBENCH_SECRETS_GID`：容器只读访问密码文件所需的宿主机组 GID

`/healthz` 不需要认证，便于容器健康检查。生产容器通过宿主机
`fribench-secrets` 组只读访问密码文件，并继续只绑定回环端口；公网只允许通过统一
Nginx 精确入口访问。

生产密码需要轮换时，在交互终端运行 `deploy/rotate-password.sh`。脚本要求输入非空密码
并再次确认，全程不回显；它会重启容器、验证未认证 `401` 和认证后 `200`，失败时自动
恢复旧密码。

## 端口规划

- 容器内部：`8080`
- 宿主机回环：`26001`
- 服务器分类：监控和诊断页面
- 公网入口：`21002`，由 Nginx 反代到回环端口并保留应用登录保护

服务器私有访问示例：

```powershell
ssh -N -L 26001:127.0.0.1:26001 fribench
```

隧道建立后访问 `http://127.0.0.1:26001`。

## Fribench 部署状态

状态：`已验证`

部署日期：`2026-07-29`

目录迁移复核：`2026-09-02`

- 当前 release：`987070a`
- 服务器路径：`/srv/fribench/ops/services/fribench-monitor`
- 容器：`fribench-monitor-monitor-1`
- 回环入口：`127.0.0.1:26001`
- 公网策略：仅由 `https://monitor.fribench.cn` 的 Nginx 入口反代，保留 HTTP Basic、限流与 `noindex`；旧公网高位端口已退役
- 运行限制：64 MiB 内存、0.20 CPU、64 PIDs、只读根文件系统、非 root
- 验收：容器 healthy，`/healthz` 通过，公网未认证访问返回 `401`

GitHub 仓库 `Max0314/fribench-monitor` 是源码准源；服务器保留 release 工作树，
正式目录分类由 `Max0314/fribench-ops` 维护。
