# Deterministic quality gates: markdown, security, hooks

Type: build
Status: ready
Blocked by: None

## What to build

Human QA wants more **deterministic** checks we will actually remember:

- Markdown lint and format.
- A code/security scanner that is local and repeatable (SonarQube was an example — prefer something that runs in `./scripts/check` or a dedicated script without a hosted dashboard, unless we later add CI-only Sonar).
- Enforcement that changelog (and other user-facing docs) stay updated with human-readable notes, plus tests/lint/format — commit hook and/or CI.

Do not weaken `./scripts/check`. Do not invent changelog *product* wording; [22](22-epoch-semver-and-changelog.md) already requires Keep a Changelog in `CHANGELOG.md`. A hook may remind or fail when `CHANGELOG.md` is unchanged on a user-facing slice — keep the rule mechanical (file touched vs not), not an LLM judge.

`--no-verify` must still work (issue 10 / git). Hooks are opt-in via a setup script if that is the repo pattern.

## Acceptance criteria

- [ ] Markdown in `docs/` (and agreed paths) is linted/formatted; `./scripts/fmt` / `./scripts/check` cover it.
- [ ] A documented local security/static check exists (e.g. gosec/clippy already, plus an extra JS/Go scanner if we add one) and is deterministic.
- [ ] Pre-commit or CI fails on fmt/lint/test the way `./scripts/check` does today; changelog rule is explicit and mechanical.
- [ ] `./scripts/check` stays the fast default; heavy scanners may be a separate script called from CI.

## Seams

- `scripts/check`, `scripts/fmt`, optional git hooks, CI workflow.

## Comments

Captured 2026-08-28 from human-found review. Kind: idea. Not a substitute for issue 16 threat model.
