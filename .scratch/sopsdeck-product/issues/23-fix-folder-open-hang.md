# Fix folder-open hang

Type: build
Status: done
Blocked by: None

## What to build

Opening a Project folder from the desktop app beach-balls (macOS spinning wait cursor) and never recovers. Human QA: “app beach balls if i try opening a folder.”

`pick_project_folder` uses a **blocking** native folder dialog on the Tauri command thread. That is the first suspect. The button still has to add a folder from disk (issue 05: explicit folder add). Drive may keep a path-typed fallback; native Tauri must not freeze the WebView.

Do not invent a new add-Project flow. Unblock the existing one.

## Already there

- `pick_project_folder` is an async Tauri command so `blocking_pick_folder` runs off the main thread (plugin docs). Cancel still returns `None`. Drive still boots `SOPSDECK_DEV_PROJECT`.

## Acceptance criteria

- [x] In the real Tauri app (not only drive), Add folder from disk opens a native folder picker and returns without hanging the UI.
- [x] Cancel leaves the app usable; choosing a folder still lists Managed Files.
- [x] Playwright/drive path still works (typed path or existing demo boot). Native dialog is Tauri-only.

## Seams

- Tauri `pick_project_folder` (and any follow-up `list_managed_files` after pick).
- Desktop chrome test if a non-dialog path exists; hang itself is a human/Tauri check.

## Implementation (2026-08-28)

- `#[tauri::command] async fn pick_project_folder` with `clippy::unused_async` allowed (thread hop; no `.await`).
- Drive `pick_project_folder` / `boot_project` unchanged.

## Comments

Captured 2026-08-28 from [human-found-bugs-and-review.md](../../../../human-found-bugs-and-review.md). Kind: bug. Priority: P0 — the app is unusable without adding a folder. Done 2026-08-28.
