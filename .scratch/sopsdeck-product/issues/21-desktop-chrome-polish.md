# Desktop chrome: motion, states, commit copy, theme, paths

Type: build
Status: ready
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

- Three-pane editor; `parentLabel` shows `~/parent` in the sidebar only.
- Breadcrumb and inspector use the raw absolute `file.path`.
- Commit message is an empty placeholder (“What changed”).
- `./scripts/demo` stills currently leak the studio temp path.

## Acceptance criteria

- [ ] Playwright (drive `--demo`): breadcrumb and inspector path do not contain `..`, `/var/folders`, or `desktop/../`.
- [ ] Empty states are distinct and copy-driven (no blank three-pane with only “Sopsdeck”).
- [ ] A slow invoke shows loading on the control that was pressed (save or Sync at minimum).
- [ ] After editing keys, the commit field is prefilled from those changes; Commit still sends `-m` with that text (editable).
- [ ] Theme can switch light/dark and survive reload (`localStorage` or equivalent is fine).
- [ ] Hover/focus states exist on file rows and primary actions (visual; concept HTML is the reference).

## Seams

- Desktop UI via `sopsdeck drive` + Playwright (`data-testid` on breadcrumb, inspector path, commit field, empty/loading regions).
- Commit still hits existing `commit_managed_file` / `sopsdeck commit -m`.

## Comments

Captured 2026-08-28; triaged 2026-08-28. Kind: idea → build. Paths first if 19 stills need to stop leaking temp dirs before videos.
