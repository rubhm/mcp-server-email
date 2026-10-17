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

## Deployment contract selected by 0001

Follow ADR 0001: one container/configuration/token per group authorized for all
its accounts; test Open WebUI connection access controls with allowed and denied
users. Never infer per-user authorization from a shared bearer token. Compose uses
`EMAIL_TRANSPORT=http`, `EMAIL_ACCESS_MODE=read-only`,
`EMAIL_HTTP_ADDR=0.0.0.0:8080`, `EMAIL_HTTP_TOKEN_FILE=/run/secrets/mcp-token`,
`EMAIL_CONFIG_FILE=/run/secrets/accounts.json`, and an explicit allowed Host such
as `email-mcp:8080`. Open WebUI connects to `http://email-mcp:8080/mcp` on the
trusted bridge with Bearer/Key. Do not publish host ports. Use verified TLS at a
proxy when crossing the bridge boundary; the backend Host must match the list.

Pin the newest stable Open WebUI release at validation time and record tag/digest.
0001 resolved latest to v0.12.3 on 2026-10-17; recheck if validation occurs later.
Never use a floating latest tag as evidence. Record Unraid, Docker, DMS/Dovecot,
LocalAI/model and proxy versions alongside tested client version and image digest.

Container target: Linux/amd64, UID/GID 10001:10001, read-only root filesystem,
drop ALL capabilities, no-new-privileges, pids_limit 128, mem_limit 256m, cpus 1.0.
Password-based P0 uses read-only config/token mounts, system/custom CA bundle,
no writable host bind mount and only a size-limited /tmp tmpfs if runtime checks
show it is necessary. OAuth token refresh needs a separately scoped writable
token store if enabled; do not claim a wholly read-only OAuth filesystem. Verify
resource ceilings against large-message/thread failure tests before finalizing.

Use DMS internal DNS name matching its certificate, implicit IMAPS 993, trusted
CA bundle, and skip-verify false. SMTP pool must be absent in read-only mode;
egress rules deny SMTP ports. Keep DMS reachable on a private mail network and
restrict other egress where feasible. Token is distinct from IMAP passwords and
provisioned out of band. Document owner/mode/rotation without printing contents.

End-to-end evidence must include initialize/discovery/tool calls, bounded last-day
search/read from two dedicated accounts, exact account/reference/date attribution,
and unchanged mailbox snapshots for list/unread/search/get/body/thread/metadata.
Exercise permitted and denied Open WebUI users, wrong/missing bearer, forged
account/reference, foreign Host/Origin, rejected writes/exports, and wrong DMS
certificate/hostname. Record client GET/DELETE/reconnect behavior in stateless
mode. Summary accuracy is assessed separately from server source-data accuracy.

Rollback restores the previous pinned image/config, removes connection access
before replacement, and rotates the bearer credential if exposure is suspected.
No production onboarding until all P0 acceptance criteria pass. Missing Unraid,
DMS or client access leaves this plan incomplete; mocked results cannot close it.

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
