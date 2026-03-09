# ms-cli Architecture

The current architecture reference lives in [`docs/ms-cli-arch.md`](./ms-cli-arch.md).

That document reflects the packages and directories that exist in this checkout, including:

- `cmd/ms-cli` and `internal/app` as the entry and wiring layers
- `agent/*` for orchestration, planning, session, memory, and context
- `integrations/*`, `permission/`, `tools/*`, `runtime/shell/`, `trace/`, `report/`, and `ui/*`

This file remains as a stable pointer so older references do not break, but `ms-cli-arch.md` should be treated as the source of truth.
