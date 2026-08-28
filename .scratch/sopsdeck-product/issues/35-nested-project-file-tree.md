# Nested Project file tree

Type: build
Status: open
Blocked by: None

## What to build

The left file tree should support nested folders, collapse folders, recents, and truncation with a show-more control (Codex-like). Discovery rules stay issue [05](05-define-project-manifest-and-registration.md) / [04](04-validate-folder-first-workspace.md): do not invent a second registry.

## Acceptance criteria

- [ ] Nested folders under a Project are grouped and collapsible.
- [ ] A recents affordance exists without a new cloud account.
- [ ] Long lists truncate with an explicit show-more control.

## Seams

- Drive + Playwright; existing `list_managed_files`.

## Comments

Captured 2026-08-28 from human-found review. Kind: idea.
