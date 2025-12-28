# Track Plan: Maintenance & Refinement

## Phase 1: Environment & CLI Synchronization
Goal: Ensure the application correctly detects and configures the latest `claude-code` environment.

- [x] Task: Audit environment detection logic for Node.js and `@anthropic-ai/claude-code`. 61b61e4
- [x] Task: Refactor: Unify Node.js version constant across platforms (Target: 22.14.0). 0e8012f
- [ ] Task: Implement Feature: Add version check and auto-update logic for `claude-code`.
- [ ] Task: Update model synchronization to support any new parameters in `claude-code` v0.x.
- [ ] Task: Write Tests: Verification of `~/.claude/settings.json` and `.claude.json` updates.
- [ ] Task: Implement Feature: Robust file system watchers for configuration changes.
- [ ] Task: Conductor - User Manual Verification 'Environment & CLI Synchronization' (Protocol in workflow.md)

## Phase 2: UI/UX Refinement (Vibe Coding)
Goal: Polishing the multi-project tabbed interface.

- [ ] Task: Refine Tabbed Interface CSS/Layout for better visibility and responsiveness.
- [ ] Task: Write Tests: Unit tests for project switching logic in `App.tsx`.
- [ ] Task: Implement Feature: Improved visual feedback when switching projects or changing working directories.
- [ ] Task: Conductor - User Manual Verification 'UI/UX Refinement (Vibe Coding)' (Protocol in workflow.md)

## Phase 3: Stability & Bug Fixes
Goal: Address known issues and improve overall robustness.

- [ ] Task: Fix Single Instance Lock issues on Darwin (macOS).
- [ ] Task: Improve Tray Menu reliability and responsiveness.
- [ ] Task: Write Tests: Integration tests for the "Launch Claude Code" trigger.
- [ ] Task: Conductor - User Manual Verification 'Stability & Bug Fixes' (Protocol in workflow.md)
