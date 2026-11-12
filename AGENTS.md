# Agent Instructions

Read the applicable specifications in `agents/specs/` (when present) before modifying code:
- `agents/specs/prd.md`: product scope, user stories, and acceptance criteria *(created in Product step)*.
- `agents/specs/design.md`: UI/API/CLI design contracts and mockup references (`*.png`, `*.svg`, `*.json`, `*.html`) *(created in Design step)*.
- [`agents/specs/architecture.md`](agents/specs/architecture.md): package boundaries, dependency direction, and composition rules.

## Context and Progress Tracking

Agents use `agents/context/` to maintain living project context and verification gates across sessions:
- [`agents/context/history.md`](agents/context/history.md): chronicle of project milestones, phases, and snapshots in reverse chronological order (most recent first).
- [`agents/context/fases.md`](agents/context/fases.md): implementation phases, roadmaps, and step-by-step checklists.
- `agents/context/qa-report.md`: integration and E2E verification report (`STATUS: PASS | FAIL`) *(created in QA step)*.
- `agents/context/security-report.md`: security hardening and vulnerability audit report (`STATUS: PASS | FAIL`) *(created in Security step)*.

## Repository Standards

- **Relative paths only**: All documentation links, scripts, and configs must remain path-agnostic from the repository root backwards.
- **Explicit Git consent**: Never run `git commit` or `git push` autonomously; follow the active Git skill (e.g., `edro-git`) or repository commit conventions when the user requests a commit.
