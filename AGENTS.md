# AGENTS

## Scope

This file defines repository-level collaboration rules for Codex and other coding agents.

## Context Control

- Do not read or include dependency directories, build outputs, caches, or generated files unless explicitly required.
- Do not read `test/` or `tests/` data files. Treat them as large, low-signal inputs unless the user explicitly asks for them.
- Do not paste large command outputs, long logs, or full generated files into the conversation. Summarize instead.
- Prefer showing code changes as `git diff` style patches, not full file dumps.

## Logging

- Use the project's predefined `logger` instance for log output.
- Do not introduce `print` or equivalent ad hoc console debugging in committed code.
- Keep runtime logging concise. Avoid dumping full payloads, stack traces, or large collections unless the task explicitly requires it.
- If log files must be inspected, only read a small tail of the latest lines, such as the newest 10 or 20 lines. Do not load entire log files into context.

## Change Process

- For large refactors, first provide a step-by-step plan and wait for user confirmation before implementing.
- For normal scoped changes, proceed directly after gathering enough context.
- Keep edits minimal and localized to the task.

## Output Discipline

- Summarize command results instead of pasting long raw output.
- When verification produces long logs, report only the decisive lines, failures, and next action.
