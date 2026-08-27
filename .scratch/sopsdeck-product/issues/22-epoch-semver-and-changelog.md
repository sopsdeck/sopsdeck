# Epoch SemVer, CHANGELOG.md, and release notes surfaces

Type: build
Status: ready
Blocked by: None

## What to build

Refine [Define packaging, updates, and platform support](14-define-release-and-support-contract.md) (“versioning is semver”) without replacing it.

**Versioning** is [Epoch Semantic Versioning](https://antfu.me/posts/epoch-semver): still `MAJOR.MINOR.PATCH` for GitHub, Tauri, Homebrew, and the updater. Interpret the first number as `EPOCH * 1000 + MAJOR`.

- **EPOCH** — named product era (rare). `1000.0.0` is a marketing/overhaul event, not “we broke one CLI flag.”
- **MAJOR** — incompatible behavior users must notice (0–999 inside an epoch).
- **MINOR** — compatible features.
- **PATCH** — compatible fixes.

Until a named era, EPOCH stays **0**, so versions look like ordinary 1.2.3. The unreleased tree may keep `0.1.0` in Tauri; the **first GitHub Release** is `1.0.0` (epoch 0) when phase 8 ships. Do not stay on leading-zero forever to hide breaks.

**Changelog source of truth is `CHANGELOG.md`** in the repo (Keep a Changelog: Unreleased + version sections). GitHub Releases are the **published copy** of that version’s section, not a second handwritten history. CI on tag `vX.Y.Z`:

1. Fail if `CHANGELOG.md` has no matching section (or still says Unreleased-only).
2. Create the GitHub Release body from that section.
3. Later (phase 8): attach signed artifacts and the Tauri updater JSON already required by issue 14.

Do **not** dump every git subject onto the landing page. Commit subjects (already `fix(api): …` shaped) may **draft** the Unreleased section; a human or agent **edits** product language before tag. Auto-changelog-only is not the public changelog.

**Surfaces** (same file, three places):

| Place | How |
| --- | --- |
| GitHub Releases | CI copies the tagged section |
| Landing page | Static: site build renders recent sections from `CHANGELOG.md` (no live GitHub API required) |
| App | Bundle the same changelog (or a JSON extract) at build time so “What’s new” works offline and matches the binary. Updater (issue 14) is how you learn a *newer* version exists — do not fetch GitHub on every launch for the notes of the version you already run. |

CLI `sopsdeck --version` / about UI read the same version string as Tauri (`tauri.conf.json` / build ldflags), not a fourth source.

## Already there

- Issue 14: semver, GitHub Releases, Tauri updater JSON, no Sopsdeck backend.
- `desktop/src-tauri/tauri.conf.json` and `desktop/package.json` are `0.1.0`.
- No repo `CHANGELOG.md`, no `.github/workflows` release job.

## Acceptance criteria

- [ ] `CHANGELOG.md` exists with Unreleased; user-facing work adds bullets there before or with the change.
- [ ] Documented bump rules: epoch-semver; first public tag `v1.0.0`; EPOCH only for a named era.
- [ ] Tag pipeline: `vX.Y.Z` → GitHub Release notes equal that changelog section (fail CI if missing).
- [ ] Landing page changelog is generated from `CHANGELOG.md` (or a generated fragment checked in by `./scripts/docs`).
- [ ] App can show What’s new for the running version from bundled notes (no required network).
- [ ] One version number for CLI + app; a test or check fails if they drift.

## Seams

- `CHANGELOG.md` + tag name.
- Site build / `./scripts/docs` for the landing fragment.
- CLI `--version` and Tauri `version`.
- GitHub Release API as a *result* of CI, not the authoring UI.

## Comments

Captured 2026-08-28. Kind: idea → build. Reporter asked about epoch-semver, GitHub Releases as changelog, CI, landing page, in-app notes, and whether a changelog file is required.
