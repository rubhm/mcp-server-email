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

## Read-only contract selected by 0001

Proposed behavior; implement only after 0002. Follow
[ADR 0001](../adr/0001-p0-trust-boundary.md) and
[ADR 0002](../adr/0002-p0-message-references.md). Add `EMAIL_ACCESS_MODE` with
`legacy` and `read-only` values. Stdio defaults to legacy; HTTP defaults to
read-only and rejects legacy. Reject explicitly invalid values. Request arguments
cannot change mode. Read-only startup validates unique nonempty account aliases,
valid configured default, credential/host fields, numeric limits and rejects
`insecure_skip_verify=true`. Use dedicated certificate-verified test credentials.

Exact P0 tool allowlist: `email_accounts`, `email_folders`, `email_list`,
`email_unread`, `email_search`, `email_get`, `email_read_body`, `email_thread`,
`email_attachment_list`. Deny all other names, including batch, send/reply/forward,
drafts, attachment get, exports and mailbox/flag mutations. `email://status`
remains authenticated and exposes only authorized aliases, connection booleans,
and version; omit email addresses, runtime, limits and default-account details.
Account discovery exposes aliases and default marker, not credentials/addresses.

Reuse Operations injection with a guarded adapter that checks the verified
principal, `mail.read`, account, mode, reference and limits before delegation.
All registered handlers/resources receive the guarded adapter. Preserve the
legacy adapter separately; never give read-only handlers a raw pool. Mutation
methods on the guarded adapter always return permission_denied before underlying
I/O. Do not construct an SMTP pool in read-only startup. A default account is
resolved only after principal verification and must belong to this instance.
Unknown accounts return account_not_found without provider details.

Implement validity-aware read request/results through the protocol boundary
(ADR 0002), not a context-free check in handlers. Every read selection uses
`SelectOptions{ReadOnly:true}`; every body/header section uses Peek. Use UID-based
search/fetch for matching/pagination to avoid sequence-number renumbering. Validate
UIDVALIDITY while holding the same lock as selection/fetch, including each retry.
Guard folder-role discovery and thread expansion under the same account policy.
Keep legacy behavior behind its adapter; shared correctness fixes must retain
legacy tests. No EXPUNGE/CLOSE-with-expunge on read-only connection cleanup.

### Bounds and result semantics

These are fixed P0 ceilings; permit no request or environment override above them.
Numeric inputs must be finite integers; reject negative, fractional, overflowing
or invalid values with invalid_argument before I/O. Preserve valid legacy behavior
in legacy mode. Responses include pagination/truncation metadata; never silently
represent partial results as complete.

| Surface | P0 contract |
|---|---|
| List/unread/search | Default 50, maximum 100 messages; offset default 0, maximum 10000. Newest UID first; offset applies to matching results. No snapshot across pages. |
| Accounts/folders | Maximum 100 configured accounts (startup), 200 returned folders per page; offset 0-10000. Bound folder enumeration to 10000 entries, otherwise return limit_exceeded. Role discovery uses bounded enumeration too. |
| Body | Absent by default, including email_get/thread. `includeBody=true` explicitly opts in; email_read_body is explicit. Plain text only; reject raw_html. Default page 10000 UTF-8 bytes, maximum 32768, offsets 0-2097152. Slice at valid UTF-8 boundaries and report actual offsets/next_offset in bytes. |
| Pre-fetch | Obtain RFC822.SIZE before body retrieval; reject body requests above 2 MiB and cap literal collection at 2 MiB even if advertised size is smaller. Metadata listing remains available. Header section at most 128 KiB, with bounded literal handling before Collect allocates. Avoid a post-allocation-only check. |
| Metadata | Subject/address/filename fields maximum 4096 UTF-8 bytes each with explicit truncation markers; at most 100 recipients, 100 attachments, 100 thread references per message. Validate locator/header input lengths separately. |
| Thread | Maximum 100 messages, 5 folders, 20 distinct Message-ID searches, and 30s total including retries. Mark incomplete with reason on limits or failed folder/search subcalls; do not silently discard failures. |
| Entire response | Maximum 1 MiB serialized MCP response including escaped text/envelope; trim only at record/UTF-8 boundaries and mark incomplete, or return limit_exceeded if even one record cannot fit. Bound serialization before sending. |
| Input text | Query/from/to at most 4096 bytes each; folder at most 1024 bytes; references follow ADR 0002. |

HTML is parsed to plain text without execution, resource fetching or following
links; never render email text as trusted instructions. Bound input before parsing.
Only attachment metadata is available. Record which limits restricted the output.
Pagination outside available results returns an empty collection or empty body
page; no wraparound. Prevent offset+limit overflow before slicing.

Search dates retain IMAP calendar-day semantics: `since` inclusive and `before`
exclusive, strict YYYY-MM-DD validation, and before must follow since. No implicit
lookback restriction; count/time limits still apply. Results use timezone-aware
RFC3339 timestamps. The last-day deployment scenario uses a documented date
window; do not promise an exact rolling 24-hour filter from calendar dates.

Use stable safe codes: invalid_argument, permission_denied, account_not_found,
stale_reference, message_not_found, limit_exceeded, timeout and upstream_error.
Do not return raw provider errors, credentials, message bodies or filesystem paths
inside failures. Capture canary secrets/private text in tests to verify redaction.

### Required regression cases

- Connector tests demonstrate EXAMINE and Peek on every body section, including
  list/unread/search with includeBody, get/body/thread and attachment metadata.
- Spy Operations verify denied direct handlers/methods perform no SMTP, mutation
  or filesystem I/O; HTTP discovery/calls/resources exercise the same adapter.
- Reference tests cover empty fields, uint overflow (including uint64 wrap),
  colons/Unicode, recreated mailbox/reused UID, retries and pooled concurrency.
- Limits cover huge literals with dishonest size declarations, large metadata,
  escaped JSON overhead, negative/huge/fractional pagination, empty results,
  timeout cancellation, thread errors/fanout and valid UTF-8 page boundaries.
- Greenmail fixtures snapshot UIDVALIDITY, UID sets, flags, contents before/after
  every allowed read path. Use a separate test fixture connection for writes;
  preserve existing legacy SMTP integration coverage. DMS repetition is in 0004.

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

Ready for implementation; prerequisite 0002 is complete. Authenticated HTTP and
instance context are available, but currently expose legacy operations. Numeric
limits, configuration and reference contracts are specified above; implementation
and execution evidence remain pending.
Next: 0004 after completion.
