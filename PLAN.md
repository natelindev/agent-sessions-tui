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

## Stage 5 - Complete transcript search and fluid result navigation

- Index searchable messages throughout complete file-backed transcripts while deduplicating repeated terms.
- Let mouse-wheel navigation transition directly from search editing to result browsing.
- Add regressions for late transcript matches and focused-search wheel navigation.

## Stage 6 - Incremental startup and complete search input

- Cache normalized file-backed session indexes in an app-owned SQLite database.
- Render the cached snapshot immediately while validating source files in the background.
- Reparse only new or changed transcripts and remove cache entries for deleted sessions.
- Accept dedicated terminal space-key events so multi-term searches work consistently.
- Verify cache hits, invalidation, removal, and multi-term input with regression tests.

## Agent progress

- [x] Audited the source app's supported providers, storage locations, and resume rules.
- [x] Implemented provider discovery and normalization.
- [x] Implemented the keyboard and mouse TUI.
- [x] Completed automated and interactive verification.
- [x] Added provider colors and the local installation workflow.
- [x] Prepared the public repository, demo screenshot, attribution, license, and CI workflow.
- [x] Verified complete transcript search and direct wheel navigation from the search field.
- [x] Added incremental session caching and reliable space-separated search input.
