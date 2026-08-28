# Build board

Agents implement from here. Spec policy stays in [map.md](map.md) and issues 01–18. Do not invent product policy in a build slice.

Use `/tdd` at the public seams in [Lock public test seams and delivery phases](issues/15-lock-public-test-seams-and-delivery-phases.md). Living proof is [docs/seams.md](../../docs/seams.md) and [docs/features.md](../../docs/features.md).

## Next

1. Incomplete tracer-bullet phases in issue 15 (three-way conflict review, `.sopsdeck.toml` Publish mappings, scan, MCP, signed macOS release). Request PR / re-encrypt PR still open on recipients.

Tickets 19–22 are done. Tag CI in [22](issues/22-epoch-semver-and-changelog.md) publishes changelog notes; phase 8 still owes signed artifacts on that workflow.

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
| 2 | Tauri folder-first editor | in progress | discover Managed Files; open; edit; atomic save; `SOPSDECK_DEV_PROJECT` boot; chrome (21); errors (20) | paste (12) |
| 3 | Git Commit / Sync | in progress | commit; Sync; refuse dirty/diverge; Review; Secret History; Restore | three-way (07) |
| 4 | Recipients | in progress | `recipient add` / `remove` + data-key rotate; studio teammate after Sync | request PR; re-encrypt PR (06) |
| 5 | GitHub Publish | in progress | `publish` prefix, dry-run default, `--yes`, `--prune` against `internal/githubfake` | `.sopsdeck.toml` mappings; `gh` auth; environment secrets; last-published names; desktop Publish (09) |
| 6 | Scan hook | not started | — | issue 10 |
| 7 | Paste + MCP | not started | — | issues 11, 12 |
| 8 | Signed macOS release | not started | — | issue 14; tag notes already from [22](issues/22-epoch-semver-and-changelog.md) |

## Ready build tickets

| # | Title | Status |
| --- | --- | --- |
| [19](issues/19-drive-professional-product-assets.md) | Professional videos, snippets, stills; docs generated/testable | **done** |
| [20](issues/20-contextual-failure-ux.md) | Contextual failures, not raw Git or toasts | **done** |
| [21](issues/21-desktop-chrome-polish.md) | Motion, states, commit default, theme, display paths | **done** |
| [22](issues/22-epoch-semver-and-changelog.md) | Epoch SemVer, CHANGELOG.md, GH Release + site + in-app notes | **done** |
