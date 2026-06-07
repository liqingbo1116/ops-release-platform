---
name: ops-release-platform-dev
description: Use for continuing development, review, testing, deployment notes, or Git submission work in the ops-release-platform repository. Includes project context, completed milestone state, repo organization, validation commands, security constraints, mock integration boundaries, and required Git workflow for this specific 运维发布交付平台 project.
---

# Ops Release Platform Dev

Use this skill before making code, docs, deployment, or Git changes in this repository.

## First Steps

1. Check the current workspace before assuming state:
   - `git status --short --branch`
   - `git log -1 --oneline`
   - Read task-specific docs under `docs/`.
2. Read references only as needed:
   - Current milestone and known state: `references/project-state.md`
   - Repository layout and ownership boundaries: `references/repo-map.md`
   - Validation, deployment, and Git workflow: `references/workflows.md`
   - TODO/backlog work is split into `../ops-release-platform-todo/`
   - Architecture decisions are split into `../ops-release-platform-architecture/`
3. Prefer existing project patterns over new abstractions.
4. Keep third-party systems behind adapter interfaces. Do not implement real Jenkins, Harbor, Kubernetes, GitLab, ArgoCD, or Nacos integration unless explicitly requested.
5. Never commit server credentials, `.secrets/`, deployment passwords, SSH connection details, or real external-system credentials.

## Development Rules

- Frontend stack: Vue 3, Vite, TypeScript, Pinia, Vue Router, Element Plus.
- Backend stack: Go, Gin, GORM, PostgreSQL, Redis Stream.
- MVP data source: mock JSON or backend mock API unless a task explicitly says to persist real data.
- Agent behavior: use Redis Stream and mock worker for now.
- Integration behavior: use mock adapters only; real adapters must preserve the existing interface contracts.
- Local development runtime: run frontend and backend locally; load remote PostgreSQL and Redis settings from `.secrets/local-dev-env.ps1` before starting the backend.
- For user requests like "继续开发", choose the next clear item from `docs/development-plan.md` and current repository state.
- For user requests like "提交", follow `docs/git-submit-workflow.md` and the extra checks in `references/workflows.md`.

## Update This Skill

After a meaningful milestone or commit, update `references/project-state.md` with:

- completed task or phase
- important files changed
- validation results
- known warnings or blockers
- whether the work was committed and pushed
