# AGENTS.md

GitHub 准源为 `Max0314/fribench-monitor`，本机路径以当前 Git 工作树为准；
`G:\Code\fribench\ops\services\fribench-monitor` 是历史开发位置，不作为跨电脑的硬编码依据。
如当前工作区存在上级 `AGENTS.md`，继续遵守其规则。

## 项目定位

- 这是 Fribench 国内云服务器的轻量只读监控看板。
- 生产运行时只保留一个 Go 进程，不使用数据库，不保存业务数据。
- 前端通过 React/Vite 构建为静态资源，并嵌入 Go 二进制。
- Linux 指标来自只读挂载的 `/proc`、cgroup 和根文件系统探针目录。
- Docker 元数据只允许读取，不得增加启停、重启、删除或执行命令等写操作。

## 安全边界

- 不提交密码、Token、Cookie、私钥、Docker 数据目录、日志或运行时采样。
- 生产入口默认绑定 `127.0.0.1:26001`；公网暴露必须经过 Nginx 和访问保护。
- 如启用 Docker 指标，必须明确记录 Docker socket 的权限风险。
- 不得把宿主机 `/`、`/etc`、`/var/lib/docker` 等宽范围目录挂载进容器。

## 验证

提交前至少运行：

```bash
npm --prefix frontend run verify
go test ./...
go vet ./...
```

有 Docker 环境时额外运行：

```bash
docker compose config
docker compose build
```
