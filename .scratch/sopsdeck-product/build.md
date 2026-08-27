# Build board

Agents implement from here. Spec policy stays in [map.md](map.md) and issues 01–18. Do not invent product policy in a build slice.

Use `/tdd` at the public seams in [Lock public test seams and delivery phases](issues/15-lock-public-test-seams-and-delivery-phases.md). Living proof is [docs/seams.md](../../docs/seams.md) and [docs/features.md](../../docs/features.md).

## Next

1. [21 Desktop chrome polish](issues/21-desktop-chrome-polish.md) — display paths, empty/loading, hover/motion, commit-message default, theme.
2. [19 Product assets and testable docs](issues/19-drive-professional-product-assets.md) — after 21 so walkthroughs and stills do not leak temp paths.
3. [22 Epoch SemVer and changelog](issues/22-epoch-semver-and-changelog.md) — parallel; Unreleased bullets already started.

Versioning/changelog ([22](issues/22-epoch-semver-and-changelog.md)) can land in parallel with 20–21: add `CHANGELOG.md` Unreleased bullets on user-facing work; tag CI waits for phase 8 artifacts.

Then return to incomplete tracer-bullet phases in issue 15 (recipient remove/rotate, Review/History/Restore, Publish mappings, scan, MCP, signed release).

## Already in the tree (not a phase by itself)

| Substrate | What it is | Proof |
| --- | --- | --- |
| Studio | Two throwaway Age Users, bare origin, fake GitHub Actions secrets API. No extra machine or GitHub account. | `TestTeammateDecryptsAfterRecipientAddAndSync`, publish tests under `internal/studio/` |
| `sopsdeck drive` | Localhost HTTP of the real UI + `POST /invoke` matching Tauri commands. `--demo` seeds `checkout` + Bob Access. | `internal/cli/drive_test.go`, `./scripts/smoke` |
| Playwright | UI smoke + stills. Not in `./scripts/check`. | `./scripts/smoke`, `./scripts/demo` → `docs/assets/` |

## Phase status

From issue 15. “Has a test” in `docs/seams.md` means the seam is open, not that the phase is done.

| Phase | Slice | Status | Proved | Still open |
| --- | --- | --- | --- | --- |
| 1 | CLI core | in progress | get/set/del/run; identity create/import with `--confirmed-backup` to a state-dir Age file | OS keychain / `SOPS_AGE_KEY_CMD` (issue 06) |
| 2 | Tauri folder-first editor | in progress | discover Managed Files; open; edit; atomic save; `SOPSDECK_DEV_PROJECT` boot | paste (12); chrome (21); errors (20) |
| 3 | Git Commit / Sync | in progress | commit `-m` `-f`; Sync = fetch + ff-only pull + push; refuse diverge | Review semantic diff; Secret History; Restore; dirty-worktree stop; three-way (07) |
| 4 | Recipients | in progress | `recipient add`; second identity decrypts; studio teammate after Sync | remove + data-key rotate; request PR; re-encrypt PR (06) |
| 5 | GitHub Publish | in progress | `publish` prefix, dry-run default, `--yes`, `--prune` against `internal/githubfake` | `.sopsdeck.toml` mappings; `gh` auth; environment secrets; last-published names; desktop Publish (09) |
| 6 | Scan hook | not started | — | issue 10 |
| 7 | Paste + MCP | not started | — | issues 11, 12 |
| 8 | Signed macOS release | not started | — | issue 14; changelog/versioning [22](issues/22-epoch-semver-and-changelog.md) |

## Ready build tickets

| # | Title | Status |
| --- | --- | --- |
| [19](issues/19-drive-professional-product-assets.md) | Professional videos, snippets, stills; docs generated/testable | ready (after 20, 21) |
| [20](issues/20-contextual-failure-ux.md) | Contextual failures, not raw Git or toasts | **done** |
| [21](issues/21-desktop-chrome-polish.md) | Motion, states, commit default, theme, display paths | ready (after or overlapping 20) |
| [22](issues/22-epoch-semver-and-changelog.md) | Epoch SemVer, CHANGELOG.md, GH Release + site + in-app notes | ready (parallel; tag CI with phase 8) |
