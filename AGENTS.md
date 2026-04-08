# Agents Operating Conventions

This repository contains conventions and mapping pointers to guide automated agents (and humans).

## Project Areas Mapping

- `cmd/bracc/` -> CLI application entrypoints and main execution setup (Cobra commands).
- `pkg/provider/` -> Core job execution and interface definitions for different data sources.
- `pkg/provider/*` -> Individual provider implementations (e.g., `portal_transparencia`, `simple`, `webdav`) handling source-specific parsing and downloading.
- `pkg/httpcontext/` -> Internal wrappers to manage and inject HTTP clients via context.
- `prelude/` -> Initialization side-effects and global registrations (often empty or simple imports to trigger `init()`).

## Conventions

- **Formatting/Linting:** Use `go vet`, `go fmt`, `staticcheck`, `errcheck`, and `goimports`. Standard Go tooling applies.
- **Documentation:** Use idiomatic single-line Go docstrings (`//`) directly above declarations. Focus on non-obvious nuances, data flow, and "why" something is done over "what" the code trivially says.
- **Error Handling:** Errors must never be swallowed. Deferred cleanups (e.g., `resp.Body.Close()`) must handle and capture returned errors, using the centralized logging (`log/slog`).
- **Security:** Run `gosec` for vulnerability checks. For critical/high security issues taking >50 lines to fix, use a `// SECURITY: [severity] — [description]` comment to escalate.
