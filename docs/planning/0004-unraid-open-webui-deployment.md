# 0004: Unraid and Open WebUI deployment

## Purpose

Validate the complete read-only workflow on the intended private deployment and
document a reproducible installation and rollback path.

## Scope

Container hardening, Compose example, persistent secrets, certificate-verified
DMS connections, Open WebUI integration, end-to-end tests, and operating docs.
Use dedicated DMS test accounts; no production mailbox onboarding in this plan.

## Expected Behavior

Build on the completed [0002 transport](0002-authenticated-http-transport.md)
and [0003 read-only policy](0003-read-only-mail-access.md).
Run unprivileged with minimal capabilities, no unnecessary host port exposure,
and controlled persistent paths for configuration and any required writable
state. Use read-only root filesystem where compatible; specify exceptions.

Verify DMS TLS certificates against the correct internal hostname. Keep transport
credentials separate from IMAP credentials. Document the MCP network/TLS boundary
and enforce the chosen trust isolation. Supply no production secrets in examples.
Confirm Open WebUI's actual authentication and tool-calling behavior rather than
assuming UI account permissions propagate to MCP.

Using two authorized test mailboxes, demonstrate bounded last-day search/read
and attributable results that Open WebUI/LocalAI can summarize. Summary quality
depends on the client/model; the server must deliver correct source data and
must preserve flags and contents. Public endpoints and SMTP remain disabled.

## Acceptance Criteria

- [ ] Linux/amd64 image builds and starts with the documented container restrictions.
- [ ] Compose/configuration examples describe networks, secrets, state, resources, certificates, and the actual endpoint.
- [ ] Two dedicated DMS mailboxes connect with certificate verification; incorrect certificates fail.
- [ ] Open WebUI connects with authentication and completes the read-only scenario with accurate account/date/message references.
- [ ] DMS before/after assertions show list/search/read/unread/thread calls preserve flags and contents.
- [ ] Negative authentication/isolation checks pass; writes, SMTP, and filesystem exports remain unavailable.
- [ ] Logs and status do not leak credentials, private bodies, or unauthorized account details.
- [ ] Install, rollback, troubleshooting, and observed client compatibility are documented; all prior plan gates remain satisfied.

## Implementation Plan

1. Confirm 0003 completion; obtain target version/network/hostname information without requesting secrets in chat.
2. Add container configuration and dedicated DMS test harness/examples.
3. Validate container restrictions, TLS, client compatibility, and non-mutation with fake data.
4. Update maintained deployment documentation and README links; record exact validation evidence and gaps here.
5. Evaluate the roadmap's P0 release gate. Missing environment access leaves deployment criteria pending, not passed.

## Validation

Not run. Docker, Unraid, DMS, Open WebUI, and LocalAI versions and environment
access must be established before environment checks. Record tested revisions,
commands, outcomes, and redacted evidence. Do not infer deployability from docs.

## Status

Blocked by 0003. Environment validation pending. Completion is the P0 release
gate; later organization, drafts, and sending need separate numbered plans.
