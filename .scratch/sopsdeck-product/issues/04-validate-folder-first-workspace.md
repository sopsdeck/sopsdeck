# Validate the folder-first desktop workspace

Type: prototype
Status: resolved
Blocked by: None

## Question

Does the folder-first three-pane concept let a developer add Projects, navigate mixed Managed Files and nested Environments, paste/edit values, understand encryption/Git state, and finish the core task with Codex-like focus; what must change before implementation?

## Answer

Yes. [sopsdeck-ui-concept.html](../../sopsdeck-ui-concept.html) is the approved information architecture: one sidebar of Project folders, Managed Files as nested leaves, one focused editor, access / encryption / Git in the inspector — Codex-like, not a chat. No structural change before implementation. Remaining screen states (onboarding, empty, error, keyboard) stay later fog, not a new layout.

## Implementation (2026-08-28)

IA is in `desktop/src` (Tauri and `sopsdeck drive`). Fog is now build tickets, not unspecified: [20 Contextual failure UX](20-contextual-failure-ux.md), [21 Desktop chrome polish](21-desktop-chrome-polish.md).

