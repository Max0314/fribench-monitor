# 开发入口迁移（2026-10-23）

监控现由私有 https://github.com/Max0314/fribench_ops/tree/main/apps/monitor 开发及独立发布。此仓库保留原代码和历史以供回滚；不要在此继续新增功能。以下原项目约束仅作历史参考，当前任务须进入 ops 读取其规范。

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

<!-- fribench-shared-policy:start -->
## Fribench 公共规范引用

- 此节只适用于 Fribench 接入及部署；保留本文件其他项目约束，其他环境按原项目流程执行。
- 每次任务先定位 ops：优先 FRIBENCH_OPS_ROOT，本机默认 D:/code_CPL/fribench_ops；不得依赖工作树的相对位置。准源为私有 https://github.com/Max0314/fribench_ops 。
- 必须实际读取 ops 最新 GitHub main 的 AGENTS.md、docs/workflow.md、docs/contracts.md，不能只读本链接或依赖会话记忆。优先运行 `python <ops>/scripts/read_shared_policy.py`；也可在核实准源后 `git fetch --no-tags <GitHub remote> main`，记录 FETCH_HEAD 的 SHA，逐个用 `git show <SHA>:<文件>` 读取。不要切换、重置或覆盖 ops 的现有工作树。
- 记录本次规范 SHA；修改公共接口、接入组件及发布前再次检查变化。网络失败须标记未核实最新，不能静默用旧缓存批准公共接口或发布。未发布本地规则必须明确标注。
- 规范按最新主线读取，公共组件依赖按明确版本使用；遇到本项目原规则与公共规则冲突，列出差异和适用环境，不静默覆盖。完整引用流程见 ops 的 docs/shared-policy.md。
- 本阶段 work 各服务与私人主页面保持现状；本轮仅补充规范引用，不改变业务、不部署，也不更新其运行依赖。
<!-- fribench-shared-policy:end -->
