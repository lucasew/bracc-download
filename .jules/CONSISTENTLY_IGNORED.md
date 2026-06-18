## IGNORE: Scope Creep & Bundling "Nice-to-Have" Changes

**- Pattern:** Bundling unrelated modifications or "nice-to-have" refactors alongside the main goal of a pull request (e.g., extracting HTTP constants in a JobRuntime refactor, renaming variables in an error handling fix, or creating global `AGENTS.md` files in a provider-specific PR).
**- Justification:** Violates the "Scope Discipline" global directive. Changes must be minimal, atomic, and strictly scoped to the explicitly requested outcome to enable quick course correction and predictable reviews.
**- Files Affected:** `go.mod`, `go.sum`, `cmd/bracc/download.go`, `pkg/httpcontext/httpcontext.go`, `pkg/provider/dou/dou.go`, `pkg/provider/portal_transparencia/portal_transparencia.go`, `pkg/provider/simple/simple.go`, `AGENTS.md`

## IGNORE: Unauthorized SECURITY-NOTE Comments

**- Pattern:** Adding ad-hoc `// SECURITY-NOTE:` comments to document vulnerabilities, open data access, or sanitized paths.
**- Justification:** Project security conventions explicitly forbid ad-hoc security comments. Only the exact `// SECURITY: [severity] — [description]` format is allowed, and it must exclusively be used to escalate Critical or High vulnerabilities that require >50 lines to fix.
**- Files Affected:** `pkg/provider/eu_sanctions/eu_sanctions.go`, `pkg/provider/simple/simple.go`
