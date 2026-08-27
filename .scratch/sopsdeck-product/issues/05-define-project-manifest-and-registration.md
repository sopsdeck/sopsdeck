# Define Project registration and the Project Manifest

Type: grilling
Status: resolved
Blocked by: 17

## Question

What is the complete lifecycle and schema for explicitly registering a Project, selecting/discovering Managed Files inside it, sharing committed project policy (if any), handling moves/worktrees, and reconciling local-only preferences without leaking secrets or machine paths?

## Answer

Projects are added explicitly (folder picker), Codex-style. No full-disk scan. Separate worktrees are separate Projects. The app stores the canonical path plus recents in machine-local state (never committed, never a secret). If the folder moved, prompt to relocate; do not search the disk.

Committed **Project Manifest** is `.sopsdeck.toml` at the Project root. It lists Managed Files (paths relative to the Project), Sync Target mappings, and scan-hook policy. It contains no credentials, no secret values, no machine paths. Recipients live in `.sops.yaml` / SOPS file metadata, not in the manifest.

Managed Files are: paths listed in the manifest, plus any file in the Project that already has SOPS metadata. Adding a file to the manifest is explicit. Local UI prefs (last selected file, window) stay machine-local.
