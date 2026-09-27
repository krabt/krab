# Repository Guidelines

## User-facing text

- Every new or changed user-facing label, button, tooltip, message, placeholder, and empty state must support both Chinese and English.
- Add the Chinese and English strings to `frontend/src/i18n.js` and render them through the existing `t(...)` translation helper. Do not hardcode bilingual UI text directly in components when a translation key can be used.

## Server duplication

- A duplicated server must be stored as an independent server with a new ID.
- Its default name must be the original server name followed by the literal lowercase suffix ` copy`, regardless of the current display language. For example, duplicating `krweb` creates `krweb copy`.
- A duplicate created from a subscription server must not retain subscription metadata that would cause a later subscription refresh to replace or remove it.

## Cross-platform support

- Every operation, feature, and code change must support macOS, Linux, and Windows by default. This applies to frontend behavior, backend logic, filesystem access, process execution, networking, configuration, builds, packaging, updates, and tests.
- Do not implement an operation only for the current development OS. When operating-system APIs differ, provide platform-specific implementations behind build constraints with equivalent behavior on all three supported platforms.
- If equivalent behavior is impossible because the operating system does not provide the required capability, implement a safe, explicit fallback and document the limitation in both Chinese and English in the user interface.
- Use Go's `filepath` and `os` helpers for filesystem paths. Do not hardcode `/`, `\\`, Unix home directories, drive letters, or platform-specific configuration directories.
- User-entered paths that begin with `~`, including forms such as `~/.krab` and `~\\.krab`, must resolve against the current user's home directory on macOS, Linux, and Windows before validation or use.

## Persistence

- Store structured application configuration and persistent data in SQLite at `~/.krab/krab.db` on macOS, Linux, and Windows.
- Do not introduce new JSON, YAML, local-storage, registry, or platform-specific persistence for application configuration or historical data. Operational files required by third-party components, such as the xray-core text log, may remain separate under `~/.krab`.
- Preserve backward compatibility by migrating legacy persisted data into SQLite when its database key or table is still empty.
