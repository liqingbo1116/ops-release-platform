# Git 提交与推送流程

本项目后续所有 Git 提交和推送按此流程处理。

## 适用范围

- 用户要求“提交”“提交 github”“推送”“发到远端”时使用。
- 每次只提交当前任务相关改动，不混入无关文件。
- 不回滚或覆盖用户未明确要求处理的改动。

## 标准流程

1. 检查仓库状态

```bash
git status --short --branch
git branch --show-current
git remote -v
```

确认当前分支、远端和待提交文件范围。若没有 remote，先说明只能本地提交，需要用户提供 GitHub 仓库地址后才能推送。

2. 按改动范围执行验证

- 前端改动：在 `frontend` 执行 `npm run build`
- 后端改动：在 `backend` 执行 `go test ./...`
- Docker/compose 改动：若本机存在 Docker，执行 `docker compose config` 或按任务要求启动验证
- 纯文档改动：可不跑构建，但需要说明未执行代码验证的原因

验证失败时不提交，先修复或向用户说明阻塞原因。

3. 清理本地产物

提交前清理本次验证产生的临时文件，例如：

```bash
Remove-Item -Recurse -Force frontend\dist -ErrorAction SilentlyContinue
Remove-Item -Force *dev*.log -ErrorAction SilentlyContinue
```

不要删除用户文件，不要执行 `git reset --hard`、`git checkout --` 等破坏性命令，除非用户明确要求。

4. 暂存相关改动

优先暂存本次任务涉及的明确路径，例如：

```bash
git add frontend/src frontend/tsconfig.json
```

暂存后检查：

```bash
git status --short
```

5. 创建提交

提交信息使用简洁中文，描述完成的任务，例如：

```bash
git commit -m "前端静态页面工程化"
```

6. 推送当前分支

```bash
git push origin <当前分支>
```

推送失败时保留本地提交，报告失败原因，不强推，除非用户明确要求。

7. 最终确认

推送后执行：

```bash
git status --short --branch
```

最终回复包含：

- 提交 hash 和提交信息
- 推送目标分支
- 执行过的验证命令和结果
- 若存在构建警告，简要说明是否影响提交
- 当前工作区是否干净
