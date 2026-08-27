# Choose the cross-platform desktop and core architecture

Type: research
Status: resolved
Blocked by: None

## Question

Which desktop shell, core runtime, process boundary, and code-sharing model best satisfy macOS-first delivery, later Windows/Linux support, secure key handling, SOPS/Git integration, CLI reuse, signing, and updates: Electron, Tauri, or another evidence-backed option?

## Answer

The combination that best satisfies those constraints together is a **Go `sopsdeck` core that is also the public CLI**, with **Tauri 2** as the desktop shell invoking that CLI as a **sidecar from the Rust core** (not from the WebView). Official SOPS is a Go project whose only stable library API is `github.com/getsops/sops/v3/decrypt`; maintainers tell everyone else to use the CLI, and the independent Rust rewrite (`rops`) still lacks ENV/dotenv, PGP, and several KMS backends—so a Go engine (decrypt package plus official `sops` or pinned internals for encrypt) is the fidelity path. Treat the UI process as untrusted: no keys, no plaintext persistence, no `git`/`sops` spawn; use Tauri capabilities (and the isolation pattern) so the frontend cannot widen shell permissions, and pull age identities via `SOPS_AGE_KEY_CMD` or OS keychain APIs rather than Tauri Stronghold (deprecated, not in v3). For Git writes that must match user Git (rebase, extra worktrees, LFS), spawn the system `git` from the privileged core—go-git documents those as missing. Tauri’s bundler covers Developer ID signing and notarization; the updater requires signatures and can use a static JSON feed, so updates do not imply a Sopsdeck backend; Linux is included, unlike Electron’s official Squirrel `autoUpdater`. Wails v2 plus the same Go core is the runner-up if in-process Go matters more than a capability DSL (no first-party updater in v2; v3 is beta). Native Swift UI plus the same core plus Sparkle is the runner-up if eliminating the WebView on macOS outweighs a single UI for later Windows/Linux. Electron remains viable (`safeStorage`, Forge signing) but is the largest TCB for a secrets app and lacks official Linux auto-update. This does not choose age vs PGP, Keychain vs file identities, or macOS-only vs multi-OS v1.

Findings: [research/01-desktop-and-core-architecture.md](../research/01-desktop-and-core-architecture.md)

## Implementation (2026-08-28)

Go CLI + Tauri sidecar is in the tree. `sopsdeck drive` serves the same WebView UI over localhost with `POST /invoke` matching Tauri commands so Playwright can drive the real UI without widening WebView permissions. Identities in tests/studio are Age files, not keychain yet (06).

