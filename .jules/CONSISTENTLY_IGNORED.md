## IGNORE: Premature Abstraction (Rule of Three Violation)

**- Pattern:** Extracting logic into standalone helper functions, structs, or new modules (e.g., `httpcontext.Get`, `downloadFileAtomically`, `JobRuntime`, `ui.NewMultibarFactory`) when the duplicated code does not yet appear in at least three places.
**- Justification:** The project strictly enforces the "Rule of Three". Code should not be abstracted or extracted for reuse until there are three clear use cases.
**- Files Affected:** `pkg/httpcontext/httpcontext.go`, `pkg/provider/simple/simple.go`, `pkg/provider/types.go`, `pkg/provider/runtime.go`, `cmd/bracc/download.go`, `pkg/ui/progress.go`

## IGNORE: Flattening Provider Subdirectories

**- Pattern:** Moving individual provider implementation files directly into `pkg/provider/` (e.g., moving `dou/dou.go` to `dou.go`) and removing their dedicated subdirectories.
**- Justification:** The project structure dictates that `pkg/provider/` is for core execution interfaces, while `pkg/provider/*` subdirectories are specifically for individual provider implementations. Flattening them violates this architectural colocation rule.
**- Files Affected:** `pkg/provider/dou/dou.go`, `pkg/provider/eu_sanctions/eu_sanctions.go`, `pkg/provider/portal_transparencia/portal_transparencia.go`, `pkg/provider/simple/simple.go`, `pkg/provider/cvm_processo_sancionador/cvm.go`

## IGNORE: Direct Logging Instead of Centralized Error Reporter

**- Pattern:** Using logging functions like `slog.Error` directly at the call site to handle unexpected errors (e.g., within deferred `Close()` calls).
**- Justification:** All code paths that handle unexpected errors MUST funnel through a centralized error-reporting function (e.g., `pkg/errorreporter.ReportError`). Direct logging of errors circumvents this system.
**- Files Affected:** `pkg/provider/portal_transparencia/portal_transparencia.go`, `pkg/provider/simple/simple.go`, `pkg/provider/webdav/webdav.go`

## IGNORE: Unauthorized SECURITY-NOTE Comments

**- Pattern:** Adding ad-hoc `// SECURITY-NOTE:` comments to document vulnerabilities, open data access, or sanitized paths.
**- Justification:** The security conventions only recognize the exact `// SECURITY: [severity] — [description]` format. It must only be used to escalate Critical or High vulnerabilities that require more than 50 lines to fix.
**- Files Affected:** `pkg/provider/eu_sanctions/eu_sanctions.go`, `pkg/provider/simple/simple.go`

## IGNORE: Fixing Complex Vulnerabilities Instead of Escalating

**- Pattern:** Implementing full code fixes and test suites for Critical or High security vulnerabilities when the overall fix exceeds 50 lines (e.g., fixing Path Traversal in SimpleJobProvider).
**- Justification:** Critical or High vulnerabilities requiring >50 lines to fix cleanly MUST NOT be fixed directly. They must be escalated by leaving a `// SECURITY: [severity] — [description]` comment at the vulnerable site and documenting the full scope in the PR body.
**- Files Affected:** `pkg/provider/simple/simple.go`, `pkg/provider/simple/simple_test.go`

## IGNORE: Incorrect Arrumador PR Title

**- Pattern:** Using incorrect emojis or prefixes in Arrumador PR titles (e.g., `🛟 Arrumador:`).
**- Justification:** Arrumador PRs must strictly use the exact title prefix `🛠️ Arrumador: [Description]`.
**- Files Affected:** `.github/workflows/autorelease.yml`, `mise.toml`

## IGNORE: Multi-Area Documentation Scope

**- Pattern:** Modifying documentation across multiple distinct domain areas in a single PR (e.g., updating HTTP utilities alongside provider orchestration, or adding global project conventions in `AGENTS.md` alongside a provider-specific PR).
**- Justification:** Documentation tasks must be strictly scoped to a single cohesive area per PR to ensure focused reviews and adhere to scope boundaries.
**- Files Affected:** `pkg/httpcontext/httpcontext.go`, `pkg/provider/progress.go`, `AGENTS.md`, `pkg/provider/portal_transparencia/portal_transparencia.go`

## IGNORE: Trivial "What" Docstrings Instead of "Why"

**- Pattern:** Writing docstrings that merely restate what the code trivially says (e.g., `// GetURL returns the absolute URL`, `// SimpleJobProvider is a static, single-URL provider`).
**- Justification:** Documentation must focus on non-obvious nuances, data flow, and "why" something is done over "what" the code trivially says.
**- Files Affected:** `pkg/provider/simple/simple.go`
