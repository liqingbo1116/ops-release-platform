---
name: ops-release-platform-todo
description: Use for maintaining, reviewing, prioritizing, or selecting TODO/backlog work in the ops-release-platform repository. Applies when the user asks what to do next, says continue development, asks to update task progress, split tasks, track completed work, or reconcile docs with implementation state.
---

# Ops Release Platform Todo

Use this skill to keep project progress explicit and avoid losing the current development thread.

## Workflow

1. Check live repository state first:
   - `git status --short --branch`
   - `git log -1 --oneline`
2. Read `references/todo.md`.
3. Compare TODO state with:
   - `docs/development-plan.md`
   - `docs/codex-implementation-tasks.md`
   - recent commits and current uncommitted files
4. When starting work, choose one clear task boundary.
5. When finishing work, update `references/todo.md` with:
   - completed item
   - validation result
   - commit hash if pushed
   - remaining next task

## Rules

- Do not duplicate full PRD or API contract content here; link to `docs/` instead.
- Keep TODOs short and actionable.
- Separate committed work from local uncommitted work.
- Do not store credentials, server addresses, SSH details, or connection strings in TODO files.
- For architectural questions, use `ops-release-platform-architecture`.
- For commit/push workflow, use `ops-release-platform-dev` and `docs/git-submit-workflow.md`.
