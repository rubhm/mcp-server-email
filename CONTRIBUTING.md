# Contributing

Thanks for your interest in contributing to mcp-server-email.

## Reference Documentation

Use these documents for their respective topics rather than maintaining copies here:

- [README development guide](README.md#development): prerequisites, build and test commands, benchmarks, fuzzing, integration setup, troubleshooting, and coverage reports.
- [Architecture](docs/ARCHITECTURE.md): package responsibilities, data flow, and design decisions.
- [Agent instructions](AGENTS.md): agent workflow, change planning, documentation organization, ADRs, and safety rules.
- [Lint configuration](.golangci.yml): enabled checks, formatting rules, and documented exceptions.
- [CI workflow](.github/workflows/ci.yml): CI jobs and the Go and linter versions used for validation.

## Development Workflow

1. Fork and clone the repository you intend to contribute to.
2. Create a feature branch from `main`.
3. Make focused changes and add or update tests for changed behavior.
4. Before committing, run the build, unit tests with the race detector, and lint checks described in the [development guide](README.md#development). Run relevant integration or fuzz tests when the change warrants them.
5. Commit using the convention below.
6. Open a pull request against the target repository's `main` branch.

Run lint early while developing. Document new public functions with GoDoc comments, and use `t.Helper()` for extracted test helpers.

## Releases

Pushing a `v*` tag runs the [release workflow](.github/workflows/release.yml).
Homebrew publishing requires the `HOMEBREW_TAP_GITHUB_TOKEN` repository secret
with write access to the tap configured in `.goreleaser.yml`. When the secret is
unset or empty, the workflow skips Homebrew and continues with the other release
targets.

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```text
type: short description

Longer explanation if needed.
```

Types: `feat`, `fix`, `refactor`, `test`, `chore`, `docs`, `ci`.

Examples:

- `feat: add email_thread tool for cross-folder thread search`
- `fix: handle nil attachment in forward handler`
- `test: add coverage for SMTP OAuth2 retry path`
- `docs: clarify integration test setup`

## Pull Requests

Describe the problem, resulting behavior, and validation performed. Include known limitations and explain any checks that could not run. Use the [coverage instructions](README.md#coverage) to assess whether changed behavior is adequately tested; historical coverage percentages and test-to-source ratios are not acceptance criteria.

Before requesting review, confirm that the development workflow above is complete and that fixtures contain only fake credentials and test data.

## Reporting Issues

Open an issue in the target repository's GitHub Issues tab with:

- Steps to reproduce.
- Expected and actual behavior.
- Server version (`go version -m /path/to/mcp-server-email`).
- Mail provider, if relevant.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
