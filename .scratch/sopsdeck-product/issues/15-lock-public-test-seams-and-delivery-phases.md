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

## Implementation (2026-08-28)

Spec above is unchanged. Vertical build status lives on [build.md](../build.md). `docs/seams.md` lists tests that currently touch each phase seam — that is not “phase complete”.

| Phase | In tree | Not in tree |
| --- | --- | --- |
| 1 | get/set/del/run; identity create/import gated on `--confirmed-backup`; Age file under `SOPSDECK_STATE_DIR` | Keychain / `SOPS_AGE_KEY_CMD` (06) |
| 2 | Tauri + drive UI: list, open, edit, atomic save, boot folder | Paste (12); chrome (21); errors (20) |
| 3 | commit; Sync fetch + ff-only pull + push; refuse diverge | Review; Secret History; Restore; dirty stop (07) |
| 4 | `recipient add` / `recipient remove` + data-key rotate; studio teammate after Sync / after remove | Request PR; re-encrypt PR (06) |
| 5 | `publish` prefix, dry-run, `--yes`, `--prune` vs `internal/githubfake` | Manifest mappings; `gh` auth; environment secrets; desktop Publish (09) |
| 6–8 | changelog/versioning [22](22-epoch-semver-and-changelog.md) (tag notes; tree still `0.1.0`) | 10, 11–12, 14 signed artifacts |

**Studio / drive** (not a phase): local two-User git world + fake GitHub + `sopsdeck drive` + Playwright. Proof: studio tests, `drive_test.go`, `./scripts/smoke`. Public assets: [19](19-drive-professional-product-assets.md).

