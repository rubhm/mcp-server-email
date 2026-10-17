# Feature roadmap and execution order

Completed plans record implemented work and validation; incomplete plans describe
proposed behavior. HTTP transport is implemented, but read-only policy and P0
deployment remain pending.
Start with the first ready plan. Complete its acceptance criteria and record
actual validation evidence before starting implementation of a dependent plan.
Investigation may inform later drafts without authorizing their implementation.

| Order | Plan | Prerequisites | Status |
|---|---|---|---|
| 0001 | [Baseline and threat model](0001-baseline-and-threat-model.md) | None | Complete; baseline validation passed |
| 0002 | [Authenticated HTTP transport](0002-authenticated-http-transport.md) | 0001 complete | Complete; local validation passed |
| 0003 | [Read-only mail access](0003-read-only-mail-access.md) | 0002 complete | Ready for implementation |
| 0004 | [Unraid and Open WebUI deployment](0004-unraid-open-webui-deployment.md) | 0003 complete | Blocked by 0003 |

Numbers establish intended implementation order; prerequisites are completion
gates, not suggestions. If an earlier plan is blocked, resolve or explicitly
revise the dependency instead of skipping ahead. Keep this table and plan statuses
in sync. Do not renumber completed plans; record any authorized sequencing changes
in this index and the affected plans.

Transport and authentication are established before read-only enforcement.
Plan 0002 was validated with mocks and dedicated fake accounts; its completion
does not authorize real-mailbox deployment.
Plan 0003 must enforce and test read-only access over that transport before 0004.

## Objective and constraints

Provide local, authenticated, read-only access to authorized IMAP mailboxes from
Open WebUI through Streamable HTTP, retaining stdio compatibility. Existing
Open WebUI/LocalAI performs summarization; this server supplies bounded mail data
and enforces access restrictions. Preserve upstream attribution and existing tests.

P0 does not include mailbox mutations, attachment exports, SMTP delivery,
background monitoring, a new email UI, model inference, or Paperless/Odoo workflows.
Organization, drafts, approved sending, and any permanent deletion require future
numbered plans after P0; their mention in the source brief does not authorize them.

## Source and document ownership

The [original handoff](reference/handoff.md) preserves requirements and longer-term
intent. It is reference material, not an executable assignment or a competing
implementation sequence. Its phase numbering and PR suggestions are superseded
by the order above. Reconcile remaining ambiguities in 0001 before implementation.

Each numbered plan combines specification and delivery steps using AGENTS.md's
required sections. Keep shared constraints here and feature detail in its plan.
Significant accepted design decisions belong in [ADRs](../adr/template.md).
Update README and architecture documentation as features become implemented;
never present unchecked acceptance criteria as proof of current behavior.

## P0 release gate

All four plans must be complete, with validation evidence and unresolved gaps
reviewed, before calling P0 deployable. Compilation or unit tests alone do not
prove Open WebUI compatibility, DMS non-mutation, or deployment isolation.
