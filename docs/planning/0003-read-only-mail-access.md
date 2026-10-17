# 0003: Read-only mail access

## Purpose

Enforce P0 read-only access in the server and over the authenticated HTTP
transport before deployment to real mailboxes.

## Scope

Tool registration, guarded mail operations, account/resource visibility,
non-marking reads, bounded output, and the reference contract selected in 0001.
SMTP, mutations, and attachment downloads/exports remain unavailable in P0.

## Expected Behavior

Use the trust boundary and compatibility contract resolved in
[0001](0001-baseline-and-threat-model.md). Apply restrictions server-side, not
through tool descriptions or model prompts. Explicitly configure read-only
operation; decide defaults and legacy stdio behavior in 0001 before coding.
Integrate the authenticated context from
[0002](0002-authenticated-http-transport.md) into policy enforcement.

Define an exact allowlist for accounts, folders, message list/search/unread,
single-message/body reads, thread reads, and attachment metadata where safe.
Guard underlying operation interfaces so direct handlers cannot bypass policy.
Reads must preserve flags and contents, including unread and thread paths; use
read-only selection and non-marking fetch where applicable. Test pooled connection
reuse and retries. Apply account restrictions to resources and encoded IDs too.

Define concrete response, body, count, pagination, and timeout limits before
implementation. Treat email/HTML as untrusted data; do not execute HTML or fetch
remote content. Deny unknown accounts and malformed/stale references according to
the selected contract; return consistent errors without exposing private data.

## Acceptance Criteria

- [ ] Exact tool allowlist and configuration/defaults are documented and tested.
- [ ] Mutation and attachment export calls are denied at registration and guarded operation boundaries; tests verify no SMTP or write call occurs.
- [ ] Account discovery, status resources, and message IDs cannot reveal or access accounts outside the selected trust boundary.
- [ ] Authenticated HTTP calls, discovery, and resource access enforce the same read-only policy as guarded direct calls; transport authentication cannot bypass it.
- [ ] List, unread, search, message/body, thread, and metadata reads preserve mailbox state in protocol integration tests.
- [ ] Limits, malformed inputs, empty results, stale references, retries, and concurrent access have relevant tests.
- [ ] Stdio behavior follows the documented compatibility decision; maintained documentation explains actual behavior.
- [ ] Required Go checks and relevant integration checks have recorded results with remaining gaps disclosed.

## Implementation Plan

1. After 0002 completes, use 0001's decisions to finalize allowlist/configuration contracts and transport integration test cases.
2. Add meaningful negative tests and non-mutation protocol tests.
3. Implement policy/operation guards, safe reads, bounded results, and resource filtering using existing interfaces.
4. Validate with mocks and dedicated Greenmail mailboxes; record any DMS-specific verification still required by 0004.
5. Update maintained docs and assess acceptance criteria before unblocking 0004.

## Validation

Not run. Use AGENTS.md's Go checks and README's integration guide. Record actual
results here when implemented. DMS behavior must also pass the 0004 deployment
gate; Greenmail results alone do not establish DMS behavior.

## Status

Blocked by 0002. This is a draft specification; numeric limits and configuration
contracts must be resolved before implementation. Next: 0004 after completion.
