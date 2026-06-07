# Codex Skills 使用说明

本项目在 `skills/` 下维护项目级 Codex skills，用于让后续 AI 会话快速恢复项目上下文、明确当前进度、遵守架构边界和提交流程。

## 目录组织

```text
skills/
  ops-release-platform-dev/
  ops-release-platform-todo/
  ops-release-platform-architecture/
  ops-release-platform-deployment/
```

每个功能一个独立 skill 目录，避免把开发流程、TODO、架构说明混在一个文件里。

## Skill 职责

### ops-release-platform-dev

开发总入口。适用于：

- 继续开发前恢复项目上下文。
- 查看本地启动、构建、测试命令。
- 确认本地研发运行拓扑。
- 提交 Git 前执行验证和敏感信息检查。

重点规则：

- 前端和后端在本地启动。
- PostgreSQL 和 Redis 使用 `.secrets/` 中记录的远程研发服务连接配置。
- 不允许把 `.secrets/`、远程主机、连接串、密码、SSH 信息提交到 GitHub。

### ops-release-platform-todo

任务与进度管理。适用于：

- 用户说“继续开发”时选择下一项工作。
- 更新已完成任务、当前未提交工作、下一步建议。
- 对齐 `docs/development-plan.md` 和 `docs/codex-implementation-tasks.md`。

维护位置：

- `skills/ops-release-platform-todo/references/todo.md`

### ops-release-platform-architecture

架构与边界说明。适用于：

- 调整后端分层、前端组织、Redis 任务流、GORM 模型或 docker-compose。
- 设计 Jenkins、Harbor、Kubernetes 等第三方系统 adapter。
- 判断是否需要新增 `internal/service` 或调整模块边界。

维护位置：

- `skills/ops-release-platform-architecture/references/architecture.md`

### ops-release-platform-deployment

部署与本地运行规则。适用于：

- 判断前端、后端、PostgreSQL、Redis 应该如何启动。
- 修改或验证 docker-compose 使用方式。
- 记录研发阶段运行拓扑。

重点规则：

- 研发阶段前端必须通过 npm 命令启动，例如 `npm run dev`。
- 研发阶段后端必须通过 Go 命令启动，例如 `go run ./cmd/server`。
- 研发阶段不要通过 docker-compose 启动前端或后端。
- PostgreSQL 和 Redis 使用 `.secrets/` 中记录的远程研发服务连接配置。
- docker-compose 只用于明确的基础设施、部署或语法校验场景。

维护位置：

- `skills/ops-release-platform-deployment/references/deployment.md`

## 维护规则

1. 重大功能完成或提交后，更新 `ops-release-platform-todo/references/todo.md`。
2. 架构边界发生变化后，更新 `ops-release-platform-architecture/references/architecture.md`。
3. 本地启动、验证、提交、安全规则变化后，更新 `ops-release-platform-dev/references/workflows.md`。
4. 部署或本地运行拓扑变化后，更新 `ops-release-platform-deployment/references/deployment.md`。
5. 不在任何 skill 或 docs 中记录真实服务器连接信息、密码、token、SSH 端口或数据库连接串。
6. 真实连接配置只保存在 `.secrets/`，该目录不得提交。

## 校验

修改 skill 后运行：

```powershell
$env:PYTHONUTF8='1'
python "$env:USERPROFILE\.codex\skills\.system\skill-creator\scripts\quick_validate.py" skills\ops-release-platform-dev
python "$env:USERPROFILE\.codex\skills\.system\skill-creator\scripts\quick_validate.py" skills\ops-release-platform-todo
python "$env:USERPROFILE\.codex\skills\.system\skill-creator\scripts\quick_validate.py" skills\ops-release-platform-architecture
python "$env:USERPROFILE\.codex\skills\.system\skill-creator\scripts\quick_validate.py" skills\ops-release-platform-deployment
```

Windows 下设置 `PYTHONUTF8=1` 是为了避免 Python 默认 GBK 读取 UTF-8 中文时失败。
