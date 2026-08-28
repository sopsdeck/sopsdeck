# Desktop chrome: motion, states, commit copy, theme, paths

Type: build
Status: done
Blocked by: None

## What to build

Finish screen chrome on the locked three-pane IA ([Validate the folder-first desktop workspace](04-validate-folder-first-workspace.md)). Not a new layout. Visual language from [brand-spec.md](../../brand-spec.md) and [sopsdeck-ui-concept.html](../../sopsdeck-ui-concept.html).

Must include:

- Hover and modest motion (rows, buttons, file selection) — feel finished, not a prototype.
- Empty states: no Project yet; Project with no Managed Files; open file with no keys.
- Loading states while list/get/save/Sync/Publish run (drive and Tauri).
- Prefill the commit message from **what actually changed** in the open Managed File (added/removed/changed keys). Issue 07 still requires an explicit commit with that message — no silent commit. User can edit the default.
- Dark mode / theme setting (persist for the app; follow brand, not a random palette).
- **Display paths** people can read: Project-relative or `~/folder/file`, never `/Users/…/desktop/../testdata/hello.yaml` or `/var/folders/…/sopsdeck-demo-…/checkout/.env.production` in breadcrumb, inspector, or screenshots. Canonical filesystem path may remain on disk and in invoke payloads.

Issue 04 called onboarding/empty/error/keyboard “later fog”. Error placement is [20](20-contextual-failure-ux.md). This ticket is the rest of the chrome, including paths that [19](19-drive-professional-product-assets.md) will film.

## Already there

- Three-pane editor; display paths `~/project/rel`; empty/loading chrome; commit default; light/dark theme.

## Acceptance criteria

- [x] Playwright (drive `--demo`): breadcrumb and inspector path do not contain `..`, `/var/folders`, or `desktop/../`.
- [x] Empty states are distinct and copy-driven (no blank three-pane with only “Sopsdeck”).
- [x] A slow invoke shows loading on the control that was pressed (save or Sync at minimum).
- [x] After editing keys, the commit field is prefilled from those changes; Commit still sends `-m` with that text (editable).
- [x] Theme can switch light/dark and survive reload (`localStorage` or equivalent is fine).
- [x] Hover/focus states exist on file rows and primary actions (visual; concept HTML is the reference).

## Seams

- Desktop UI via `sopsdeck drive` + Playwright (`data-testid` on breadcrumb, inspector path, commit field, empty/loading regions).
- Commit still hits existing `commit_managed_file` / `sopsdeck commit -m`.

## Implementation (2026-08-28)

- Breadcrumb and inspector use `~/project/rel` with `..` stripped (`displayPath`). Canonical path stays on invoke payloads.
- Empty copy: no Project (`/?empty=1`), Project with no Managed Files, open file with no keys.
- Sync/save set `aria-busy` and a short label while the invoke runs.
- Commit field prefills `Change KEY` / `Add KEY` from dirty rows; typing a different message sticks.
- `data-theme` + `localStorage sopsdeck-theme`; hover/focus on file rows and primary actions.
- Playwright: `e2e/chrome.spec.js` via `./scripts/smoke`.

## Comments

Captured 2026-08-28; triaged 2026-08-28; done 2026-08-28. Kind: idea → build.
