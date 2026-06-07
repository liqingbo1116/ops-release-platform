---
name: ops-release-platform-architecture
description: Use for architecture decisions, module boundary reviews, adapter design, data flow analysis, integration planning, or structural refactors in the ops-release-platform repository. Applies when changing backend layers, frontend page/API organization, Redis task flow, PostgreSQL/GORM models, docker-compose topology, or external integration boundaries.
---

# Ops Release Platform Architecture

Use this skill before changing project structure, module boundaries, or third-party integration design.

## Workflow

1. Check repository state:
   - `git status --short --branch`
   - `git log -1 --oneline`
2. Read `references/architecture.md`.
3. For implementation details, read only the relevant source files.
4. Keep changes aligned with:
   - `docs/PRD.md`
   - `docs/domain-model.md`
   - `docs/api-contract.md`
   - `docs/integration-boundary.md`
   - `docs/non-functional-requirements.md`
5. Prefer small, explicit boundaries over broad refactors.

## Rules

- Handlers should stay thin as orchestration grows; introduce `internal/service` when business workflow becomes non-trivial.
- Third-party calls must go through `backend/internal/integration` interfaces.
- Mock adapters are the default; real adapters require explicit user instruction.
- Redis Stream remains the task queue boundary for mock Agent behavior.
- PostgreSQL/GORM models belong in `backend/internal/repository`.
- Do not put real infrastructure credentials in source, docs, or skills.
- For TODO selection, use `ops-release-platform-todo`.
- For local run and Git workflow, use `ops-release-platform-dev`.
