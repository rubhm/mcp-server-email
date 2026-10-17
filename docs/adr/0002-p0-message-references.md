# 0002: Require UIDVALIDITY-aware references in P0

## Status

Accepted for planned P0 implementation; not implemented. Prerequisite:
[ADR 0001](0001-p0-trust-boundary.md).

## Context

The legacy `account:mailbox:uid` format cannot detect mailbox recreation and its
parser cannot handle mailbox colons. Retrying a stale reference must not return
an unrelated message that reused a UID. Existing stdio clients need compatibility.

## Decision

Read-only mode emits and accepts only `v1.` followed by unpadded base64url of a
UTF-8 JSON array `[account,mailbox,uidvalidity,uid]`. Account and mailbox must be
nonempty strings; validity and UID must be positive uint32 JSON integers. Reject
extra fields, invalid encoding, overflow, fractional numbers, invalid UTF-8,
decoded payloads over 4096 bytes and encoded references over 5500 bytes.
Limit account aliases to 128 UTF-8 bytes and mailbox names to 1024 bytes.
Colons and Unicode in mailbox names round-trip without delimiter ambiguity.

References are locators, not credentials. Resolve/authorize the account before
network I/O. Under the connection lock, select with EXAMINE, compare UIDVALIDITY
and fetch by UID; mismatch returns `stale_reference` without fetching content.
Revalidate after every reconnect/retry. Never reconstruct validity from the
current mailbox for a supplied old reference. Absence of a matching UID returns
`message_not_found`. Mailbox rename/move invalidates its old locator.

Extend the existing read-operation seam with validity-aware request/result data
so validation reaches the protocol boundary; handler-only checks are insufficient.
List/search/unread/thread results capture validity from the same selected mailbox
as their UID fetches. Use UID search/fetch rather than persisting sequence numbers
across commands. Per-request mutation guards remain required independently.

Legacy stdio retains its current format through its legacy adapter. Do not let
P0 accept a legacy ID or fall back to legacy parsing. References do not promise a
snapshot of a mailbox across pages or identity across mailbox renames.

## Consequences

P0 pays a small reference-size cost and needs interface/test changes in 0003.
It rejects ambiguous/stale locators instead of silently misrouting them. Clients
must relist after a stale-reference error; old stdio IDs cannot migrate directly.

## Alternatives Considered

- Document legacy IDs as temporary: cannot detect silent UID reuse even shortly
  after issuance.
- In-memory opaque IDs: need storage, expiry and restart semantics.
- Message-ID headers: neither unique nor mandatory; unsuitable as IMAP locators.
