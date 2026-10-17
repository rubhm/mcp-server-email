> Reference brief: preserved as source material. The [numbered roadmap](../README.md) governs implementation order and status. The phases and immediate assignment below are historical proposals, not current execution instructions.

# LocalAI Email MCP — Coding Agent Implementation Brief

**Status:** Implementation handoff · **Prepared:** 2026-10-15\
**Target repository:** https://github.com/rubhm/mcp-server-email\
**Upstream:** https://github.com/boutquin/mcp-server-email\
**Source license:** MIT; preserve Pierre Boutquin's copyright and LICENSE notice.\
**Priority:** P0: secure, usable read-only Streamable HTTP MCP on Unraid; P1+: gradual managed actions.

## 1. Executive objective

Build a maintainable, self-hosted **multi-account email MCP server** starting from the existing Go codebase in `rubhm/mcp-server-email`. It will expose standard IMAP/SMTP mail capabilities to **Open WebUI**, whose existing LocalAI-hosted **Qwen ~27B Q4** model uses these tools to summarize, search, and later help manage email. This is **one specialist tool service used by a future general-purpose assistant**, not a general-purpose MCP or a new AI orchestrator.

**Primary first-use case:** In Open WebUI, ask: “Summarize my important emails from the last 24 hours across my authorized mailboxes; show actions and deadlines.” Get accurate, attributable summaries without altering mailbox state or transmitting email content to external model APIs.

**Longer-term use cases:** Follow threads, triage and flag, organize mail, save drafts, send replies with independent approval, and later coordinate with Paperless/Odoo. Those integrations belong to the assistant/orchestrator, **not** this MCP server.

## 2. Environment and constraints

- Host: **Unraid**, Docker containers. Existing **Docker Mailserver (DMS)** directly hosts the mailboxes and uses Dovecot IMAP and Postfix SMTP; there is no fetchmail synchronization layer.
- AI: Existing **Open WebUI → LocalAI → Qwen ~27B Q4**, running locally on RTX 4090 24 GB / 64 GB DDR5 host. **No model inference inside the MCP.**
- Multiple mailboxes from day one (personal, spouse with explicit authorization, shared household, business); may span more than one account/domain. Each account has independent IMAP/SMTP credentials.
- Open WebUI must connect via **MCP Streamable HTTP** from another container over a private Docker network, not only stdio; keep stdio compatibility for other MCP clients.
- Local-first privacy. Do not require Gmail, cloud services, Home Assistant, reverse cloud tunnels, vector databases, or an external LLM API.
- Store configuration/secrets in controlled persistent paths, not in `docker.img`, git or image layers. No bulk duplication of mailbox contents for P0. Keep logs metadata-only; existing Loki is available later.
- Internal DNS/hostname must match the DMS TLS certificate; verify certificates, including when containers share a private Docker network. Avoid falling back to plaintext or `insecure_skip_verify` in production.

## 3. Baseline codebase: verify before modifying

The upstream Go project currently uses Go 1.24, `github.com/mark3labs/mcp-go` v0.43.2, `github.com/emersion/go-imap/v2` v2.0.0-beta.8, and `github.com/wneessen/go-mail` v0.6.2 (versions observed 2026-10-15; lock to the fork's actual `go.mod`). It has a `cmd/` executable, `internal/imap`, `smtp`, `tools`, `models`, `config`, `auth`, `retry` packages; stdio entrypoint; ~22 tools; Dockerfiles; unit/fuzz/integration tests. Preserve those assets.

**Verified issues to address:**
1. `cmd/mcp-server-email/main.go` calls `server.ServeStdio`; there is no native HTTP startup path wired into the executable.
2. `internal/tools/register.go` registers all tools, including send, draft-send, destructive operations, without a P0 server-side read-only gate. `SendHandler` reaches `ops.Send` if normal validation passes.
3. `internal/tools/attachments_download.go` accepts arbitrary absolute `saveTo` paths, performs basic validation and writes with `os.WriteFile`; this is **not** a sufficiently constrained filesystem sandbox for an LLM tool.
4. Message IDs use `account:mailbox:uid` but lack IMAP `UIDVALIDITY`; stale references can become unsafe if a mailbox is recreated.
5. There is no verified robust mapping from authenticated Open WebUI identity to per-account authorization in the current server.

Do not mistake MCP tool annotations, UI hiding, or prompting the LLM for **enforced** security controls. Read and inspect code and tests before assuming anything else about behavior.

## 4. Guiding architecture

```text
User (Open WebUI, later voice/tablet)
    │
    ▼
Existing Open WebUI orchestration + LocalAI/Qwen
    │   MCP Streamable HTTP (authenticated, private service)
    ▼
Email MCP (this Go repo)
    ├─ Transport/authentication boundary
    ├─ Authenticated principal + resource/action policy engine
    ├─ MCP handlers (account discovery, list/search/read/thread, later actions)
    ├─ IMAP service and connection pool ───────► DMS/Dovecot :993 TLS
    └─ SMTP adapter (disabled in P0) ──────────► DMS/Postfix :465 TLS or :587 STARTTLS

Future independent approval service / interface → signed one-time approval → send authorization
```

Keep model reasoning and cross-application workflows **outside** the MCP. The MCP exposes narrow, dependable operations and enforces permissions itself. No Home Assistant integration is required for this project.

## 5. Phased implementation plan

### Phase 0 — Baseline and threat model (required first)

- Confirm fork ancestry, document upstream commit SHA/tag and all relevant dependencies. Preserve LICENSE/attribution and upstream remote.
- Run existing lint, unit and race tests; run fuzz seeds; identify flaky or untested components. Do not claim tests pass unless actually run. Capture baseline build/test results.
- Audit IMAP operations for side effects: use read-only mailbox selection (`EXAMINE` or equivalent) and non-marking fetch (`BODY.PEEK` where supported); verify that reading does not change `\\Seen` or any flags. Specifically test `email_unread`, full message reads, search and `email_thread`.
- Threat model: hostile email content as prompt injection; unauthorized access across accounts; leaked credentials; cross-container MCP exposure; arbitrary filesystem reads/writes; unintended SMTP delivery; mailbox deletions; duplicate sends; logging of private content.
- Define a small, explicit permission vocabulary: `mail.read`, `mail.organize`, `mail.draft`, `mail.send`, `mail.delete`; apply at the server boundary. P0 grants **only** `mail.read`.

### Phase 1 — Native HTTP and P0 read-only operation

- Add `stdio` (existing/default where appropriate) and `http` transport modes using the current MCP-Go library's supported **Streamable HTTP** API. Support configurable bind address, port, `/mcp` path, and clean shutdown. Prefer configuration flags/env vars with documentation and predictable defaults.
- **Do not expose an unauthenticated HTTP MCP endpoint to LAN/Internet.** Bind/route to a trusted private Docker network; implement authenticated caller identity at the boundary, initially via a strong bearer token or trusted reverse-proxy identity where carefully verified. Use TLS at network boundary where needed. Avoid accepting forgeable arbitrary user headers as identity.
- Implement a **hard P0 read-only server policy**: do not register mutating tools for read-only mode, and also deny mutating calls at a central authorization boundary (defense in depth). IMAP credentials may have underlying write rights; tool restrictions must actually prevent writes.
- Initially expose only minimal tools such as account list (filtered to authorized accounts), folder list, list/search/unread messages, read one message, and read conversation thread. Ensure metadata is bounded; message body only on explicit request and bounded by size, count, and time range.
- Use consistent structured error codes with no secrets or sensitive email bodies in error messages.
- Ensure normalized results contain account, mailbox, stable reference, sender, recipients as needed, subject, timestamp (timezone aware), flags, threading metadata, and bounded plain-text body. Treat HTML as untrusted; sanitize and avoid executing/fetching remote content. Attachment metadata only in P0.
- Protect health endpoint from leaking mailbox addresses or credential details.
- Ship a documented Dockerfile/Compose example, unprivileged container user, read-only root FS where feasible, dropped Linux capabilities, `no-new-privileges`, constrained resources, private bridge network, no unnecessary host port exposure, pinned base images/dependencies where practical.

### Phase 2 — Multi-account security (before shared household use)

- Formalize a **principal → allowed accounts → permitted actions** policy model. Expose no account merely because the MCP process has its credentials. Every tool call validates principal and target account, including IDs that encode an account. No “default account” that silently bypasses authorization.
- Design for separate Open WebUI users (owner, spouse, possibly business accounts). Verify whether Open WebUI actually propagates a trustworthy per-user identity/token; if not, use separate client credentials or separate MCP instances per trust boundary, rather than assuming UI permissions are enough.
- Provide per-account config with environment/secret-file references rather than logging/exposing plaintext passwords. Restricted file ownership/permissions, startup validation, and redacted config/errors.
- Authorization tests: unauthorized account lookup, guessed message ID, tool discovery filtering, move/search across accounts, spoofed headers, malformed composite IDs, concurrent callers.
- Decide between isolated instances per user and single shared service based on **verified** authentication mechanics; document residual risks.

### Phase 3 — Controlled organization and drafts

- Permit `mail.organize` and `mail.draft` separately, controlled per principal/account. Add safe flags, move/archive, append draft; log metadata-only audit events with actor, action, account alias, result and correlation ID.
- Prefer reversible moves to known folders; disable permanent delete entirely initially. If `Trash` is ambiguous, refuse rather than expunge. Add dry-run/preview option for bulk actions.
- Handle Dovecot folder naming, special-use flags, real threading headers, UIDVALIDITY and mailbox UID changes. Ensure attachments do not grant arbitrary access to host paths, HTTP URLs, or unrelated shares. Use a configured limited storage directory for any exported attachment.
- Test drafts with IMAP APPEND against DMS; ensure drafts appear in a normal mail client and aren't sent.

### Phase 4 — Outbound mail with independent human approval

- **Never** unlock direct `email_send` / `email_reply` / `email_draft_send` just because the model claims a user approved it. The model can be manipulated by incoming email text.
- Separate prepare/preview from confirmation. A person reviews **exact From, To, Cc, Bcc, subject, body and attachments** outside the model's tool-call loop. Confirmation must be performed through a trusted UI/service or authenticator the model cannot directly call.
- Bind approval to authenticated actor, account, normalized recipients, content hash, attachment hashes, expiry, and one-time nonce. Persist or atomically consume authorization to prevent replay or duplicate sends. Fail closed for mismatches.
- Add SMTP submission controls, recipient/volume limits, optional contact allowlists, and detailed metadata-only audit trails. Address partial failures/ambiguous SMTP status without automatic duplicate retries.
- Keep `mail.delete` a separate later decision; no permanent expunge by default.

## 6. Recommended implementation approach for code agents

- **Small PRs on feature branches**, no large sweeping rewrite. Preserve and reuse upstream IMAP/SMTP adapters and tests.
- First PR: `docs/` implementation plan, threat model, baseline results, and failing tests for read-only/transport behavior where feasible.
- Second PR: transport selection and HTTP MCP with authentication and health checks; retain stdio compatibility.
- Third PR: centralized read-only policy, tool allowlisting, read non-mutation guarantees, and security tests.
- Fourth PR: Unraid Docker Compose and Open WebUI connection instructions, plus DMS integration test harness.
- Subsequent PRs: identity-aware authorization, UIDVALIDITY, drafts, approved sends.
- Prefer constructor-injected interfaces and middleware rather than embedding permissions in each handler; any critical action should still perform a final guarded check close to its underlying IMAP/SMTP operation.
- Keep CI checks for `go test ./...`, `go test -race ./...`, lint/static analysis, dependency scans, and tagged integration tests where supported. Never silently disable or rewrite upstream tests to make CI green.
- No production credentials in test fixtures or CI; use fake/ephemeral DMS/GreenMail mailboxes with TLS.

## 7. Definition of Done — first deployable milestone (P0)

**Functionality**
- [ ] Fork builds reproducibly as a Docker image on Unraid Linux/amd64.
- [ ] Two configured DMS test mailboxes can connect over certificate-verified IMAPS.
- [ ] Open WebUI connects to `http(s)://<private-email-mcp-host>:<port>/mcp` (actual URL documented) using native Streamable HTTP, authenticated and inaccessible from untrusted networks.
- [ ] From Open WebUI/Qwen, a prompt can search and summarize last-day email **from authorized accounts** with accurate sender/date/thread references and actions/deadlines.
- [ ] Plain text / sanitized HTML, pagination, message limits, and timezone handling work.

**Security**
- [ ] No model-callable send, reply, delete, draft-send, move, mark-read or other mutation in P0; tested at tool registration and direct-handler levels.
- [ ] Read and search do not change IMAP `\\Seen`, flags, or mailbox contents in DMS; tested against test mailboxes.
- [ ] Unauthorized accounts and fabricated/altered composite message IDs are denied.
- [ ] No email bodies, credentials, raw tokens or attachment contents appear in application logs or CI output.
- [ ] No arbitrary filesystem read/write tool surface; attachment exports disabled or sandboxed.
- [ ] Host paths, container privileges, ports and network exposure documented and minimal.

**Quality**
- [ ] Baseline tests remain green; new transport/policy tests, integration tests and negative authorization tests pass.
- [ ] Docs include install, rollback, configuration, secrets, certificate troubleshooting and a minimal Open WebUI connection example.
- [ ] LICENSE retained, upstream credited, changed behavior documented in CHANGELOG/README.

## 8. Immediate agent assignment

**Start here:** Clone `https://github.com/rubhm/mcp-server-email`; inspect actual main branch and test status; do not assume any of the planned improvements exist. Produce (1) a short current-state audit with specific file/function references and high-risk gaps; (2) a proposed minimal design for native HTTP transport and hard read-only policy; (3) tests showing failure before implementation; and then (4) an initial small PR implementing those two features with documentation and automated tests.

**Do not** connect real personal/business mailboxes, enable SMTP, expose a public MCP endpoint, or perform destructive mailbox operations during the first phase. Ask for missing hostnames/secrets only at deployment time, and do not request actual passwords in chat or commit them.

## 9. Explicit non-goals for P0

- General-purpose MCP gateway, memory/RAG database, autonomous email processing daemon, background inbox monitoring.
- Home Assistant/voice interface, cloud LLM delegation, third-party Gmail-specific logic.
- Automatic archiving, sending, deletion, invoice posting, or Odoo/Paperless workflows.
- Building an email client UI or duplicating message bodies into a new searchable datastore.

## 10. Handoff / evidence expected from agents

For each PR, report: branch and commit SHA, changed files, architectural choices, security invariants, build/test commands with actual outputs, known limitations, exact Docker run/Compose instructions, how to reverse the changes, and outstanding issues. Prefer links to repository PRs over copied code dumps. Do not mark a milestone complete on the strength of README claims alone.
