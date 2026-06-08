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

## Priority Strategy

- The default priority is V1 feature closure, not optimization work.
- The V1 delivery bar is: the platform must at least support remote project-environment deployment and management.
- Treat the following path as the mainline unless the user explicitly reprioritizes:
  - project environment management
  - Agent registration and status
  - runtime snapshot collection or mock-equivalent collection flow
  - baseline generation and lock
  - baseline-to-target diff
  - release creation for target-existing services
  - deploy task creation for target-missing services
  - remote execution through Agent task flow
  - release/deploy logs, retry states, and audit visibility
- Performance tuning, bundle optimization, warning cleanup, refactor-only cleanup, and UI polish should be scheduled after the mainline unless they block build, test, or feature delivery.
- When the user says `继续` or `继续开发`, select the next unfinished item on this mainline before taking optimization work.

## V1 Ordered Path

1. Runtime snapshot and baseline generation flow.
2. Baseline lock and baseline detail persistence/display.
3. Backend-owned diff classification and action generation.
4. Release management closure:
   - single-service release
   - multi-service release
   - baseline diff release
   - remote Agent execution state updates
5. Deployment management closure for remote project environments:
   - deploy task creation for missing services
   - step orchestration
   - retry / skip / manual confirm support
   - detail logs and result display
6. Agent protocol completion:
   - heartbeat
   - task pull
   - step status report
   - log report
   - final result report
7. Audit and permission completion for the above flows.
8. Product/service management and image management features that are required to operate the above flows.
9. Non-functional optimization work after V1 functional closure.

## Rules

- Do not duplicate full PRD or API contract content here; link to `docs/` instead.
- Keep TODOs short and actionable.
- Separate committed work from local uncommitted work.
- Prefer backend/domain completion over UI-only expansion when the core flow is incomplete.
- Do not move optimization-only work ahead of the remote deploy/manage mainline unless the optimization is blocking delivery.
- Do not store credentials, server addresses, SSH details, or connection strings in TODO files.
- For architectural questions, use `ops-release-platform-architecture`.
- For commit/push workflow, use `ops-release-platform-dev` and `docs/git-submit-workflow.md`.
- If log inspection is needed while checking progress, only read the newest 10 or 20 lines from a log file. Do not load full logs into context.
