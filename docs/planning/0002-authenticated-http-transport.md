# 0002: Authenticated HTTP transport

## Purpose

Establish authenticated MCP connectivity over
native Streamable HTTP while retaining stdio compatibility.

## Scope

Transport selection/configuration, authentication, request identity, HTTP
lifecycle, minimal health response, and transport-level security tests.
Read-only enforcement follows in 0003. Validate this transport with mocks or
dedicated fake test accounts; do not connect real mailboxes or deploy for use.

## Expected Behavior

Use the library and identity decisions verified in
[0001](0001-baseline-and-threat-model.md). Preserve authenticated context for
the policy integration in [0003](0003-read-only-mail-access.md).
The transport contract below specifies configuration names,
defaults, bind address, port, endpoint path, token sourcing, request/session
handling, and shutdown timeouts. Invalid or incomplete security configuration
must fail closed. Validate the target client's actual MCP handshake and requests.

Authenticate all MCP entry points, including discovery and resource access;
never trust arbitrary identity headers. If using a shared service credential,
do not treat it as per-user identity. Address origin/host validation, credential
redaction, request limits, and session isolation in the transport threat model.
Health output must not expose accounts or credentials. Shutdown must drain or
cancel requests and close pools predictably. Keep stdout protocol-safe in stdio.

## Transport contract selected by 0001

Implemented after 0001 completion. Uses pinned
mcp-go v0.43.2's `NewStreamableHTTPServer` as a handler on an owned `http.Server`,
with `WithStateLess(true)` and `WithDisableStreaming(true)`. Register exactly
`/mcp`, no redirect/trailing-slash aliases. Endpoint path is fixed for P0.
Use the original ServeStdio path for stdio. No dependency upgrade is planned.

| Configuration | Contract |
|---|---|
| `EMAIL_TRANSPORT` | `stdio` default; `http` opt-in; reject unknown/empty explicit values. |
| `EMAIL_HTTP_ADDR` | Default `127.0.0.1:8080`; explicit `0.0.0.0:8080` for the private container network. Validate address before listening. |
| `EMAIL_HTTP_TOKEN_FILE` | Required in HTTP; no inline env-token alternative. Read at startup, at most 4096 bytes; strip one terminal LF/CRLF, then require 32-256 printable ASCII non-whitespace bytes. Provision at least 32 random bytes encoded as base64url. Missing/unreadable/invalid secret fails startup without its contents. |
| `EMAIL_HTTP_ALLOWED_HOSTS` | Comma-separated exact authorities including port; no wildcard. Defaults to `localhost:8080,127.0.0.1:8080`. Non-default listener/client authorities require explicit entries. |
| Request/HTTP limits | 1 MiB request body, 16 KiB headers, 5s ReadHeaderTimeout, 15s ReadTimeout, 60s WriteTimeout, 60s IdleTimeout; 30s overall MCP request context including retries. |
| Shutdown | Stop admission on signal, cancel in-flight contexts, 5s HTTP shutdown then force-close if needed; close IMAP/SMTP pools using existing configured pool timeout. Wait for cleanup; no os.Exit in signal goroutine. |

Reject invalid HTTP configuration before constructing pools/listeners; HTTP-only
settings do not affect stdio. Authentication wraps the entire MCP handler before
body parsing or dispatch, including GET, DELETE, discovery and resource requests.
Require exactly one Authorization header with a Bearer token; compare using
constant-time comparison. Missing/invalid credentials return 401 with a Bearer
challenge. Never put tokens in URLs. Never log headers, request bodies, query
strings or raw authentication errors. Ignore forwarded/user identity headers.
Attach the fixed instance principal from ADR 0001 using a private context key;
handlers must not derive identity from arguments or caller-supplied context keys.

Validate exact Host after normalization of DNS case; reject any Origin header,
including `null`, with 403; no CORS support. Backend Open WebUI calls are the
target. Do not trust X-Forwarded-Host. Plain HTTP is restricted to loopback or a
trusted private Docker bridge; a verified TLS reverse proxy is required across
other networks. Network placement is verified in 0004, not inferred by this code.

POST initialize/initialized/ping/list/call/resource requests use the native
transport. Valid authenticated GET returns 405 with streaming disabled; DELETE
uses the library's stateless termination behavior, verified in tests. Session
headers grant no access and do not change the principal. No sampling, elicitation,
session-local tools, or persistent server notification channel in P0. Set static
tool/resource capabilities consistently with this choice.

`GET /healthz` may be unauthenticated and returns only `{"status":"ok"}`; it
does not connect to mail servers. Other health methods return 405. Health shares
Host/Origin validation. Unknown paths return 404. Oversized requests return 413
before library io.ReadAll; malformed requests return bounded errors without raw
payloads. Use metadata-only logs and stable error codes for application failures.
HTTP authentication alone does not impose read-only behavior in this intermediate
plan; test only with mocks/dedicated fake accounts until 0003 completes.

Compatibility target is latest stable Open WebUI, resolved during 0001 to
v0.12.3 on 2026-10-17. Use native MCP connection type with Bearer/Key. Add a
version-pinned client test covering initialize, tool pagination/call, resources,
GET 405 handling, reconnect and shutdown. Exercise the mcp-go client independently.
Source-level support is recorded in 0001; actual client execution remains a gate.

## Acceptance Criteria

- [x] Configuration and legacy stdio behavior are explicit and covered by startup tests.
- [x] Missing/invalid credentials cannot initialize, call tools, list accounts, or access resources; valid requests carry the authenticated context required by 0003.
- [x] Session, origin/host, request-limit, and spoofed-identity cases have tests aligned with the selected library mode.
- [x] Health output is minimal; logs/errors redact tokens and email content.
- [x] Native Streamable HTTP handshake and relevant request methods are tested with the selected library; stdio regression tests pass.
- [x] Graceful shutdown and invalid startup configuration are tested.
- [x] Required checks and maintained documentation reflect the implemented transport.

## Implementation Plan

1. Confirmed 0001 completion and pinned library APIs before implementation.
2. Added configuration, authentication-before-dispatch, protocol, limit, isolation,
   error/log canary, deadline, shutdown and executable startup/stdio tests.
3. Implemented `internal/transport` and wired main to validated startup, an owned
   signal context, stateless HTTP serving, bounded shutdown and deferred pool close.
4. Tested native Go and pinned Open WebUI clients; updated README and architecture.
5. Recorded evidence below and reviewed every acceptance criterion. 0003 is ready;
   no production mailbox deployment or read-only enforcement occurred in this plan.

## Validation

Executed 2026-10-17 against the working tree based on
`3b407e43ac3803553688fbce9e4826c2c40e8c8f`. No new-change CI result is claimed.
Linux/amd64 validation used Docker Desktop with Go 1.24.13/GCC 12 and source
mounted read-only for tests, UID/GID 10001:10001. Cache and Git mount handling
follow [0001's container reproduction](0001-baseline-and-threat-model.md#container-reproduction).

| Check | Actual result |
|---|---|
| `go build ./...` | Passed. |
| `go test -race -count=1 ./...` | Passed across entrypoint and all ten internal packages, including transport. |
| `golangci-lint run ./...` | Passed, v2.11.3, `0 issues.` |
| `go test -tags=integration -race -count=1 ./...` | Passed against Greenmail 2.1.0 on a dedicated private bridge; explicit fake-account test variables supplied, no host ports. IMAP 13.810s, SMTP 2.281s. |
| Native mcp-go v0.43.2 client | Passed initialize, multi-page discovery (including terminal empty page), calls, resources, principal propagation and provider-error redaction in transport tests. |
| Executable stdio regression | Passed initialize, tool discovery/account call, status resource and signal shutdown; invalid startup fails without protocol stdout or secret-path disclosure. HTTP-only variables do not affect default stdio. |
| Open WebUI client | `scripts/test-openwebui-client.py` passed against a built executable with only a fake configured account and temporary random bearer secret. Python 3.12, mcp 1.27.2, httpx 0.28.1; tagged v0.12.3 client SHA256 `3969ca30bd5a4a3a674a6582d5120790efe0824d5fbdcad1ee71c273aa3ec5eb`. |
| Documentation | Local file links, required plan sections and completed acceptance checklist passed; `git diff --check` passed. |

The Python harness imports the actual checksum-verified client module; it supplies
only environment constants and a JSON codec import, rather than starting the
entire WebUI application. It verifies all 22 tools over ten-entry pages, account
call, resources, missing/wrong authentication, GET 405, stateless DELETE 200,
reconnect, clean SIGTERM exit, empty HTTP stdout and log canaries. This establishes
client-module protocol compatibility, not full UI authorization, model behavior,
TLS routing or deployment isolation. Those remain 0004 acceptance gates.

To reproduce the client check, build `bin/http-client-server` from
`./cmd/mcp-server-email`, install `mcp==1.27.2 httpx==0.28.1` into an isolated
Python 3.12 environment, then run:

```sh
python scripts/test-openwebui-client.py --server-binary bin/http-client-server
```

The harness downloads only the tagged client source, checks its checksum, and
creates/removes its own temporary fake configuration and secret. The new CI job
runs the same check; its execution is pending the next push. Existing unit/fuzz
and Greenmail jobs are retained. No dependency versions were changed in go.mod.

HTTP uses metadata-only JSON info logs; stdio retains existing log configuration.
Library diagnostic output is discarded rather than logging arbitrary request
data. Middleware redacts HTTP tool/resource errors and panics, and the buffered
non-streaming response scrubs RPC error details. Successful authorized mail
content remains visible. Response-size/body-fetch bounds remain work for 0003;
this plan enforces the request boundary only.

## Status

Complete. Prerequisite 0001 is complete. Acceptance criteria are satisfied by the
implementation, tests, local validation and maintained documentation above.
Next: 0003, enforcing read-only mail access. HTTP currently registers legacy
mutating/export/SMTP tools; use mocks or dedicated fake accounts only. P0 is not
deployable until 0003 and 0004 also complete.
