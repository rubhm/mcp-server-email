# 0001: Isolate P0 by shared account authorization

## Status

Accepted for planned P0 implementation; not implemented. See [baseline](../planning/0001-baseline-and-threat-model.md).

## Context

P0 needs authenticated Open WebUI access without a verified per-user identity
protocol. A connection bearer credential authenticates the connection, and
cannot distinguish people sharing it. Current configuration exposes every account.

## Decision

Use a separate MCP process/container, configuration, secret, and network route for
each group of callers authorized for every account configured in that instance.
Never place differently authorized accounts behind one shared credential.
Open WebUI access controls must restrict the connection to that group; deployment
testing must verify this rather than infer it from a server token.

Authenticate every HTTP MCP request with a dedicated random bearer secret sourced
from a mounted file. Attach a fixed instance principal to verified request context.
Do not trust user IDs, forwarded headers or Open WebUI session tokens as identity.
P0 uses stateless HTTP with no persistent notification stream or server sampling.
Grant only `mail.read`; reserve `mail.organize`, `mail.draft`, `mail.send`, and
`mail.delete` for later plans. Guard operations as well as tool registration.

After 0003, HTTP defaults to read-only and refuses legacy access mode. Stdio keeps
legacy mode by default for existing local clients; `EMAIL_ACCESS_MODE=read-only`
opts stdio into the same policy, limits, and references as HTTP. Stdio is a local
process trust boundary with an explicit local principal. Neither mode can be
changed by request parameters. Plan 0002's intermediate transport is test-only
until 0003 enforces this policy.

## Consequences

This avoids building an identity service for P0 and gives account isolation a
testable deployment boundary. More containers and credentials are required.
A stolen bearer secret exposes all accounts in its instance; rotation requires
restarting the instance and updating its client. Use TLS across networks outside
the trusted private bridge. A shared Open WebUI admin or host compromise is
outside this isolation guarantee. Legacy stdio retains broad existing powers.

## Alternatives Considered

- Shared token plus caller user headers: forgeable without a verified issuer.
- One shared instance with per-user OAuth: valid future direction, but needs
  issuer/audience validation, account mapping and separate authorization tests.
- Relying on UI-hidden tools or model prompts: does not enforce server policy.
