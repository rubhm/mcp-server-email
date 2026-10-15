# Agent Instructions

## Development Workflow

- Inspect relevant code, project structure, and documentation before making changes.
- Follow existing architecture, conventions, and patterns. Prefer reusing existing functionality over introducing new abstractions or dependencies.
- For complex tasks, create a concise plan before implementing.
- Keep changes focused on the requested task. Avoid unrelated refactoring.
- For Go code changes, run `go build ./...`, `go test -race -count=1 ./...`, and `golangci-lint run ./...`.
- Run relevant integration tests for changes affecting IMAP/SMTP behavior. These require Docker/Greenmail and the `integration` build tag; follow the [integration test guide](README.md#integration-tests).
- Never claim validation succeeded unless it was actually performed.
- Summarize changes, verification results, and unresolved issues upon completion.

## Change Planning

- Review relevant plans in `docs/planning/` before starting related work.
- For complex or multi-step changes, create or update a concise plan in `docs/planning/` before implementing.
- Include scope, implementation steps, validation, and completion criteria.
- Update the plan as work progresses, recording completed steps and unresolved issues.
- Keep simple changes lightweight; do not create a plan for routine edits.
- Record significant architectural decisions separately in `docs/adr/`.

## Documentation

- Keep `README.md`, `AGENTS.md`, and `CONTRIBUTING.md` in the repository root.
- Store all other project documentation in `docs/`, except files that tools or frameworks require elsewhere.
- Update relevant documentation when implementation, configuration, or architecture changes.
- Prefer updating existing documentation over creating new files.
- Keep documentation concise, accurate, and avoid duplication.

## Architecture Decision Records

- Store ADRs in `docs/adr/`.
- Use sequential filenames: `0001-short-description.md`.
- Each ADR should include: Status, Context, Decision, Consequences, and Alternatives Considered.
- Review relevant ADRs before making architectural changes.
- Create ADRs only for significant decisions with long-term implications, not routine implementation details.
- Never silently contradict an accepted ADR. When replacing a decision, create a new ADR and mark the previous one as superseded.

## Persistent Project Knowledge

- Treat `docs/` and existing ADRs as shared project knowledge across agent sessions.
- Read relevant documentation before starting work; do not read everything unnecessarily.
- Document important discoveries, constraints, and non-obvious behavior that future agents would benefit from.
- Keep documentation synchronized with actual code. Avoid temporary notes and session logs.

## Safety and Scope

- Never commit secrets or credentials.
- Do not overwrite unrelated user changes.
- Avoid destructive operations without explicit authorization.
- Ask for clarification only when ambiguity materially affects the implementation.
- Use relevant project skills from `.agents/skills/` when applicable.
