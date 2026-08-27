# Lock public test seams and delivery phases

Type: grilling
Status: resolved
Blocked by: 04, 05, 06, 07, 09, 10, 11, 12, 13, 14, 16, 17, 18

## Question

Once the product decisions are resolved, which public seams must TDD exercise across core, CLI, desktop, provider, MCP, and Git-hook behavior, and which tracer-bullet phases reach a secure usable release without horizontal subsystem builds?

## Answer

Public seams (test here, not internals): (1) `sopsdeck` CLI — encrypt/decrypt/get/set/del/run against SOPS fixtures, identity backup gate, recipient add/remove rotate; (2) Git adapter — review/commit/Sync/history/restore via system git in a temp repo; (3) GitHub Publish adapter — PUT/list/delete with prefix and prune rules against a fake API; (4) scan hook — staged fixtures, ciphertext ignored, allowlist, block vs warn; (5) MCP — metadata vs approved plaintext vs run; (6) atomic write + no plaintext on disk.

Tracer-bullet phases, each a vertical slice on those seams:

1. CLI core: Age identity + PM-backup confirm, dotenv/JSON/YAML SOPS get/set/del/run.
2. Tauri folder-first editor: add Project, open Managed File, edit, atomic encrypted save.
3. Git Review / Commit / Sync / Secret History.
4. Recipients: request PR, re-encrypt PR, remove/replace with data-key rotate.
5. Publish to GitHub Actions secrets (prefix, dry-run, prune off).
6. Opt-in scan hook.
7. Paste preview + local MCP/skills.
8. Signed macOS app + `sopsdeck`/`sd` on GitHub Releases.
