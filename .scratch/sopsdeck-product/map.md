# Find the implementation-ready Sopsdeck specification

Label: `wayfinder:map`

## Destination

An implementation-ready product and technical specification for Sopsdeck: domain model, desktop workflows, security boundaries, CLI compatibility contract, integrations, test seams, and delivery phases are explicit enough for agents to build the product test-first without inventing product policy.

**Status: destination reached.** Remaining work is vertical build from [build.md](build.md), not this map. Phases: [issue 15](issues/15-lock-public-test-seams-and-delivery-phases.md). Ready chrome/assets tickets: [21](issues/21-desktop-chrome-polish.md) (next), [19](issues/19-drive-professional-product-assets.md). Versioning/changelog: [22](issues/22-epoch-semver-and-changelog.md). [20 Contextual failure UX](issues/20-contextual-failure-ux.md) is done.

## Notes

- Domain: local-first encrypted configuration management. Use [the domain glossary](../../CONTEXT.md).
- Build with `/tdd` one tracer-bullet phase at a time from [Lock public test seams and delivery phases](issues/15-lock-public-test-seams-and-delivery-phases.md). Current board: [build.md](build.md). Consult `/wayfinder` and `/domain-modeling` if a build slice invents policy — it must not.
- After identity, remaining tickets were resolved to recommended defaults rather than grilled.
- The existing [folder-first UI concept](../../sopsdeck-ui-concept.html) is the approved information architecture. Brand board and [brand spec](../../brand-spec.md) remain visual context, not screen-level spec.
- Canonical domain: sopsdeck.com.
- Canonical source: SOPS-encrypted Managed Files in local Projects. No required hosted backend.
- Encryption is SOPS public-key recipients. Dotenvx compatibility is CLI workflow, not ciphertext or shared private keys.

## Decisions so far

- [Validate GitHub and Infisical-like sync capabilities](issues/08-validate-provider-capabilities.md) — GitHub Actions secrets are write-only; Infisical-style prefix plus opt-in prune is required because GitHub has no ownership metadata.
- [Establish the Managed File fidelity contract](issues/02-establish-managed-file-fidelity.md) — SOPS preserves structure, not formatting; SOPS dotenv is not Node/dotenvx; encrypted `eas.json` is not valid EAS CLI input.
- [Choose the cross-platform desktop and core architecture](issues/01-choose-desktop-and-core-architecture.md) — Go `sopsdeck` CLI/core; Tauri 2 sidecar from Rust; WebView untrusted; system `git`.
- [Define the Dotenvx-shaped CLI compatibility contract](issues/03-define-dotenvx-cli-contract.md) — match `run`/`get`/`set`/`del` injection and precedence, not ciphertext or a `dotenvx` shim.
- [Define the threat model and security boundaries](issues/16-define-threat-model-and-security-boundaries.md) — dotenvx-shaped product, SOPS public-key only; plaintext in memory/`run`/copy/Publish, never a disk working copy.
- [Validate the folder-first desktop workspace](issues/04-validate-folder-first-workspace.md) — three-pane UI concept is the IA.
- [Specify identity, Access, removal, and recovery](issues/06-specify-identity-access-and-recovery.md) — Age in keychain, mandatory personal PM backup; Access per Managed File; request PR vs re-encrypt PR; CI is its own User.
- [Specify coexistence with existing SOPS projects](issues/17-specify-sops-yaml-coexistence.md) — coexist with `.sops.yaml`; honor existing trees; do not silent-encrypt mixed files.
- [Define Project registration and the Project Manifest](issues/05-define-project-manifest-and-registration.md) — explicit folder add; `.sopsdeck.toml` committed (paths, Publish mappings, scan); machine-local recents only.
- [Specify Git review, Sync, Secret History, and conflicts](issues/07-specify-git-lifecycle.md) — Git action is Sync (ff-only, never force); Publish is the Sync Target verb; restore is a new change.
- [Specify Expo eas.json handling](issues/18-specify-eas-json-handling.md) — JSON Managed File with an EAS-CLI warning; no decrypt-to-disk; EAS API later.
- [Specify MCP and AI skill security contracts](issues/11-specify-ai-tool-contract.md) — local MCP; metadata default; plaintext/mutate per-call approval; prefer `run`.
- [Specify paste, import, and editing workflows](issues/12-specify-paste-and-editing-workflows.md) — detect format, preview, confirm; SOPS dotenv grammar; structured default plus raw.
- [Define packaging, updates, and platform support](issues/14-define-release-and-support-contract.md) — `sopsdeck`/`sd`, Apache-2.0, macOS signed first, Tauri updater on GitHub Releases.
- [Specify Sync Target mapping, ownership, and pruning](issues/09-specify-sync-target-contract.md) — GitHub Actions repo/environment secrets; prefix; prune off; dry-run; only previously published prefixed names.
- [Specify local secret scanning and commit prevention](issues/10-specify-local-secret-scanning.md) — opt-in hook; block high-confidence; ignore SOPS ciphertext; `--no-verify` still works.
- [Specify failure, privacy, and operational recovery behavior](issues/13-specify-failure-privacy-and-recovery-ux.md) — atomic ciphertext writes; no values in logs; no prune after failed PUTs.
- [Lock public test seams and delivery phases](issues/15-lock-public-test-seams-and-delivery-phases.md) — CLI/Git/GitHub/scan/MCP/write seams; eight tracer-bullet phases to a signed macOS release. Implementation progress is on that issue and [build.md](build.md).

## Not yet specified

- File-watching and in-process persistence details inside the Go core (writes are atomic ciphertext).
- Later Sync Targets: EAS API, GitLab, cloud secret managers, GitHub org/Codespaces/Dependabot.

Screen chrome, contextual errors, and demo assets are specified as ready builds (20, 21, 19) on [build.md](build.md).

## Out of scope

- Production implementation is outside this decision map; it is tracked on [build.md](build.md).
- A hosted proprietary vault or Sopsdeck cloud source of truth is outside the product boundary.
- Shared passwords or shared private keys (Dotenvx `.env.keys` / `DOTENV_PRIVATE_KEY` distribution) as the access model are out of scope.
- Dotenvx ciphertext and private-key compatibility are out of scope unless a later ticket explicitly reopens them.
