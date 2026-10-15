---
name: documentation-maintenance
description: Audit, reconcile, and update repository documentation against implementation, configuration, and accepted decisions. Use for documentation drift, duplicate guidance, broken references, ADR consistency, or syncing docs after changes. Not for general code review or implementing undocumented features.
---

# Documentation maintenance

Keep documentation for this Go IMAP/SMTP MCP server accurate, discoverable, and owned in one place.
Use this workflow for a targeted fix or a broader documentation audit; scale the
investigation to the requested scope.

## Establish scope and ownership

Start with [AGENTS.md](../../../AGENTS.md) for repository policy. Inspect the
working tree and relevant diff when the request concerns recent changes.
For an audit-only request, report findings without editing. For a maintenance
request, make relevant corrections directly within the authorized scope.

Find documents describing the affected feature and the code or configuration
supporting their claims. Search for related references and repeated guidance
before editing; do not read the whole documentation tree for a local change.

Use the existing document that owns the information:

- [README.md](../../../README.md): project entry point, setup, and development commands.
- [CONTRIBUTING.md](../../../CONTRIBUTING.md): contribution workflow and links to
  development checks and their defining configuration.
- [Architecture](../../../docs/ARCHITECTURE.md): architectural rationale and design decisions.
- [ADR template](../../../docs/adr/template.md): record structure; follow AGENTS.md
  for record lifecycle. Review relevant numbered records in `docs/adr/`.
- Existing plans in `docs/planning/`, when present: change-specific scope,
  progress, and completion criteria.

`CLAUDE.md` is only an entry point to AGENTS.md; do not duplicate project guidance there.

These links identify owners, not content to copy into this skill. Follow
AGENTS.md for documentation locations and exceptions. Flag out-of-scope location
issues instead of relocating unrelated files during another maintenance task.

## Reconcile claims with evidence

Check concrete claims against the artifacts that define them:

- Build, test, and lint commands: `Makefile`, `.golangci.yml`, and `.github/workflows/ci.yml`.
- Go version, module identity, and dependencies: `go.mod` and `go.sum`.
- Startup, transport, and shutdown: `cmd/mcp-server-email/main.go`.
- Account configuration and provider defaults: `internal/config/` and `examples/`.
- Authentication and token storage: `internal/auth/`.
- Tool names, schemas, registration, and behavior: `internal/tools/` and its tests.
- Message models, IDs, and errors: `internal/models/`.
- IMAP/SMTP behavior, connection handling, and retries: `internal/imap/`,
  `internal/smtp/`, and `internal/retry/`.
- Status resources and logging: `internal/resources/` and `internal/log/`.
- Packaging and releases: Dockerfiles, `.github/workflows/release.yml`,
  `.goreleaser*`, `scripts/build-mcpb.sh`, `Formula/`, and registry manifests.

Read only the artifacts relevant to the request. Verify implementation before
claiming a transport, permission restriction, provider, or tool is supported.
Distinguish the fork's repository URL from its Go module path and upstream
installation/release URLs; do not replace all upstream references indiscriminately.

A configuration or test definition establishes intent or coverage, not that a
check was executed or passed. Do not connect real mailboxes, send email, alter
messages or folders, export attachments, or deploy containers merely to verify
examples. Use the documented Greenmail test setup when integration validation
is warranted, with fake credentials and dedicated test data. Keep credentials,
tokens, and private email content out of documentation and validation output.

Distinguish current behavior, accepted intent, historical decisions, and proposed
work. When code conflicts with an accepted ADR, surface the discrepancy rather
than silently declaring the code correct, rewriting the decision, or changing
implementation to match the prose. Make a superseding ADR only when a replacement
decision is authorized; otherwise record the unresolved mismatch in the existing
appropriate owner or report it.

Use Git history when historical rationale matters. Do not invent dates,
alternatives, approvals, or validation results to fill gaps. Distinguish
"no alternatives were documented" from evidence that none were considered.

## Repair without duplication

Update the owning document and replace repeated instructions elsewhere with
links. Keep historical explanation where needed for an ADR, while linking to the
owner of current procedures. Preserve useful details when consolidating; do not
replace them with a link to a document that does not contain them.

After moving or renaming a document, repair inbound links, relative paths, and
index entries. Search for old paths, commands, and terminology within the
affected scope. Preserve intentional historical references and clearly mark
their provenance. Avoid unrelated code changes and blanket rewrites of all docs.

## Validate and report

Check edited Markdown for local link targets, heading anchors, relative paths,
and whitespace. Review examples against actual APIs and configuration; report
external links or commands that could not be verified. When ADRs change, check
their required sections, numbering, index entries where an index exists, and
supersession links using the current repository policy and template.

Run documentation checks already configured for the scope. Use CONTRIBUTING.md
for relevant application checks if executable behavior changed. For prose-only
edits, targeted link, structure, and diff checks are normally sufficient.

Report what was corrected, where maintained guidance now lives, checks actually
performed, and remaining discrepancies. Do not generate a separate audit report
or session log unless requested; use existing plan entries for
durable follow-up when appropriate.
