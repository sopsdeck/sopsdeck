# Nested Project file tree

Type: build
Status: done
Blocked by: None

## What to build

The left file tree should support nested folders, collapse folders, recents, and truncation with a show-more control (Codex-like). Discovery rules stay issue [05](05-define-project-manifest-and-registration.md) / [04](04-validate-folder-first-workspace.md): do not invent a second registry.

## Acceptance criteria

- [x] Nested folders under a Project are grouped and collapsible.
- [x] A recents affordance exists without a new cloud account.
- [x] Long lists truncate with an explicit show-more control.

## Seams

- Drive + Playwright; existing `list_managed_files`.

## Implementation (2026-08-28)

Sidebar groups Managed Files by relative folder from `list_managed_files`. Nested folders collapse; state is `localStorage` `sopsdeck-tree-folders`. Recents are project paths in `sopsdeck-recents` (machine-local, not a registry). Root lists longer than 8 show Show more. `set -f` creates parent folders so Add file can write nested paths.

## Comments

Captured 2026-08-28 from human-found review. Kind: idea.
