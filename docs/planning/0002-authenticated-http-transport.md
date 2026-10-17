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
Before coding, specify configuration names,
defaults, bind address, port, endpoint path, token sourcing, request/session
handling, and shutdown timeouts. Invalid or incomplete security configuration
must fail closed. Validate the target client's actual MCP handshake and requests.

Authenticate all MCP entry points, including discovery and resource access;
never trust arbitrary identity headers. If using a shared service credential,
do not treat it as per-user identity. Address origin/host validation, credential
redaction, request limits, and session isolation in the transport threat model.
Health output must not expose accounts or credentials. Shutdown must drain or
cancel requests and close pools predictably. Keep stdout protocol-safe in stdio.

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

Blocked by 0001. Configuration/library/client details await 0001 investigation.
Next: 0003 after completion. Transport completion alone is not P0 readiness.
