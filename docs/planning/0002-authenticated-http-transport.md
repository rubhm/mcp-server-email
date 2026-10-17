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

Proposed behavior; 0001 is complete and implementation may begin. Use pinned
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

- [ ] Configuration and legacy stdio behavior are explicit and covered by startup tests.
- [ ] Missing/invalid credentials cannot initialize, call tools, list accounts, or access resources; valid requests carry the authenticated context required by 0003.
- [ ] Session, origin/host, request-limit, and spoofed-identity cases have tests aligned with the selected library mode.
- [ ] Health output is minimal; logs/errors redact tokens and email content.
- [ ] Native Streamable HTTP handshake and relevant request methods are tested with the selected library; stdio regression tests pass.
- [ ] Graceful shutdown and invalid startup configuration are tested.
- [ ] Required checks and maintained documentation reflect the implemented transport.

## Implementation Plan

1. Confirm 0001 is complete; finalize transport contracts and testable error cases.
2. Add transport/startup/authentication tests before or alongside implementation.
3. Wire library-supported HTTP serving, authentication, authenticated context propagation, and shutdown.
4. Exercise protocol requests and stdio regression checks; update configuration and architecture documentation.
5. Record acceptance evidence before unblocking 0003. Document the pending read-only restriction and do not deploy to real mailboxes as part of this plan.

## Validation

Not run. Required Go checks, HTTP protocol tests, and applicable integration tests
are pending. Open WebUI/DMS environment validation is a separate 0004 release gate.

## Status

Ready for implementation; prerequisite 0001 is complete. Configuration/library/
client contracts are recorded above. HTTP implementation and client handshake
evidence remain pending; baseline checks do not establish transport behavior.
Next: 0003 after completion. Transport completion alone is not P0 readiness.
