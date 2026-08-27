# Contextual failure UX instead of raw Git or toasts

Type: build
Status: done
Blocked by: None

## What to build

Map Git, SOPS, and GitHub failures to **short, actionable copy** at the CLI seam, then show that copy **on the control the user just used** (Sync, Encrypt & save, Commit, Publish). Retry in that same place.

Unacceptable (current Sync path via `runGitCmd`): dumping `git-pull(1)` / `--set-upstream-to` text into `#error` as red paragraphs.

Also unacceptable: toast-only failures. Attention is on the inspector Git block or the save zone, not a corner popup.

Policy already locked in [Specify failure, privacy, and operational recovery behavior](13-specify-failure-privacy-and-recovery-ux.md): visible, retryable, no secret values in the message. This ticket is presentation + CLI wording. Do not change atomic writes, prune-on-partial-PUT, or redaction rules.

At least these cases (names in CONTEXT.md):

| Situation | User-facing idea (exact words during TDD) |
| --- | --- |
| Branch has no upstream | Sync cannot see a remote branch yet — set upstream or push first. Not `git-pull(1)`. |
| Diverged (already refused in tests) | Same rule as issue 07: stop and say the branch diverged; never force. |
| Dirty Managed File on Sync | Stop; save/commit first (issue 07). |
| Decrypt / no Access | File is shown, not overwritten (issue 13); say Access is missing. |
| Publish PUT failed | No prune (issue 13); say Publish did not finish; retry on Publish. |

CLI stderr is the string the desktop surfaces. One or two sentences, no man-page paste.

## Already there

- `cmdSync` / `cmdCommit` wrap `git` and prefix `sync:` / `commit:` with whatever Git printed.
- Desktop `showError(String(err))` into a single `#error` in the editor column.
- Tests: `TestSyncRefusesWhenBranchHasDiverged`.

## Acceptance criteria

- [x] `TestSyncWithoutUpstreamExplainsMissingTracking` (or equivalent) fails if stderr contains `git-pull(1)` or `--set-upstream-to`.
- [x] Diverged and dirty-worktree Sync failures stay refusals (issue 07) with the new copy, not raw Git.
- [x] Desktop: Playwright (drive) — after a Sync failure, the message is next to the Sync control (`data-testid` on that region), `#error` is not a toast, and the control can be clicked again.
- [x] Decrypt and Publish failures use the same pattern (control-adjacent, retryable, no secret values).
- [x] No toast component for these failures.

## Seams

- `sopsdeck` CLI (`sync`, `commit`, `get`/`set`, `publish`) stderr.
- Drive/Tauri invoke error payload the UI already returns.
- Playwright against `sopsdeck drive` for placement.

## Next slice

Start at the CLI: upstream-missing Sync. Then wire the same string into the Sync control in `desktop/src`. Then the other rows in the table.

## Comments

Captured 2026-08-28; triaged 2026-08-28. Kind: idea → build. **Implement this first** ([build board](../build.md)).
