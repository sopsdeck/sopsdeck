# Deferred product ideas from human QA

Type: build
Status: ready
Blocked by: See each item — several need spec, not just code

## What to build

Backlog only. Do not start these ahead of [23](23-fix-folder-open-hang.md)–[25](25-sidebar-window-and-scroll.md). Do not invent policy here; if a slice needs new rules, open a spec ticket first.

### Unused secrets

Show keys in a Managed File that are not referenced in the Project. Adjacent to scan ([10](10-specify-local-secret-scanning.md)), but "unused" is reference analysis, not leak detection. Needs a rule for what counts as a reference (exact `KEY`, `$KEY`, `${KEY}`, …).

**Done (2026-08-29).** `sopsdeck references -f FILE` lists each key with its reference count and the files that reference it; `sopsdeck unused -f FILE` lists keys with zero references. Reference rule is a whole-word match (`\bKEY\b`), which covers `KEY`, `$KEY`, and `${KEY}` and rejects substrings. The scan walks the project root, skips `.git`, binary files, and the Managed Files themselves. Desktop: keys with zero references show an "unused" badge in the inspector.

### Smart rename

Renaming a key offers to update references in the repo. Needs the same reference rule as unused secrets. In-file rename is [24](24-editor-key-row-actions.md); this is cross-file.

**Done (2026-08-29).** `sopsdeck rename OLD NEW -f FILE` previews the files whose references would be rewritten; `--yes` renames the key in the Managed File and rewrites whole-word references across the project. Desktop: when a key is renamed, a "Rewrite references in the project" checkbox appears; on Encrypt & save it calls `rename_key` (one decrypt/encrypt + cross-file rewrite).

### Smart clipboard

On focus, if the clipboard looks like a secret or an Age recipient, offer a modal: which Managed File / which Project gets the value or Access. This **is** issue [12](12-specify-paste-and-editing-workflows.md) (sniff + preview + confirm) plus [06](06-specify-identity-access-and-recovery.md) for recipient keys. Implement with 12/paste, not as a side channel.

**Done (2026-08-29).** On window focus the app reads the clipboard, classifies it (path / recipient / bulk / lone), and opens a modal offering the matching action. Implemented with the 12/paste seam (`classifyClipboard`) and the recipient grant seam.

### OpenBao

A later Sync Target. Map: not specified (EAS API, GitLab, cloud secret managers, …). Do not build until a specify ticket exists. OpenBao is a candidate, not a decision.

**Parked.** No specify ticket; no code.

### Paste a filesystem path to open a Project

Copy an absolute path from a terminal or editor, focus Sopsdeck, open that folder. Fits issue [05](05-define-project-manifest-and-registration.md) explicit add. Clipboard path sniff can share UI with smart clipboard. Depends on [23](23-fix-folder-open-hang.md) so add-folder actually works.

**Done (2026-08-29).** The clipboard modal's path kind offers "Open this folder as a Project", reusing the existing `addProjectFromPath` seam.

## Acceptance criteria

- [ ] Each idea is either implemented behind a later ticket with real AC, or explicitly dropped.
- [ ] No OpenBao/Sync Target code without a resolved specify issue.

## Comments

Captured 2026-08-28 from human-found review Feature Ideas. Kind: idea.
