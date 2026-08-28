# Editor key rows: reveal, copy, rename, delete, add

Type: build
Status: done
Blocked by: None

## What to build

The focused editor cannot finish basic key work. Human QA:

- No delete on a secret (CLI `del` exists; UI does not).
- Add-key UI hides the key field; hide/show and add feel lazy.
- Values need click-to-reveal, copy, and delete as icons — not a text toggle.
- Keys should copy on hover and edit in place by clicking the name.

Replace the current add-row chrome. Options that fit issue 12 (paste/edit) without a new layout:

- Icon reveal/hide and copy/delete on each value row.
- Add via a quiet hover affordance at the bottom of the list, **or** a single composer (key, `key=value` paste, or just a key) — structured rows stay the default (issue 12).
- Click the key name to rename; hover copy for the key.

Do not add smart rename of references in the rest of the repo here ([31](31-deferred-product-ideas.md)). This ticket is in-file key CRUD and chrome.

## Already there

- Per-row reveal / copy value / delete icons; hover copy on the key name; names always editable.
- Composer `KEY` or `KEY=value`; Add secret focuses it. Empty files still show the composer.
- Save: `del_managed_key` for removed and renamed orig keys, then `set_managed_key`. Drive and CLI `set -f FILE` / `del` are the backend.

## Acceptance criteria

- [x] An existing key can be deleted from the editor; save writes ciphertext without that key (`del` semantics).
- [x] Adding a key always shows a visible key field (or composer) before save; Playwright covers the add path.
- [x] Value row: reveal/hide, copy value, delete — icon controls, not only a text button.
- [x] Key name: click to edit; hover copy icon copies the key (not the value).
- [x] New-key affordance is a hover/composer control, not a stuck empty row that clips the name.

## Seams

- Existing `get` / `set` / `del` via drive + Playwright (`data-testid` on row actions).
- Paste composer, if used, must follow issue 12 (preview/confirm for bulk; lone value vs dotenv). Single-line composer only; bulk paste stays [12](12-specify-paste-and-editing-workflows.md).

## Implementation (2026-08-28)

- `e2e/chrome.spec.js`: composer visible; name editable; row icons; add survives reload; delete + save removes the key.
- Commit prefill includes `Remove KEY`.

## Comments

Captured 2026-08-28 from human-found review. Kind: bug + idea. Spec: [12](12-specify-paste-and-editing-workflows.md), CLI `del` in issue 03. Done 2026-08-28.
