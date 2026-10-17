# 0001: Baseline and threat model

## Purpose

Establish evidence about the existing fork and resolve security and compatibility
questions before implementing the [P0 roadmap](README.md).

## Scope

Inspect the fork, dependencies, CI, IMAP reads, tool/resource exposure,
configuration, logging, and deployment assumptions. Run baseline checks where
tooling permits. No production mailbox access or feature implementation.

## Expected Behavior

Keep confirmed current behavior separate from handoff claims and proposed work.
The inspected entrypoint serves stdio and calls `tools.RegisterAll`; the status
resource includes account data. Reads select mailboxes with nil options. In
`buildFetchOptions`, only the References header sets `Peek: true`; the optional
TEXT section does not. Body reads can therefore mark messages seen. Attachment
body fetching also omits Peek. Non-mutation is not an existing guarantee.

Resolve these prerequisites for later specifications:

- P0 trust boundary: proposed separate deployment/credential per group of callers
  authorized for every configured account. Do not assume a shared bearer token
  identifies individual Open WebUI users. Record the chosen model and alternatives
  in an ADR; mixed-permission shared access needs explicit authorization design.
- Message references: decide whether to implement UIDVALIDITY-aware references in
  P0 or narrow the reference contract with an explicit stale-reference limitation.
  Do not promise stable references without evidence or silently permit misrouting.
- Verify the pinned MCP library's transport/authentication hooks and the target
  Open WebUI integration using authoritative version-specific evidence.
- Define the read-only tool allowlist, resource visibility, permission vocabulary,
  response/body limits, and compatibility expectations for existing stdio clients.
- Model hostile email content, cross-account access, credential leakage, arbitrary
  filesystem access, mutations, unintended SMTP, and private-data logging.

## Acceptance Criteria

- [x] Fork ancestry/upstream commit, dependency versions, and baseline commands/results are recorded here without secrets.
- [x] Build, race tests, lint, and relevant fuzz/integration coverage have recorded outcomes; unavailable checks remain explicit gaps.
- [x] Relevant code paths and existing protections are identified with file/function references.
- [x] The trust-boundary and reference-contract decisions are documented, with accepted significant decisions in ADRs.
- [x] The threat model has mitigations and planned negative tests linked to 0002/0003.
- [x] Plans 0002-0004 are refined with concrete configuration, limits, interfaces, and testable contracts; no prerequisite design question is silently assumed resolved.

## Implementation Plan

1. Inspected local provenance, configuration, CI, tools/resources, and read paths.
2. Completed containerized build, uncached race tests, lint, and Greenmail tests;
   verified all six exact-revision CI fuzz targets passed.
3. Recorded source evidence and threat model below; accepted ADRs 0001/0002.
4. Refined dependent plans and reviewed all acceptance criteria; 0002 is ready.

## Validation

Inspection date: 2026-10-17. Application revision:
`a56024516f4bf4ca8532c1ba8fffc660f4b77d24`; initially clean working tree.
No Go code was changed by this investigation.

| Check | Actual outcome |
|---|---|
| Local tooling | No host Go/lint installation required. Docker Desktop 4.95.0, client/engine 29.8.2, linux/amd64 verified after startup. Sandbox denies the engine pipe; approved Docker commands outside the sandbox work. |
| `go build ./...` | Passed in local Docker, Go 1.24.13 linux/amd64, GCC 12. |
| `go test -race -count=1 ./...` | Passed in local Docker as UID/GID 10001:10001; all nine tested packages returned uncached `ok`. Entrypoint has no tests. |
| `golangci-lint run ./...` | Passed in local Docker, v2.11.3: `0 issues.` and exit code 0. Exact-revision CI lint also passed. |
| Six CI fuzz targets, 2 minutes each | Exact-revision job 117472293029 completed successfully. Logs show PASS for ParseMessageID, HtmlToText, BuildSearchCriteria, SplitAddresses, ExtractContentType and ExtractAttachmentByIndex (120-131s per command including package tests). |
| `go test -tags=integration -race -count=1 ./...` | Passed in local Docker as UID/GID 10001:10001 against dedicated Greenmail 2.1.0 on a private bridge; explicit IMAP/SMTP host/port and fake-account variables set. All nine tested packages returned uncached `ok`, including IMAP (12.611s) and SMTP (1.920s). |
| GitHub evidence | Combined status and PR-filtered run lookups were empty; direct Actions lookup found push run [39058810257](https://github.com/rubhm/mcp-server-email/actions/runs/39058810257) at exact HEAD. Subsequent inspection confirmed test, integration, lint and fuzz jobs all completed successfully; integration and fuzz logs inspected. |
| Initial container test failures | Root bypassed chmod restrictions in TestTokenStore_Save_CreateTempError, failing both suites. Source inspection confirmed the assumption; rerunning as UID 10001 passed without changing/skipping tests. Non-root standalone build initially hit Git ownership checks; explicitly trusting only /src via per-container Git configuration resolved it. |
| Integration log inspection | Go 1.24.13 linux/amd64; Greenmail 2.1.0 created; TEST_IMAP_HOST set; IMAP and SMTP packages returned uncached `ok` results (13.215s and 1.865s). No test failure shown. Non-verbose logs do not enumerate individual test cases. |
| Documentation checks | `git diff --check` passed; Python local-link and required-section checks passed across seven planning/ADR documents. |

Validation review is complete for this baseline. These results establish legacy
behavior only: they do not prove the future read-only policy, HTTP handshake,
Open WebUI compatibility, DMS non-mutation or deployment isolation. Those checks
remain acceptance gates in 0002-0004. No production mailbox was accessed.

### Container reproduction

Mount the checkout read-only at `/src`, set workdir `/src`, and use
`golang:1.24.13-bookworm`. Run the build/race commands above as `--user 10001:10001`
with `HOME=/tmp`, `GOCACHE=/cache`, and writable named module/build caches owned
by that UID. For a Windows bind mount owned by another UID, set per-container
`GIT_CONFIG_COUNT=1`, `GIT_CONFIG_KEY_0=safe.directory`, and
`GIT_CONFIG_VALUE_0=/src`. Do not use a wildcard safe.directory or disable VCS
stamping to hide the ownership error. Cache volumes were created specifically
for validation; the source checkout was never chowned or made writable.

For integration, attach the runner and Greenmail to a dedicated bridge without
publishing ports. Use README's Greenmail options and set both TEST_IMAP_HOST and
TEST_SMTP_HOST to the Greenmail container name, ports 3993/3465, and fake
`test@example.com` / `password` credentials. Wait for IMAPS readiness before
testing. Existing Greenmail fixtures deliberately use self-signed certificates;
these tests do not establish certificate verification for P0/DMS deployment.
Remove the temporary Greenmail container and network after the suite finishes.

Lint uses `golangci/golangci-lint:v2.11.3`, a read-only `/src` checkout and
separate module/lint cache volumes. Retain only reusable images/caches.

Image digests observed for reproducibility:

- Go: `sha256:1a6d4452c65dea36aac2e2d606b01b4a029ec90cc1ae53890540ce6173ea77ac`.
- Lint: `sha256:e838e8ab68aaefe83e2408691510867ade9329c0e0b895a3fb35eb93d1c2a4ba`.
- Greenmail: `sha256:308685b99ad840f05bd2dee43f47f7956f876adbf396523f68166f078300cd29`.

## Source audit

### Provenance and dependencies

`origin` is `https://github.com/rubhm/mcp-server-email.git`; no upstream remote
is configured. Local root commit is
`520db5ef3dd33c4eaadc6c63d340969ab4d60e90`; last local commit preceding fork
documentation is `0ba695c9126fca2b89959620729f5d2e65690d75`. GitHub's commit
API independently confirmed that SHA in
[upstream](https://github.com/boutquin/mcp-server-email/commit/0ba695c9126fca2b89959620729f5d2e65690d75).
`git diff 0ba695c HEAD -- '*.go' go.mod go.sum` is empty. This records the
fork baseline; it does not assert that the fork matches today's upstream tip.
LICENSE retains Pierre Boutquin's 2026 MIT attribution; module identity and
README upstream links are retained. No remotes or attribution were changed.

`go.mod` specifies Go 1.24.0, mcp-go v0.43.2, go-imap/v2 v2.0.0-beta.8,
go-mail v0.6.2, x/net v0.50.0; indirect OAuth2 is v0.35.0 and x/crypto v0.48.0.
CI specifies Go 1.24, golangci-lint v2.11.3, and Greenmail 2.1.0. The presence
of versions and CI jobs does not establish runtime compatibility or security.

### Current behavior and gaps

| Surface | File/function evidence | Implication |
|---|---|---|
| Startup | `cmd/mcp-server-email/main.go`: `main`, `newServer` | Stdio only; builds both pools and calls RegisterAll. Shutdown context is not passed into ServeStdio. |
| Exposed operations | `internal/tools/register.go`: `RegisterAll` | 22 tools, including SMTP, drafts, mutations, exports; no policy guard. |
| Account disclosure | `internal/resources/status.go`: `StatusHandler`; `internal/imap/pool.go`: `AccountStatus` | All configured aliases/addresses and default account visible; no caller identity. |
| Configuration | `internal/config/config.go`: `LoadFromEnv`, `GetAccount`, `applyEnvOverrides` | Unknown explicit accounts error; empty account uses default. No comprehensive duplicate/empty-ID or numeric override validation. |
| Read selection | `internal/imap/client_messages.go`: `GetMessage`, `ListMessages`, `ListUnread`, `Search`, `SearchByMessageID`, `GetAttachments` | All select with nil options; no EXAMINE. |
| Body fetch | `internal/imap/helpers.go`: `buildFetchOptions` | References header uses Peek; TEXT body does not. GetMessage is used by get/body/thread paths. |
| Attachment fetch | `internal/imap/client_messages.go`: `fetchAttachmentBody` | Whole-message body section omits Peek; collects message before attachment download-size check. |
| Folder reads | `internal/imap/client_folders.go`: `ListFolders`, `getFolderByRole` | LIST/STATUS, no selection; scans all folders and tolerates STATUS errors as zero counts. |
| Reference parsing | `internal/models/email.go`: `ParseMessageID`, `parseUID` | No UIDVALIDITY. First-two-colon split contradicts support for colons in mailbox names. Empty account/mailbox accepted; uint64 accumulation lacks pre-multiply overflow guard. |
| Volume/HTML | `internal/tools/messages.go`: `GetHandler`, `buildReadBodyResult`, `htmlToText`; `thread.go`: `searchThread` | Get preview retains full body; body slicing accepts unsafe limits; thread nested searches unbounded and silently skip errors. HTML parser does not fetch resources and skips script/style, but output remains untrusted. |
| Filesystem | `internal/tools/attachments.go`: `parseAttachments`; `attachments_download.go`: `handleSaveAttachment` | Caller paths support reads for sending and arbitrary absolute-path writes for exports. No sandbox root. |
| TLS/secrets/logs | `internal/imap/conn.go`: `dialTLS`, `dialSTARTTLS`; `internal/auth/store.go`: `Save`; `internal/smtp/client.go`: `Send` | TLS hostname/minimum version present; configurable skip-verify remains. Token files use temp/rename and restrictive modes. SMTP debug log includes subject; raw provider errors may expose private text. No global redactor. |

Existing connector mocks, injected Operations interfaces, retry timeout handling,
and test corpora are reusable seams. Existing tests exercise legacy behavior;
they do not prove authentication, policy enforcement, UIDVALIDITY, bounded
responses, or non-mutation. Add regression assertions in their owning plans;
no unexecutable failing test scaffolding was added during the audit.

### Library and client evidence

The pinned [mcp-go v0.43.2 transport source](https://github.com/mark3labs/mcp-go/blob/v0.43.2/server/streamable_http.go)
provides NewStreamableHTTPServer, ServeHTTP and context injection. Request context
is inherited; an outer authentication handler can attach verified identity.
Use WithStateLess(true) and WithDisableStreaming(true) for P0. These are library
capabilities, not application authentication. The handler reads POST bodies with
io.ReadAll, so enforce a limit first. Own routing and HTTP shutdown externally.

Pinned [go-imap selection source](https://github.com/emersion/go-imap/blob/v2.0.0-beta.8/imapclient/select.go)
maps SelectOptions.ReadOnly to EXAMINE; nil uses SELECT. Test protocol behavior
and retry/pool reuse rather than relying solely on this source observation.

User target is newest stable Open WebUI. Its [latest release](https://github.com/open-webui/open-webui/releases/tag/v0.12.3)
resolved to v0.12.3 on the inspection date. The [tagged client](https://github.com/open-webui/open-webui/blob/v0.12.3/backend/open_webui/utils/mcp/client.py)
uses streamablehttp_client with headers/auth, initializes ClientSession and
paginates tool discovery. [Official MCP documentation](https://docs.openwebui.com/features/extensibility/mcp/)
describes Bearer mode and its Key field. This supports the proposed integration;
it is not a tested handshake. Pin the tested version; recheck latest stable at
deployment and rerun compatibility if it changed. Installed version is unknown.

## Threat model and decisions

Accepted design contracts are in [ADR 0001](../adr/0001-p0-trust-boundary.md)
and [ADR 0002](../adr/0002-p0-message-references.md). They describe future behavior,
not implemented protections. Permission vocabulary: mail.read, mail.organize,
mail.draft, mail.send, mail.delete; only mail.read is granted in P0.

| Threat / boundary | Planned mitigation | Required negative evidence |
|---|---|---|
| Forged user identity or stolen session ID | 0002: one credential per isolated deployment, verified request context, stateless transport; ignore user/proxy identity headers | Missing/wrong/duplicate bearer, spoofed headers, arbitrary session IDs on every MCP method; no dispatch before authentication |
| Cross-account access/discovery | 0003: instance configured only with accounts its callers may read; guard accounts/resources and decoded references; validate defaults | Unknown/empty/duplicate account configuration, fabricated IDs, independent instances with distinct credentials; no network call on denial |
| DNS rebinding, browser requests, untrusted network | 0002: exact Host allowlist, reject Origin, no CORS, private network; TLS outside trusted bridge | Foreign Host/Origin, forwarded-header spoofing, oversized and slow requests |
| Malicious mail prompts invoking mutations/SMTP/files | 0003: allowlist plus guarded Operations, no SMTP pool in read-only mode, deny exports and file attachment inputs | Direct mutation handlers and guarded methods cannot issue Store/Move/Copy/Append/Create/Expunge, SMTP, file read/write |
| Reads change Seen or mailbox contents | 0003: EXAMINE, every fetched body section uses Peek; same rule after reconnect | Snapshot flags/content before/after list/search/unread/get/body/thread/metadata; pooled retries and concurrent access |
| Stale or ambiguous IDs route to wrong mail | 0003: ADR 0002 versioned references, validity check under selection lock, UID-based fetch | Recreate mailbox/reuse UID, stale reference on retry, colon/unicode names, overflow/empty fields; no fallback to old ID |
| Large mail/thread or hostile HTML exhausts memory | 0002 request cap; 0003 pre-fetch limits, bounded result/serialized output and deadline, plain text only | Oversized literals, headers, metadata, thread fanout, invalid/overflowing pagination; no remote HTML fetch |
| Secrets/private text in output or logs | 0002/0003: stable error codes, metadata-only logging, no raw request/provider-error logging | Canary token/password/subject/body absent from captured logs/errors, including failures |
| Host compromise or DMS impersonation | 0004: non-root/read-only container, private networks, verified certificate and secrets mounted read-only | Wrong CA/hostname fails; public port absent; writable-path restrictions and rollback verified |

Residual risks: a stolen instance token grants all configured accounts; trusted
Open WebUI/admin/model components see authorized mail; prompt injection can still
influence summaries. No per-user isolation or model correctness is promised.
Search responses may enumerate many matches server-side; hard execution/response
bounds and literal handling must be tested before calling the read path bounded.

## Status

Complete. No prerequisites. All acceptance criteria reviewed against the source
audit, threat model, accepted ADRs, upstream evidence, refined contracts and
actual validation above. Local Docker build/race/lint/integration checks and
exact-revision CI fuzzing passed. No unresolved baseline validation gaps remain.
Next: 0002, using only mocks or dedicated fake accounts until 0003 is complete.
