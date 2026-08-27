# Desktop shell, core runtime, process boundary, and code-sharing

Research for issue `01-choose-desktop-and-core-architecture`. This note gathers primary-source evidence. It does not set product policy (identity mechanism, Git state machine, threat-model claims, license, or shipping order beyond what the ticket already asked to compare).

Date gathered: 2026-08-27.

## Question

Which desktop shell, core runtime, process boundary, and code-sharing model best satisfy macOS-first delivery, later Windows/Linux support, secure key handling, SOPS/Git integration, CLI reuse, signing, and updates: Electron, Tauri, or another evidence-backed option?

## Method

Claims below follow the source that owns them: Electron, Tauri, Wails, Apple, CNCF/getsops SOPS, age, Sparkle, git-scm / go-git / libgit2, and first-party plugin/crate docs. Secondary blogs are not used as evidence.

## Constraints in scope

From the ticket and the product map (context only): local-first desktop; no required hosted backend; SOPS-encrypted dotenv/JSON/YAML; a CLI the desktop can reuse; macOS first; Windows/Linux later; secure key handling; Git integration; code signing; updates.

Related tickets own policy that this note must not decide: identity/age vs PGP vs KMS (`06`), Git state machine (`07`), threat model (`16`), packaging/support contract (`14`, blocked on this issue).

---

## 1. How SOPS is typically embedded

### Canonical implementation

SOPS is a Go project. It encrypts YAML, JSON, ENV, INI, and BINARY, with AWS KMS, GCP KMS, Azure Key Vault, HuaweiCloud KMS, age, and PGP ([getsops.io docs](https://getsops.io/docs/), [github.com/getsops/sops](https://github.com/getsops/sops)). It is a CNCF Sandbox project; file format is backward compatible on the major version ([docs](https://getsops.io/docs/)).

The package comment on `sops.go` is the library contract:

> This package should not be used directly. Instead, Sops users should install the command line client via `go get -u github.com/getsops/sops/v3/cmd/sops`, or use the decryption helper provided at `github.com/getsops/sops/v3/decrypt`.
>
> We do not guarantee API stability for any package other than `github.com/getsops/sops/v3/decrypt`.

Source: [github.com/getsops/sops/blob/main/sops.go](https://github.com/getsops/sops/blob/main/sops.go), also restated on [pkg.go.dev/github.com/getsops/sops/v3](https://pkg.go.dev/github.com/getsops/sops/v3).

The `decrypt` package is documented as “the external API other Go programs can use to decrypt SOPS files. It is the only package in SOPS with a stable API.” Helpers: `Data`, `DataWithFormat`, `File`; formats include `json`, `yaml`, `ini`, `dotenv`, `binary` ([pkg.go.dev/github.com/getsops/sops/v3/decrypt](https://pkg.go.dev/github.com/getsops/sops/v3/decrypt)).

**Implication:** encrypt, rotate keys, and edit-in-place are CLI-shaped. A product that must write SOPS files has three evidence-backed options:

1. **Bundle and exec the official `sops` CLI** (the path maintainers recommend for non-decrypt use).
2. **Import internal Go packages** (`aes`, `age`, `stores/*`, `cmd/sops/common`, …) and pin the SOPS version. This is what in-tree tools and GitHub issue examples do; it is **not** a stable API.
3. **Use a non-official rewrite** (below). Completeness is then that rewrite’s problem, not getsops’.

### Age key discovery (official SOPS)

SOPS recommends age over PGP when possible. Default identity file on macOS: `$XDG_CONFIG_HOME/sops/age/keys.txt`, else `$HOME/Library/Application Support/sops/age/keys.txt`. Overrides: `SOPS_AGE_KEY_FILE`, `SOPS_AGE_KEY`, **`SOPS_AGE_KEY_CMD`** (command that prints identities; can read `SOPS_AGE_RECIPIENT`) ([getsops.io age identities](https://getsops.io/docs/usage/identities/age/)).

`SOPS_AGE_KEY_CMD` is the first-party hook for “do not leave an age identity as a world-readable file”: a privileged process can pull the identity from Keychain and print it on stdout for SOPS.

SSH-as-age is also first-party (`SOPS_AGE_SSH_PRIVATE_KEY_FILE` / `_CMD`, then `~/.ssh/id_ed25519` / `id_rsa`; password-protected keys are not supported via `_CMD`) ([same page](https://getsops.io/docs/usage/identities/age/)).

### Age itself

age is a Go library and CLI; format spec at [age-encryption.org/v1](https://age-encryption.org/v1). The README names [str4d/rage](https://github.com/str4d/rage) as an interoperable Rust implementation and [FiloSottile/typage](https://github.com/FiloSottile/typage) as TypeScript ([age-encryption.org](https://age-encryption.org) / [github.com/FiloSottile/age](https://github.com/FiloSottile/age)). Age-the-format is not SOPS-the-file-format: rage/typage can do age, not SOPS trees, MACs, or `.sops.yaml` creation rules.

### Rust “SOPS”

[rops](https://github.com/gibbz00/rops) (`crates.io/crates/rops`) is an independent “SOPS alternative in pure rust,” not a getsops deliverable. Its own goals include “full sops encrypted file compatibility” as a goal, and explicitly **do not** currently include ENV, INI, BINARY, PGP, GCP KMS, or Azure Key Vault. YAML/JSON/TOML + age + AWS KMS are marked done ([rops goals](https://gibbz00.github.io/rops/goals.html)). Identical CLI / feature parity with `sops` is listed as a non-goal.

**Implication for a product that must handle dotenv, JSON including `eas.json`, and YAML:** an in-process Rust SOPS is not feature-complete relative to official SOPS as of this research. A Rust desktop host that needs SOPS fidelity should treat official `sops` (Go CLI or Go module) as the crypto engine.

### SOPS security model (historical, still published)

Values encrypted with AES-256-GCM; data keys wrapped by KMS and/or PGP (age is documented in usage, not in that older threat-model page). Stated threats: compromised cloud credentials, compromised PGP keys, weak RSA, AES breaks. Operational requirement from the original authors: secrets encrypted on disk until decrypt on the target; per-value encryption so Git diffs/conflicts stay tractable ([getsops.io security](https://getsops.io/docs/security/)).

---

## 2. Git integration

Three first-party-ish embedding paths:

| Path | Owner | What it is | Gaps that matter here |
| --- | --- | --- | --- |
| Spawn `git` | Git project | Full porcelain, credential helpers, hooks, mergetool, worktrees, LFS, rebase | Process mgmt, PATH, version skew |
| libgit2 | [libgit2.org](https://libgit2.org) | C library “with a focus on having a nice API for use within other programs” ([Pro Git Appendix B](https://git-scm.com/book/en/v2/Appendix-B:-Embedding-Git-in-your-Applications-Libgit2)) | C/CGO, not a CLI replacement |
| go-git | [go-git/go-git](https://github.com/go-git/go-git) | Pure Go; “aims to be fully compatible” but documents gaps | **`rebase` ❌, `mergetool` ❌, `stash` ❌, `worktree` ❌, `lfs` ❌, `pull`/`merge` fast-forward only** ([COMPATIBILITY.md](https://raw.githubusercontent.com/go-git/go-git/master/COMPATIBILITY.md)) |

Pro Git also documents go-git as a pure-Go option with no native deps ([Appendix B: go-git](https://git-scm.com/book/en/v2/Appendix-B:-Embedding-Git-in-your-Applications-go-git)).

**Implication:** a desktop that must rebase, use extra worktrees as separate Projects (domain glossary), or decrypt three-way conflicts will hit go-git’s documented non-goals. The evidence-backed default for Git *writes* that must match user Git is **exec the `git` the user already has**, from the privileged process, with arguments constructed internally (never from unsanitized UI strings). go-git/libgit2 remain reasonable for read-only history / encrypted diffs if those operations stay within documented support.

---

## 3. OS keychain access

### Apple

Keychain Services: store secrets with `SecItemAdd`; generic passwords via `kSecClassGenericPassword` when internet-password attributes are not needed ([Adding a password to the keychain](https://developer.apple.com/documentation/Security/adding-a-password-to-the-keychain), [kSecClassGenericPassword](https://developer.apple.com/documentation/security/ksecclassgenericpassword)).

Notarization (Gatekeeper): Developer ID signature, **Hardened Runtime**, secure timestamp, no `get-task-allow=true`; `notarytool` + `stapler`. Beginning macOS 10.15, Developer ID software built after 2019-06-01 must be notarized ([Notarizing macOS software](https://developer.apple.com/documentation/security/notarizing-macos-software-before-distribution)). Hardened Runtime is required to upload for notarization; JIT and related capabilities need explicit entitlements ([Hardened Runtime](https://developer.apple.com/documentation/security/hardened-runtime)).

### Electron

`safeStorage` encrypts strings using OS cryptography. **macOS:** keys in Keychain Access; other apps cannot load them without user override. **Windows:** DPAPI; protected from other *users*, **not** from other apps in the same userspace. **Linux:** kwallet / gnome-libsecret / portal; if none, `basic_text` uses a hardcoded password. Sync APIs can block the thread for Keychain prompts. **Code signing is required** for consistent Keychain identity across updates ([safeStorage](https://www.electronjs.org/docs/latest/api/safe-storage)).

### Tauri 2

There is **no first-party OS-keychain plugin** in the current Tauri 2 plugin set that this research could treat as supported. The official Stronghold plugin is **deprecated** and “will not be available in Tauri v3”; maintainers say the upstream crate is unmaintained and the planned replacement is OS keychains. They “cannot vouch” for community keychain plugins ([tauri-apps/plugins-workspace#3494](https://github.com/tauri-apps/plugins-workspace/issues/3494)).

Rust apps can still call Keychain via the [keyring](https://crates.io/crates/keyring) / [keyring-core](https://docs.rs/crate/keyring-core/latest) ecosystem and [apple-native-keyring-store](https://crates.io/crates/apple-native-keyring-store) (login Keychain vs data-protection store; the latter needs a provisioning profile). That is first-party to those crates, not to Tauri.

### Go (CLI or Wails host)

[zalando/go-keyring](https://pkg.go.dev/github.com/zalando/go-keyring): set/get/delete on macOS, Linux/BSD (Secret Service), Windows. **macOS implementation shells out to `/usr/bin/security`**, not Security.framework in-process. Documented size limits (macOS combined service+user+password ~3000 bytes).

### SOPS hook

Regardless of shell, `SOPS_AGE_KEY_CMD` can be a small helper that only the privileged process can run, reading Keychain and writing the identity to SOPS stdin/stdout — never to the WebView.

---

## 4. Desktop shells

### 4.1 Electron

**Process model.** Chromium multi-process: one **main** process (Node.js, OS APIs, window lifecycle) and a **renderer** per `BrowserWindow`. Renderer is web-standard; no `require` by default. Preload + `contextBridge` is the intended privileged bridge. `utilityProcess` runs Node in a Chromium child (alternative to `child_process.fork`) ([Process Model](https://www.electronjs.org/docs/latest/tutorial/process-model)).

**Sandbox.** From Electron 20, renderer sandbox is on by default. Sandboxed renderer has no Node; privileged work goes through IPC. `nodeIntegration: true` **disables** the sandbox. `--no-sandbox` is testing-only ([Process Sandboxing](https://www.electronjs.org/docs/latest/tutorial/sandbox)).

**Security posture (vendor).** “Electron is not a web browser.” Electron states it cannot match Chromium’s security: no equivalent staffing, Safe Browsing / Certificate Transparency disabled, many apps with different behavior, **vendors must ship Electron upgrades** for Chromium fixes (backports not guaranteed). Checklist includes: no Node in remote content, context isolation (default since 12), sandbox, CSP, no `webSecurity` disable, validate IPC `sender`, prefer custom protocols over `file://`, fuses ([Security](https://www.electronjs.org/docs/latest/tutorial/security)).

**SOPS / Git / CLI.** Main (or utility) process can `child_process` a bundled `sops` / `sopsdeck` / `git`. Renderer must not spawn. Extra binaries are a packager `extraResource` concern (Forge packager config), not a first-class “sidecar” permission DSL.

**Keychain.** `safeStorage` as above; signing required.

**Signing / notarization.** Official path: Electron Forge / `@electron/osx-sign` / `@electron/notarize`; Windows via `@electron/windows-sign`. APIs that **require** a consistent code signature: `safeStorage`, login items, cookieEncryption fuse, **`autoUpdater` (Squirrel.Mac will not work unsigned)** ([Code Signing](https://www.electronjs.org/docs/latest/tutorial/code-signing)).

**Updates.** Official: Squirrel + `autoUpdater`. Static storage (`releases.json` on macOS, `RELEASES` on Windows), `update.electronjs.org` (public GitHub Releases, **macOS or Windows**, macOS must be signed), or self-hosted Hazel/Nuts/etc. Documented server spec covers **Windows and macOS only** ([Updating Applications](https://www.electronjs.org/docs/latest/tutorial/updates)). Linux is not in that official module.

**Win/Linux later.** First-class; ships Chromium so UI is consistent. Cost: Chromium+Node TCB and update cadence.

**Attack surface for a secrets app.** Largest of the options: bundled Chromium + Node + npm. XSS in the renderer is contained only if sandbox + isolation + a tiny preload API hold. Electron’s own docs treat untrusted content as “somewhat uncharted” relative to Chrome.

### 4.2 Tauri 2

**Process model.** One **Core** (Rust, “the only component with full access to the operating system”), plus OS **WebView** processes. IPC is routed through Core so it can intercept/filter. “Never handle secrets in the Frontend”; put business logic in Core. WebViews are **not bundled**: WebView2 / WKWebView / webkitgtk, dynamically linked ([Process Model](https://v2.tauri.app/concept/process-model/)).

**Trust boundary.** Rust core/plugins: unconstrained OS access. WebView: only what **capabilities** allow. Capabilities can reduce frontend compromise and privilege escalation; they do **not** protect against malicious Rust, lax scopes, WebView 0-days, or supply-chain on the developer machine ([Capabilities](https://v2.tauri.app/security/capabilities/), [Security](https://v2.tauri.app/security/)).

**IPC.** Asynchronous message passing; Core may discard messages. Commands are JSON-RPC-like `invoke`, not raw FFI ([IPC](https://v2.tauri.app/concept/inter-process-communication/)).

**Isolation pattern.** Optional (recommended) sandboxed `<iframe>` that intercepts **all** frontend IPC, may validate, then AES-GCM-encrypts with a **per-launch** key before Core decrypts ([Isolation Pattern](https://v2.tauri.app/concept/inter-process-communication/isolation/)).

**SOPS / Git / CLI — sidecar.** First-party: `bundle.externalBin` embeds a helper; platform triples required (`my-sidecar-aarch64-apple-darwin`, …). Rust runs it via `app.shell().sidecar(...)`. JS may run it only if capabilities grant `shell:allow-execute` with `sidecar: true` and **argument allow-lists** (`true` = any args — inappropriate for a secrets CLI) ([Embedding External Binaries](https://v2.tauri.app/develop/sidecar/)).

This is the documented way to reuse a `sopsdeck` CLI: **the desktop bundles the same binary the user also installs as CLI**, and only the Rust core is allowed to spawn it.

**Keychain.** See §3: implement in Rust or in the Go sidecar; do not rely on Stronghold.

**Signing / notarization.** First-party bundler: `APPLE_SIGNING_IDENTITY` / `bundle.macOS.signingIdentity`; notarization via App Store Connect API (`APPLE_API_KEY`, `APPLE_API_ISSUER`, `APPLE_API_KEY_PATH`) or Apple ID env vars ([macOS Code Signing](https://v2.tauri.app/distribute/sign/macos/), [environment variables](https://github.com/tauri-apps/tauri-docs/blob/v2/src/content/docs/reference/environment-variables.mdx)). Ad-hoc `-` is documented for local ARM runs, not distribution.

**Updates.** Official plugin: **signature cannot be disabled**. Static JSON (S3/GitHub Releases) or dynamic server; TLS required in production. Artifacts for linux AppImage, macOS `.app.tar.gz`, Windows MSI/NSIS. Endpoints can be static files — no Tauri-hosted backend required ([Updater](https://v2.tauri.app/plugin/updater/)).

**(Not) bundling WebViews.** Tauri argues OS WebView patches reach users faster than app-vendors shipping Chromium; trade-off is OS WebView variance and 0-days outside the app’s update ([Security](https://v2.tauri.app/security/)).

**Win/Linux later.** Same shell; expect WKWebView vs WebView2 vs webkitgtk differences.

**Attack surface.** Smaller binary TCB than Electron (no shipped Chromium/Node). Remaining: OS WebView, IPC allow-list mistakes, sidecar `args: true`, putting secrets in JS.

### 4.3 Wails (Go + OS WebView)

**What it is.** “Write desktop apps using Go and web technologies”; “lightweight and fast Electron alternative for Go.” Windows, macOS, Linux; does **not** embed a browser — WebView2 on Windows, native engine elsewhere. Go methods are bound into JS; production build is a single executable with assets bundled. “Applications built with Wails are Apple & Microsoft Store compliant” ([Introduction](https://wails.io/docs/introduction/)). Version at fetch: v2.15.0 docs. **v2 is the current stable release; v3 is beta** ([Wails v3 Beta](https://v3.wails.io/blog/wails-v3-beta/)).

**Process / security.** Bindings are Go functions called from the WebView. v2 added origin verification for bindings (`BindingsAllowedOrigins`); a 2026 origin-wildcard bypass was fixed in-tree ([wails#4480](https://github.com/wailsapp/wails/pull/4480), [GHSA-47hv-j4px-h3c9 fix](https://github.com/wailsapp/wails/commit/bcc5941b72fdab6a13942b58871a2696faa85b81)). There is **no** Tauri-style capability file or Chromium renderer sandbox. Compromise of the WebView is a compromise of whatever Go methods were bound — typically the whole app if SOPS lives in-process.

**SOPS / CLI reuse.** Best language fit: import `github.com/getsops/sops/v3/decrypt` (and pinned internals or exec `sops`) in the same module as `sopsdeck`. Desktop and CLI share packages without a sidecar. That is also the blast-radius: decrypt runs in the Wails process.

**Keychain.** `go-keyring` in the Go host (§3).

**Signing / updates.** v2 introduction covers packaging, not a first-party updater. Wails v3 documents Windows `wails3 sign` and packaging ([Windows Packaging](https://v3.wails.io/guides/build/windows/)); v3 updater pages were not retrievable here (CDN challenge). Treating **v3 auto-update as beta-only** is the conservative reading of the v3 beta post.

**Win/Linux later.** Same as Tauri: OS webviews.

**Attack surface.** OS WebView + in-process Go. Smaller than Electron; weaker documented IPC least-privilege than Tauri 2.

### 4.4 Native Swift UI + shared Go (or Rust) core

Not a named framework. Pattern: SwiftUI (or AppKit) talks to a Go static library / XPC helper / bundled `sopsdeck` CLI; later Windows/Linux get a second UI (Tauri/Wails/Electron) on the same core.

**Process isolation.** No app WebView ⇒ no XSS-to-RCE class from UI HTML/JS. Optional WKWebView for rich editors reintroduces a WebView TCB. An XPC helper can hold keys with a tighter sandbox than the UI (Apple process model; not unique to this product).

**Keychain / signing / updates.** First-party Apple Keychain, Xcode Developer ID, notarization, Hardened Runtime. Updates: [Sparkle](https://sparkle-project.org/documentation/) (HTTPS appcast, Ed25519 signatures, notarized archives, optional signed feeds as of Sparkle 2.9). Sparkle is macOS-only.

**SOPS / CLI.** Same Go core as a CLI. FFI/CGO or sidecar.

**Win/Linux later.** **Not** a free UI. The core/CLI can ship; the desktop shell is rewritten or a web shell is added later.

**Attack surface.** Smallest macOS UI TCB if the UI stays native. Highest macOS-specific engineering cost.

---

## 5. Process-boundary patterns (shell-agnostic)

All three web shells document the same rule in different words: **the UI process is untrusted; secrets and crypto belong in a privileged process.**

Recommended layering for this product class (evidence synthesis, not policy):

```
┌─────────────────────────────────────────────────────────┐
│ WebView / Swift UI  —  no keys, no plaintext persist,   │
│                         no spawn, no raw paths          │
└───────────────────────────┬─────────────────────────────┘
                            │ IPC / bindings (allow-listed)
┌───────────────────────────▼─────────────────────────────┐
│ Desktop host (Electron main / Tauri Core / Wails Go /   │
│ Swift) — windowing, updater, maybe keychain             │
└───────────────────────────┬─────────────────────────────┘
                            │ exec or in-process API
┌───────────────────────────▼─────────────────────────────┐
│ sopsdeck core (Go)  —  SOPS encrypt/decrypt, Git argv,  │
│                         age identity via Keychain hook, │
│                         also the public CLI             │
└─────────────────────────────────────────────────────────┘
```

**Do not** expose `sops`/`git` as a generic shell from JS. Tauri’s `args: true` and Electron `nodeIntegration` are the documented ways to destroy this boundary.

Electron `utilityProcess` can isolate crash-prone native work from main; it is still Node, still in the app TCB ([utilityProcess](https://www.electronjs.org/docs/latest/api/utility-process)).

---

## 6. Code-sharing models

| Model | How CLI and desktop share | Fidelity to official SOPS | Cost |
| --- | --- | --- | --- |
| **A. Go module is the product** | `sopsdeck` CLI and Wails (or a C-exported lib) import the same packages | Highest (in-process decrypt API; encrypt via pinned internals or exec `sops`) | Two languages if UI is not Wails |
| **B. CLI binary is the product; desktop sidecars it** | One Go binary, invoked with a stable argv/JSON protocol | Highest if that binary wraps official SOPS; desktop cannot drift | Extra process; must sign the sidecar; Tauri documents this |
| **C. Rust core reimplements SOPS** | `sopsdeck` in Rust + Tauri in-process | Incomplete today (`rops` gaps: ENV, PGP, …) | Divergence from getsops |
| **D. TypeScript core + exec `sops`** | Electron main and a Node CLI both spawn `sops` | As good as the bundled `sops` version | Two stacks; npm TCB; encrypt/decrypt UX must shell out |

Ticket language “a CLI that can be reused by the desktop app” matches **B** literally and **A** as a library. **C** is not evidence-backed for dotenv+SOPS parity yet. **D** is how many Electron secret-adjacent tools work, at Chromium cost.

---

## 7. Signing, notarization, updates (cross-cut)

All distribution paths on modern macOS need Developer ID + notarization + Hardened Runtime ([Apple](https://developer.apple.com/documentation/security/notarizing-macos-software-before-distribution)). WebView/Chromium hosts typically need `com.apple.security.cs.allow-jit` (Electron/WKWebView).

| | Electron | Tauri 2 | Wails v2 | Native Swift |
| --- | --- | --- | --- | --- |
| macOS sign/notarize tooling | Forge / osx-sign / notarize | Bundler + Apple env | CLI packaging; Store-compliant claim | Xcode / notarytool |
| Auto-update (official) | Squirrel Mac/Win; **not Linux** in `autoUpdater` | All three desktop OSes; **signatures required**; static JSON OK | None first-party in v2 docs | Sparkle (macOS) |
| No hosted backend | Static S3/GitHub + Squirrel metadata | Static `latest.json` | Roll your own | Static appcast |

A product with “no required hosted backend” still needs **some** HTTPS place to put artifacts (GitHub Releases, S3, own site). That is hosting of files, not a Sopsdeck control plane.

---

## 8. Comparison against the ticket criteria

| Criterion | Electron | Tauri 2 | Wails v2 | Native Swift + Go core |
| --- | --- | --- | --- | --- |
| macOS-first delivery | Mature | Mature | Mature | Fastest native fit; Sparkle |
| Windows/Linux later | Strongest UI parity (ship Chromium) | Same app, OS WebView diffs | Same as Tauri | Second UI required |
| Process isolation | Chromium sandbox + IPC; vendor-limited vs Chrome | Core vs WebView + capabilities + optional isolation iframe | Origin-checked bindings; no capability DSL | No WebView if UI is native |
| OS keychain | First-party `safeStorage` (macOS Keychain; Win DPAPI weaker) | DIY Rust/Go; Stronghold deprecated | `go-keyring` (`security` CLI on macOS) | Security.framework |
| Embed / exec SOPS | extraResource + main process | Official sidecar | In-process Go module | Same as Wails or sidecar |
| Git | `child_process` `git` | sidecar/`plugin-shell` from Core | `os/exec` or go-git | `Process` or go-git |
| Share core with CLI | extraResource the CLI, or Node CLI | **Sidecar of the same CLI** | Same Go module | Same Go module |
| Signing / notarization | Documented; required for Keychain + Squirrel | Documented bundler | Claimed Store-compliant; less updater docs | First-party Xcode |
| Updates | Official Mac/Win; Linux elsewhere | Official 3-OS, signed, static JSON | v2: custom; v3 beta | Sparkle |
| Secrets-app attack surface | Highest TCB; must stay on latest Electron | Smaller TCB; OS WebView 0-days | OS WebView + in-process crypto | Lowest if no WebView |

---

## 9. Trade-offs (not a founder decision)

These are engineering consequences of the sources above. Identity, Git UX, and threat-model *claims* remain other tickets.

- **Putting SOPS in-process (Wails / Go lib)** maximizes format fidelity and avoids parsing `sops` CLI output, but a WebView XSS that reaches a bound `Decrypt` method dumps plaintext in the same process. **Sidecaring the CLI (Tauri / Electron main)** keeps a process boundary and makes “CLI reuse” literal; it costs IPC design and signed helper binaries.
- **Tauri capabilities** are the only first-party, config-file least-privilege story for “frontend must not spawn git/sops.” Electron can do the same in code (tiny preload) but has no equivalent static allow-list. Wails defaults to exposing bound methods.
- **Bundled Chromium (Electron)** buys UI consistency and a well-understood sandbox, at the cost Electron itself states: you *are* the Chromium vendor for your users.
- **OS WebView (Tauri/Wails)** shrinks the binary and shifts patching to the OS, at the cost of WebView variance and 0-days you cannot patch except by waiting on Apple/Microsoft/GTK.
- **rops-in-Tauri** looks elegant until ENV/dotenv and PGP/KMS matter; the rops book says those are unfinished.
- **Native Swift** wins macOS keychain, notarization, and TCB, and loses “one desktop codebase” for Windows/Linux.
- **go-git as the only Git** will surprise users on rebase, extra worktrees, and LFS — all documented absences.

---

## 10. Recommendation (evidence-backed, not product policy)

The combination that **best satisfies the listed constraints together** is:

**A Go `sopsdeck` core that is also the public CLI**, using official SOPS (`decrypt` plus either the `sops` binary or version-pinned internals for encrypt), talking to the user’s `git` for porcelain that libraries do not implement, and obtaining age identities from the OS keychain via `SOPS_AGE_KEY_CMD` or `go-keyring`/`security` — **never from the UI process**.

**Desktop shell: Tauri 2**, with that CLI as a **sidecar spawned only from Rust**, capabilities that do not grant the WebView arbitrary `shell:allow-execute`, isolation pattern enabled, secrets never in frontend state. Signing/notarization through Tauri’s macOS bundler + Apple notary. Updates through the official updater with required signatures and a **static JSON** feed (GitHub Releases or object storage), which does not imply a Sopsdeck backend. Windows/Linux later keep the same shell.

Why this pairing rather than Wails-in-process or Electron:

- **SOPS is Go and the stable library is decrypt-only** — a Go CLI as the engine avoids a Rust rewrite (`rops` incomplete) and avoids reimplementing crypto in Node.
- **“CLI reused by the desktop”** is Tauri’s documented sidecar feature, with argument allow-lists the frontend cannot widen at runtime.
- **Capability IPC + isolation iframe** is the strongest *documented* frontend-to-core control among the web shells.
- **Updater signatures on three OSes with static files** match “no required hosted backend” better than Electron’s official Squirrel path (Linux absent) or Wails v2 (no first-party updater).
- **Attack surface** is smaller than Electron’s Chromium+Node TCB for a secrets product, while still giving a cross-platform UI when Windows/Linux ship.

**If a later decision prefers one language over that isolation story:** Wails v2 + the same Go core is the runner-up (in-process SOPS, weaker IPC ACL, updater TBD until v3 is stable).

**If a later decision prefers eliminating the WebView on macOS:** Swift UI + the same Go core + Sparkle is the runner-up (best Keychain/notarization, second UI later).

**Electron** remains a viable delivery vehicle (`safeStorage`, Forge signing, Squirrel) when Chromium consistency and ecosystem size outweigh TCB; it is the weakest fit *specifically* for a local secrets app’s attack surface and for official Linux auto-update.

This recommendation does **not** choose age vs PGP vs KMS, Keychain vs file identities, Git “sync” naming, or macOS-only vs multi-OS v1 — those are other issues.

---

## 11. Uncertainties

- **SOPS encrypt via Go:** only `decrypt` is stable. Encrypt-in-process may break across SOPS minor versions; wrapping the official CLI is the supported alternative. Uncertain which ops a `sopsdeck` CLI would need beyond decrypt (owned in part by the Dotenvx-contract ticket).
- **Tauri Stronghold replacement:** OS-keychain plugin is “current plan,” not shipped. Until then, keychain is a crate/sidecar concern, not a Tauri API.
- **Wails v3:** desktop API described as stable by authors, still labeled beta; updater/signing docs were not fully retrievable (CDN). Using v3 for a secrets product would be betting on a beta.
- **WKWebView vs Chromium sandbox:** Tauri/Wails inherit Apple/Microsoft/GTK WebView sandboxes; this research did not compare them to Chromium’s sandbox design doc in depth. Treat as “different TCB,” not “weaker/stronger” without a specific CVE class.
- **go-keyring vs Security.framework:** shelling out to `security` vs in-process SecItem; ACL/prompt behavior and data-protection vs login keychain differences are easy to get wrong (Apple forums; not fully mapped here).
- **Hardened Runtime + JIT:** Electron and WKWebView typically need `allow-jit`. Entitlement set for a bundled `sops`/`git` sidecar was not verified end-to-end in this note.
- **Linux secret stores:** Electron documents `basic_text` fallback (effectively unprotected). Tauri/Wails/Go inherit Secret Service availability. “Secure key handling on Linux” is environment-dependent.
- **Windows DPAPI:** Electron `safeStorage` does not hide secrets from other apps on the same user. A secrets app may want a different Windows primitive; not researched beyond that vendor statement.
- **Git credential helpers / signing:** spawning `git` uses the user’s helper and `gpg`/`ssh`; libraries often do not. Exact interaction with SOPS-encrypted files in merge conflict is a Git-lifecycle ticket.
- **rops** may gain ENV support after this date; check [goals](https://gibbz00.github.io/rops/goals.html) before revisiting a Rust-only core.
- **Electron Linux updates:** third-party tools (e.g. electron-builder) exist; they are not the official `autoUpdater` module and were not evaluated.

---

## 12. Source list

- Electron: [process model](https://www.electronjs.org/docs/latest/tutorial/process-model), [sandbox](https://www.electronjs.org/docs/latest/tutorial/sandbox), [security](https://www.electronjs.org/docs/latest/tutorial/security), [safeStorage](https://www.electronjs.org/docs/latest/api/safe-storage), [updates](https://www.electronjs.org/docs/latest/tutorial/updates), [code signing](https://www.electronjs.org/docs/latest/tutorial/code-signing), [utilityProcess](https://www.electronjs.org/docs/latest/api/utility-process)
- Tauri 2: [process model](https://v2.tauri.app/concept/process-model/), [security](https://v2.tauri.app/security/), [capabilities](https://v2.tauri.app/security/capabilities/), [IPC](https://v2.tauri.app/concept/inter-process-communication/), [isolation](https://v2.tauri.app/concept/inter-process-communication/isolation/), [sidecar](https://v2.tauri.app/develop/sidecar/), [updater](https://v2.tauri.app/plugin/updater/), [macOS signing](https://v2.tauri.app/distribute/sign/macos/), [Stronghold deprecation](https://github.com/tauri-apps/plugins-workspace/issues/3494)
- Wails: [introduction](https://wails.io/docs/introduction/), [v3 beta](https://v3.wails.io/blog/wails-v3-beta/), [Windows packaging](https://v3.wails.io/guides/build/windows/)
- SOPS: [docs](https://getsops.io/docs/), [security](https://getsops.io/docs/security/), [age identities](https://getsops.io/docs/usage/identities/age/), [sops.go contract](https://github.com/getsops/sops/blob/main/sops.go), [decrypt API](https://pkg.go.dev/github.com/getsops/sops/v3/decrypt)
- age: [age-encryption.org](https://age-encryption.org)
- rops: [goals](https://gibbz00.github.io/rops/goals.html), [github.com/gibbz00/rops](https://github.com/gibbz00/rops)
- Apple: [notarization](https://developer.apple.com/documentation/security/notarizing-macos-software-before-distribution), [Hardened Runtime](https://developer.apple.com/documentation/security/hardened-runtime), [adding a password](https://developer.apple.com/documentation/Security/adding-a-password-to-the-keychain)
- Sparkle: [documentation](https://sparkle-project.org/documentation/)
- Git: [Pro Git libgit2](https://git-scm.com/book/en/v2/Appendix-B:-Embedding-Git-in-your-Applications-Libgit2), [Pro Git go-git](https://git-scm.com/book/en/v2/Appendix-B:-Embedding-Git-in-your-Applications-go-git), [go-git COMPATIBILITY.md](https://raw.githubusercontent.com/go-git/go-git/master/COMPATIBILITY.md)
- Keyring: [go-keyring](https://pkg.go.dev/github.com/zalando/go-keyring), [keyring-core](https://docs.rs/crate/keyring-core/latest)
