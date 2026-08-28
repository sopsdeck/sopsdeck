# Sidebar files, window identity, theme control, scroll

Type: build
Status: done
Blocked by: None

## What to build

Chrome around the three-pane IA that is still wrong or missing:

- Sidebar cannot add another Managed File (create/register a new encrypted file under the Project).
- Theme control is a “dark mode” text button; it should be a compact icon (or equivalent) that still persists (`sopsdeck-theme`).
- The running app shows as “desktop” with no product logo (Dock / window / about). Bundle already has icons in `desktop/src-tauri`; the process/title/bundle identity is wrong in the build people actually launch.
- Scroll: ugly scrollbars; the window can scroll so the chrome “body” is visible. Panes should scroll internally; the shell should not.

Adding a Managed File is a real product gap (issue 05 paths live in `.sopsdeck.toml` / discovery). Do not invent a second tree. Sidebar gets an add-file action that creates a SOPS-encrypted file the existing discoverer will list.

## Already there

- Sidebar name + Add file → `create_managed_file` (`sopsdeck set -f`); `..` and paths outside the Project are refused.
- Theme is a sun/moon icon; `sopsdeck-theme` still persists.
- Cargo/package name `sopsdeck`, `productName` / `mainBinaryName` `Sopsdeck`, lib still `desktop_lib`.
- `html`/`body`/`.app` do not scroll; `.tree`, `.keys`, `.inspect-body` scroll with thin bars.

## Acceptance criteria

- [x] From the sidebar, the user can add a new Managed File; after save it appears in the tree and opens in the editor.
- [x] Theme toggle is an icon (or equally compact control), still persists across reload.
- [x] Packaged/dev app title and Dock/taskbar name are **Sopsdeck** with the product icon, not “desktop”.
- [x] Outer window does not rubber-band-scroll to empty body chrome; lists/editor/inspector scroll inside panes. Scrollbars match brand (minimal, not OS-default ugly if the WebView allows).

## Seams

- Drive + Playwright for add-file and theme control.
- Tauri `productName` / bundle / `package.json` name for window identity (human check for Dock).

## Implementation (2026-08-28)

- Playwright: add `.env.ui-added`; reject `../escape.env`; document `scrollHeight` vs `clientHeight`.
- Empty encrypted create is `sopsdeck set -f FILE` (no KEY).

## Comments

Captured 2026-08-28 from human-found review. Kind: bug + idea. Spec: [05](05-define-project-manifest-and-registration.md), [04](04-validate-folder-first-workspace.md). Done 2026-08-28.
