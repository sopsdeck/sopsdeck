# Changelog type tags and grouping

Type: build
Status: open
Blocked by: [22](22-epoch-semver-and-changelog.md)

## What to build

CHANGELOG / What’s new / site notes should group by change type (bug fix, feature, performance) with tags/badges, and show date, version, and platform (macOS, Windows, Linux) where we know it. Canonical source stays `CHANGELOG.md` ([22](22-epoch-semver-and-changelog.md)). Do not invent a second notes database.

Keep a Changelog already uses added/fixed/changed. Map those to tags rather than requiring Conventional Commit trailers in git.

## Acceptance criteria

- [ ] Generated What’s new and `site/changelog.html` show type tags/badges.
- [ ] Grouping is by version (existing) plus type; platform shown when the bullet names one.
- [ ] `CHANGELOG.md` remains the source; `./scripts/docs --check` still fails when generated pages are stale.

## Seams

- `./scripts/docs`, `CHANGELOG.md`, [22](22-epoch-semver-and-changelog.md), [26](26-visual-polish-and-changelog-look.md).

## Comments

Captured 2026-08-28 from human-found review. Kind: idea.
