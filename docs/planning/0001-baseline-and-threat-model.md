# 0001: Baseline and threat model

## Purpose

Establish evidence about the existing fork and resolve security and compatibility
questions before implementing the [P0 roadmap](README.md).

## Scope

Inspect the fork, dependencies, CI, IMAP reads, tool/resource exposure,
configuration, logging, and deployment assumptions. Run baseline checks where
tooling permits. No production mailbox access or feature implementation.

## Expected Behavior

Keep confirmed current behavior separate from handoff claims and proposed work.
The inspected entrypoint serves stdio and calls `tools.RegisterAll`; the status
resource includes account data. Reads select mailboxes with nil options, while
the shared body-fetch helper already sets `Peek: true`. These observations require
tests and a complete read-path audit before any non-mutation guarantee.

Resolve these prerequisites for later specifications:

- P0 trust boundary: proposed separate deployment/credential per group of callers
  authorized for every configured account. Do not assume a shared bearer token
  identifies individual Open WebUI users. Record the chosen model and alternatives
  in an ADR; mixed-permission shared access needs explicit authorization design.
- Message references: decide whether to implement UIDVALIDITY-aware references in
  P0 or narrow the reference contract with an explicit stale-reference limitation.
  Do not promise stable references without evidence or silently permit misrouting.
- Verify the pinned MCP library's transport/authentication hooks and the target
  Open WebUI integration using authoritative version-specific evidence.
- Define the read-only tool allowlist, resource visibility, permission vocabulary,
  response/body limits, and compatibility expectations for existing stdio clients.
- Model hostile email content, cross-account access, credential leakage, arbitrary
  filesystem access, mutations, unintended SMTP, and private-data logging.

## Acceptance Criteria

- [ ] Fork ancestry/upstream commit, dependency versions, and baseline commands/results are recorded here without secrets.
- [ ] Build, race tests, lint, and relevant fuzz/integration coverage have recorded outcomes; unavailable checks remain explicit gaps.
- [ ] Relevant code paths and existing protections are identified with file/function references.
- [ ] The trust-boundary and reference-contract decisions are documented, with accepted significant decisions in ADRs.
- [ ] The threat model has mitigations and planned negative tests linked to 0002/0003.
- [ ] Plans 0002-0004 are refined with concrete configuration, limits, interfaces, and testable contracts; no prerequisite design question is silently assumed resolved.

## Implementation Plan

1. Establish tooling/CI access and inspect repository provenance and configuration.
2. Run the checks required by AGENTS.md; inspect read operations and existing tests.
3. Record evidence, threats, decisions, and unresolved gaps in this plan and ADRs.
4. Refine dependent plans; check these acceptance criteria before marking complete.

## Validation

Application checks have not been executed for this plan. Previous shell inspection
did not find Go, golangci-lint, or Docker on PATH; recheck availability before
running them or use verified CI results for the exact revision. Greenmail tests
and later DMS tests must use dedicated test data. A tool/config definition is not
a passing validation result. Record command, revision, outcome, and limitations.

## Status

Ready for investigation. No prerequisites. Baseline audit, design decisions,
and application validation are pending. Next: complete this plan before 0002.
