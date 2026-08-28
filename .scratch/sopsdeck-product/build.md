# Build board

Agents implement from here. Spec policy stays in [map.md](map.md) and issues 01–18. Do not invent product policy in a build slice.

Use `/tdd` at the public seams in [Lock public test seams and delivery phases](issues/15-lock-public-test-seams-and-delivery-phases.md). Living proof is [docs/seams.md](../../docs/seams.md) and [docs/features.md](../../docs/features.md).

Human QA lands in [human-found-bugs-and-review.md](../../human-found-bugs-and-review.md). Process that file into tickets, then build those before more phase-15 tail unless a slice is blocked.

## Next

1. Incomplete tracer-bullet phases in issue 15 (scan, MCP, signed macOS release). Request PR / re-encrypt PR still open on recipients. Fake GitHub PUTs are not Libsodium-sealed. [31](issues/31-deferred-product-ideas.md) is backlog (clipboard/paste path ride with 12/05; OpenBao needs spec).

Tickets 19–30 are done as shipped. Tag CI in [22](issues/22-epoch-semver-and-changelog.md) publishes changelog notes; phase 8 still owes signed artifacts on that workflow. Check CI is [`.github/workflows/check.yml`](../../.github/workflows/check.yml).

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
| 2 | Tauri folder-first editor | in progress | discover Managed Files; open; edit; atomic save; `SOPSDECK_DEV_PROJECT` boot; chrome (21); errors (20); folder pick (23); key CRUD (24); add-file / identity / scroll (25) | paste (12) |
| 3 | Git Commit / Sync | in progress | commit; Sync; refuse dirty/diverge; Review; Secret History; Restore; three-way Review | — |
| 4 | Recipients | in progress | `recipient add` / `remove` + data-key rotate; studio teammate after Sync | request PR; re-encrypt PR (06) |
| 5 | GitHub Publish | in progress | `publish` prefix, dry-run, `--yes`, `--prune`; `.sopsdeck.toml` mappings; last-published prune; `GH_TOKEN` / `gh auth token`; inspector mapping + prune vs `internal/githubfake` | — |
| 6 | Scan hook | not started | — | issue 10; unused-key analysis is [31](issues/31-deferred-product-ideas.md) |
| 7 | Paste + MCP | not started | — | issues 11, 12; clipboard modal is 12 + [31](issues/31-deferred-product-ideas.md) |
| 8 | Signed macOS release | not started | — | issue 14; tag notes already from [22](issues/22-epoch-semver-and-changelog.md) |

## Ready build tickets

| # | Title | Status |
| --- | --- | --- |
| [19](issues/19-drive-professional-product-assets.md) | Professional videos, snippets, stills; docs generated/testable | **done** (pipeline; clips unusable → 28) |
| [20](issues/20-contextual-failure-ux.md) | Contextual failures, not raw Git or toasts | **done** |
| [21](issues/21-desktop-chrome-polish.md) | Motion, states, commit default, theme, display paths | **done** |
| [22](issues/22-epoch-semver-and-changelog.md) | Epoch SemVer, CHANGELOG.md, GH Release + site + in-app notes | **done** |
| [23](issues/23-fix-folder-open-hang.md) | Folder picker beach-balls | **done** |
| [24](issues/24-editor-key-row-actions.md) | Reveal, copy, rename, delete, add-key composer | **done** |
| [25](issues/25-sidebar-window-and-scroll.md) | Add Managed File, theme icon, app identity, scroll | **done** |
| [26](issues/26-visual-polish-and-changelog-look.md) | Icons/motion; changelog look | **done** |
| [27](issues/27-realistic-managed-file-fixtures.md) | eas.json, compose, multiline fixtures | **done** |
| [28](issues/28-usable-product-recordings.md) | webreel + CLI casts; fail sub-second clips | **done** |
| [29](issues/29-docs-site.md) | One public site: landing, changelog, docs | **done** |
| [30](issues/30-deterministic-quality-gates.md) | Markdown, security scanner, changelog/test hooks | **done** |
| [31](issues/31-deferred-product-ideas.md) | Unused keys, smart rename, clipboard, OpenBao, paste-path | backlog |
