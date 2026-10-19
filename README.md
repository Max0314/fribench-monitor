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
- `/healthz` 公开；启用统一账号后公开登录外壳与静态资源，指标 API 仍要求所有者授权或原 HTTP Basic 凭据
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

统一账号接入状态：已验证（2026-10-16），业务提交 `70339131611cab4718fabc3756ea1cbdd67ffbdb` 已推送合并并部署，Go 测试/vet、前端构建与审计、公网匿名权限边界通过。新版本公开的仅为登录外壳与静态资源，指标和历史 API 仍受服务端授权。普通 Fribench 注册账号没有监控权限。主账号首次登录后，需要验证原监控管理员口令；公共底座在 PostgreSQL 中只登记一个监控所有者，不能由另一个账号再次占用（本次未代替用户领取）。后续同账号登录态自动识别，不再反复输入原口令；原 HTTP Basic 凭据作为既有运维访问方式保留。

监控进程本身仍不使用数据库、不写采样文件、不实现 Docker 写操作。新增的内部身份与权限查询通过受保护的应用凭据完成，配置只引用 `/etc/fribench/apps/fribench-monitor/app.env`。密钥、内部账号标识和原管理员口令均不进入前端构建、日志或 Git。中央会话短缓存最多五秒；账号变化时清除界面中的指标并重新验证权限。

发布固定 `MONITOR_VERSION` 为已推送提交，先运行前端 verify、Go 测试与 vet，再构建镜像。只更新 `fribench-monitor`，保持 26001 回环端口、64 MiB 上限、原只读挂载与 Docker socket 风险边界不变。主账号登记保存在公共底座，应用容器重启不会丢失；代码恢复需要重建指定提交，不删除授权记录。

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
- 公网入口：`https://monitor.fribench.cn` 的标准 HTTPS `443`，由 Nginx 反代到回环端口并保留服务端授权
- 旧公网高位端口 `21002` 已停用；不能与现行相册的回环 `21002` 混淆

服务器私有访问示例：

```powershell
ssh -N -L 26001:127.0.0.1:26001 fribench
```

隧道建立后访问 `http://127.0.0.1:26001`。

## Fribench 部署状态

状态：`已验证`

最后核对：`2026-10-19`。本次仅修正文档，未重建监控容器。

- 当前运行镜像提交：`d8af8c2e145e4e04da00de78b5895c218b330d4c`
- 运行版本准源：服务器 `/var/lib/fribench/releases/fribench-monitor.json`，不是 README 的文档提交
- 服务器发布目录：`/srv/fribench/ops/services/fribench-monitor/releases/<完整提交>`
- Compose 项目：`fribench-monitor`；服务：`monitor`；实际容器名用 `docker compose ps` 查询
- 回环入口：`127.0.0.1:26001`
- 公网策略：仅由 HTTPS Nginx 入口反代；登录外壳可公开，真实指标须账号所有者授权或原 HTTP Basic，保留限流与 `noindex`
- 运行限制：64 MiB 内存、0.20 CPU、64 PIDs、只读根文件系统、非 root
- 验收：容器 healthy，`/healthz` 通过，未认证指标 API 拒绝访问；不能再把“首页必须 401”当作统一账号版本的验收标准

GitHub 仓库 `Max0314/fribench-monitor` 是源码准源；服务器保留 release 工作树，
正式目录分类和发布工具由 `Max0314/fribench_ops` 维护。新公共服务运维入口为
`/srv/fribench/ops-unified-20261015`，旧 `/srv/fribench/ops` 仍承载共享平台/work，不能整树覆盖。
