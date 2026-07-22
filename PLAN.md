# Agent Sessions TUI implementation plan

## Stage 1 - Core model and discovery

- Normalize session metadata from Codex, Claude, Antigravity, OpenCode, Hermes, Copilot CLI, Droid, OpenClaw, Cursor, and Pi.
- Read local files and SQLite indexes without modifying provider data.
- Build compact searchable text from identifiers, project metadata, and transcript content.

## Stage 2 - Terminal interface

- Render a responsive session table with a clear selection and detail footer.
- Add immediate search, keyboard navigation, mouse selection, wheel scrolling, and double-click resume.
- Replace the TUI process with the provider resume command so the resumed agent owns the current terminal.

## Stage 3 - Verification and delivery

- Cover parsing, filtering, command construction, and viewport behavior with tests.
- Run formatting, tests, static analysis, build, and an interactive terminal smoke test.
- Document installation, supported providers, controls, data access, and limitations.

## Stage 4 - Public release

- Add a privacy-safe product screenshot and explain the relationship to the original project.
- Publish the MIT-licensed source under the owner's personal GitHub account.
- Add continuous verification for tests, static analysis, and builds.

## Agent progress

- [x] Audited the source app's supported providers, storage locations, and resume rules.
- [x] Implemented provider discovery and normalization.
- [x] Implemented the keyboard and mouse TUI.
- [x] Completed automated and interactive verification.
- [x] Added provider colors and the local installation workflow.
- [x] Prepared the public repository, demo screenshot, attribution, license, and CI workflow.
