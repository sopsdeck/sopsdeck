# Changelog

All notable user-facing changes are listed here. Versioning is [Epoch SemVer](docs/versioning.md) (still `MAJOR.MINOR.PATCH` on the wire).

## Unreleased

### Added

- Standard SOPS commands, flags, and file arguments pass through to the installed `sops` CLI on PATH. Encryption, decryption, editing, key updates, and command execution preserve arguments, stdin, output, and exit codes. `sopsdeck sops ...` exposes upstream help and version commands; Sopsdeck's own commands and browser editor continue using embedded SOPS.
- Dotenv values support `$VAR` and `${VAR}` references. Values resolve in file order, using file keys before the process environment; missing references become empty strings. Escaped dollar signs and single-quoted values stay literal.
- Dotenv values support `$(command)` substitution through POSIX `sh`, with earlier file keys available to the command. Each command has a 30-second limit; failures stop resolution. Windows requires a compatible `sh.exe` on `PATH`. Resolving values can execute stored commands when reading a key, opening the editor, running a command, or syncing secrets. Escape the dollar sign or single-quote the value to keep it literal.
- `sopsdeck run` injects dotenv values into the child environment. For locked structured Managed Files such as `eas.json` and Compose YAML, it temporarily writes plaintext at the file's real path and restores the original encrypted bytes and permissions after the child exits or after SIGINT/SIGTERM. Sopsdeck writes and another run of the same file are refused while active. If the child changes the file, run exits unsuccessfully and leaves the change for inspection and recovery. SIGKILL and system crashes cannot run cleanup.
- Child stdout and stderr from `sopsdeck run` mask nonempty protected values as `[sopsdeck:KEY]`, including short values. A Managed File's `public_keys` list opts values out of output redaction; `--no-redact` disables masking and also allows running without a Managed File.
- `scan` blocks staged Age identities and plaintext protected values from Managed Files. Its hook installer resolves Git's active hooks path, preserves existing hooks, and supports removing an unchanged Sopsdeck hook with `scan --uninstall`.
- Project file lists refresh automatically while the browser tab is visible. Refresh files checks immediately, and the Project panel reports unmanaged files created outside Sopsdeck.
- JSON/YAML path pickers support dragging across fields and selecting nested object groups, with partial-selection indicators. The same picker is used during Project setup and when editing encryption paths.
- The npm launcher streams native runner downloads to disk, shows size and progress, stops downloads that receive no data for 30 seconds, and removes incomplete downloads.
- `bun run site:screenshots` regenerates the landing page's editor, rename, and unused-secret screenshots from a fresh fictional demo.
- GitHub links are available on the website and in the local workspace.
- The repository includes the Apache License 2.0.

### Changed

- `sopsdeck sync -f FILE` replaces Sopsdeck's former `publish` workflow and writes selected values to configured GitHub Actions targets immediately. The dry-run step and `--yes` flag are gone; `--mapping` prints target configuration without sending secrets. `sopsdeck publish` now selects the upstream SOPS command.
- Secret Sync tracks previously synced names in the manifest's `synced` field instead of `published`. Rename existing `published` lists to `synced` to retain pruning history.
- Managed File selection accepts regular files inside the Project regardless of filename, including `.en` and `eas.json`. Sopsdeck detects the content format when possible and records it in the manifest. Unlock `eas.json` before running EAS directly, or use `sopsdeck run`.
- The structured editor shows encrypted fields only. Choose encryption paths in the picker beside the Path heading; file actions use compact header icons, and per-row padlocks and Add folder path are removed.
- The landing page highlights key renaming and unused secrets with focused product screenshots, compares Sopsdeck's workflows with SOPS, and explains encryption's protections and limits with primary sources. The layout adapts to small screens and respects reduced-motion settings.
- The public changelog shows published releases only. Work in progress remains under Unreleased in this file.
- CLI, Access, and Secret Sync documentation reflects the current commands and recipient labels. Product recordings match the current browser workflow.

### Fixed

- Missing or unsafe manifest entries show recoverable warnings without blocking valid files. Stale entries can be removed without deleting files, and Projects with only missing files remain open for recovery. Manifest saves are atomic; malformed manifests can be fixed on disk and retried without reinitializing the Project or replacing its identity.
- Encrypt & save refreshes the lock and saved state immediately, clears unsaved changes, and relocks plaintext files.
- Changing encryption paths preserves existing SOPS recipients and key groups. Unlocking records Age recipients for relocking and refuses operations that cannot preserve non-Age, grouped, or unmanaged multi-recipient Access.
- Account details and private-key backups respect explicit `SOPS_AGE_KEY`, `SOPS_AGE_KEY_FILE`, and `SOPS_AGE_KEY_CMD` identities before falling back to the OS keychain. Account also loads without an open Project or when a missing file prevents the Project from opening.
- Project folders resolve to canonical paths. Initialized monorepo subfolders reopen their Project; discovery, reference scans, and renames keep nested repositories and independent Projects isolated. Files outside the Project and symlinked files cannot be imported.
- Git history and restore resolve nested Managed Files relative to the repository root. Folders outside Git remain editable, with history controls disabled.
- Nested JSON/YAML path selections include all descendant fields and keep parent checkbox states in sync. The Path header renders its encryption control correctly.
- Dialogs dismiss on outside clicks and Escape and scroll within their rounded edges. Dismissing a save preview cancels the pending save; closing Account clears its private-key backup, including delayed responses.
- Copy and other button actions confirm inside the clicked button with a brief check animation and report failed copies there.
- Site text preserves spaces around links split across source lines. Documentation keeps inline code and link URLs literal instead of treating underscores or Markdown characters inside them as formatting.
- The npm launcher lists supported CLI commands, accepts `version`, reports unknown commands or missing Project folders clearly, and rejects occupied browser ports before starting the runner. Upstream SOPS stderr is passed through without saving it to Sopsdeck's error log.
- Team startup refuses occupied ports before seeding or resetting identities, waits for both instances, and stops its servers on exit. `./scripts/dev --team --reset` refuses workspace roots, home directories, symlinks, and unrecognized studios.
- The local browser API rejects mismatched Host and Origin headers, non-JSON commands, and browser preflights. Same-origin UI requests and local clients without an Origin header remain supported.
- Failed CLI and API diagnostics now store only an exit summary. Child stderr is streamed to the caller and is not retained in `errors.json`.
- Dotenv references resolve in file order, so forward and self references no longer see unresolved values. A command's generated `$(...)` text is not executed as a second expansion, and failed or timed-out substitutions now stop reads instead of returning the original expression.
- Playwright test output is isolated from the persistent Alice/Bob studio, so browser tests no longer remove its keys and checkouts.
- Changed and unused labels have a space between them.

### Removed

- Project owner roles and owner-only Access restrictions. Anyone who can decrypt a Managed File can manage its Recipients; names from older owner entries remain as recipient labels.
- Automatic clipboard reads and prompts when the browser regains focus. Explicit paste and copy still work.
- The MCP server and AI tools, including `sopsdeck mcp` and the `sopsdeck-mcp` skill.
- Sopsdeck's Git commit, pull, and push actions, including `sopsdeck commit` and the old Git `sopsdeck sync`. Saving encrypts locally; use Git directly to share changes. `review`, `history`, and `restore` remain available.
- Obsolete npm command routing and recordings of the removed CLI Git actions.

## 0.2.0 - 2026-09-01

### Added

- Project setup now has loading and recovery states, file filtering, and per-field JSON/YAML selection. Managed files can be added or removed from a Project without deleting the underlying file, and encrypted paths can be edited after setup.
- Account shows a copyable Age private-key backup and can remove the local identity. The CLI adds `identity remove --yes`, and identity creation/import no longer require `SOPSDECK_STATE_DIR`.

### Fixed

- Multiline dotenv values now retain their embedded newlines.

### Changed

- Project owners now control both adding and removing Recipients. Recipient removal re-encrypts the current file while clearly preserving the limits of revocation.
- The landing page development banner and product layout now share one consistent treatment.

## 0.1.2 - 2026-09-01

### Fixed

- Opening `sd` in a project with a lockfile (e.g. `package-lock.json`) crashed the setup tree on the empty-string root key under `packages`. Lockfiles are now excluded from candidates and the key-path tree tolerates empty segments.
- A freshly-created identity could encrypt a file but not decrypt it: the desktop stores the Age key in the OS keychain, but SOPS only reads `SOPS_AGE_KEY*` from the environment. The drive now bridges the keychain identity into `SOPS_AGE_KEY` so owners can open their own files.

## 0.1.1 - 2026-09-01

### Fixed

- Global and `npx` installs printed nothing for any command (including `-h` and `-v`) because the launcher's main-module check did not resolve the npm bin symlink.

## 0.1.0 - 2026-09-01

### Added

- Opening a Managed File without Access shows a recovery panel instead of a raw decrypt error.
- JSON and YAML files open as a tree. Encrypt or leave plaintext per path, including after the file is already managed.
- Account copies your Age public key and an Access request in the modal. The inspector no longer has a Request access button.
- Clipboard prompts remember dismissed payloads so the same snippet does not keep interrupting.
- Public docs are a user guide with a sidebar. Contributor pages (seams, features, assets, glossary, versioning) stay in the repo, not the site.
- The site footer is a large lockup; the landing page has load animations.
- Recipient add accepts a name or git identity (`Name <email>`). Project init records your Git identity in `.sopsdeck.toml` so teammates can see who you are.
- Project owners in `.sopsdeck.toml`: only owners can add Recipients once owners are recorded.
- `npx sopsdeck .` uses a single-Project sidebar without recents or extra folders.
- Clipboard modal: on app focus, a sniffed secret, Age recipient, or absolute path opens a confirm modal — paste into the open Managed File, Grant Access, or open the folder as a Project.
- `sopsdeck rename OLD NEW -f FILE` renames a key and rewrites whole-word references across the project; the editor offers the same cross-file rewrite on Encrypt & save.
- `sopsdeck references -f FILE` lists each key with its reference count and files; `sopsdeck unused -f FILE` lists keys with zero references; the inspector shows an "unused" badge.
- Native Go runners for macOS, Windows, and Linux attach to GitHub Releases; the npm launcher downloads the matching runner.
- Landing install points at the npm package; the hero plays the catalog walkthrough.
- Public site pages now render from Astro and deploy through the Cloudflare adapter, including the roadmap.
- Public site deploys with Wrangler from `site/`.
- Demo seed opens several Projects with nested Managed Files.
- Notes show type tags, group by Added/Fixed/Changed, and platform when a bullet names macOS, Windows, or Linux.
- Nested Project folders collapse; recents reopen a folder from this machine; long lists use Show more.
- Inspector sections collapse; reveal/hide values sits on the Value heading. Add secret is gone (composer remains).
- Failed CLI commands append to `$SOPSDECK_STATE_DIR/errors.json`; repeats increment a count. Messages never include private keys or ciphertext.
- `./scripts/dev` builds a fresh CLI and launches the browser app against it.
- `./scripts/dev --team` shares one Git origin between Alice and Bob worktrees and prints those folders for the terminal.
- `identity create` / `import` store the Age private key in the OS keychain; `SOPS_AGE_KEY_CMD='sopsdeck identity key'` decrypts. Existing `SOPS_AGE_KEY_FILE` still works.
- Editor paste sniffs dotenv, JSON, or YAML and previews key names until Apply paste.
- Local MCP (`sopsdeck mcp`) returns metadata by default; `get_value` needs approval; `run` returns exit status only.
- `set` reads dotenv, JSON, or YAML from stdin as a paste preview; `--yes` writes. Lone values need a KEY.
- `scan` blocks staged cloud keys, private key PEMs, and common tokens; SOPS ciphertext is ignored; `--install` writes an opt-in pre-commit hook.
- Inspector Publish shows repo, environment, prefix, and opt-in prune from `.sopsdeck.toml`.
- Publish uses `GH_TOKEN`, `GITHUB_TOKEN`, or `gh auth token` as the GitHub Authorization bearer.
- Publish encrypts each value with GitHub's Libsodium public key before sending it.
- Publish reads `.sopsdeck.toml` for repo, environment, prefix, keys, and last-published names; prune deletes only names Sopsdeck previously published.
- Markdown in `docs/`, README, and CHANGELOG is linted; `./scripts/scan` runs govulncheck and bun audit. CI runs `./scripts/check`; opt-in hooks require a CHANGELOG bullet on user-facing commits.
- Docs, notes, and living pages share one public site under `site/` (landing, changelog, `site/docs/`).
- Product clips hold for seconds with typed motion; `./scripts/demo --check` fails sub-second videos. CLI casts cover get, set, commit, and Sync.
- `sopsdeck set -f FILE` with no KEY creates an empty encrypted Managed File.
- Editor key rows reveal, copy, rename, and delete from icons; a composer adds `KEY` or `KEY=value`.
- The sidebar can add a Managed File; theme is an icon; panes scroll inside the window.
- Inspector can Grant Access and dry-run or Publish to a Sync Target.
- `recipient remove` drops Access, rotates the data key, and warns that Git history still decrypts.
- `recipient request` opens a metadata-only access PR; `recipient grant` re-encrypts selected or all Managed Files and opens the Access PR.
- Review shows a plaintext semantic diff of uncommitted Managed File keys.
- Secret History lists commits on a Managed File; `get --at` decrypts a revision.
- Restore copies a revision’s values into the worktree and leaves them uncommitted.
- Review of a decryptable merge conflict shows base / ours / theirs for each key.
- Inspector Review, History, and Restore call those CLI commands.
- Product stills, clips, and a studio walkthrough are generated by `./scripts/demo`.
- `sopsdeck --version` matches the npm launcher; What’s new is bundled from this file.

### Fixed

- Plain dotenv files are no longer discovered as Managed Files (they failed to decrypt with parse errors); only SOPS-encrypted dotenv files are, matching the Project Manifest spec.
- The browser app logs failed CLI commands to `~/.config/sopsdeck/errors.json` and uses the keychain Age identity for decryption, so it works without a shell env.
- `get` on a file without Access or on a non-SOPS file now says what to do instead of leaking a raw SOPS parse error.
- Adding a Project path no longer depends on a native folder dialog.
- Opening a large Project no longer blocks the browser: Managed File listing and decryption run in the Go runner, and the walker skips generated/build dirs (`.next`, `build`, `coverage`, `__pycache__`, …).
- Sync, get, and Publish print short recovery copy instead of raw Git or SOPS text.
- The browser shows Sync, commit, and save failures next to those controls, not as a toast.
- `get` of encrypted `eas.json` warns that EAS CLI will not read SOPS ciphertext.
- Browser breadcrumb and inspector show `~/project/file` paths, not temp or `..` paths.

### Changed

- Site nav says Changelog instead of Notes. The public roadmap page is gone.
- The editor lock badge follows Locked / Unlocked instead of a static SOPS encrypted label.
- Cipher seam mark is the site favicon, Open Graph image, and app icon master.
- Changelog and What’s new use the product layout; primary actions have icons.
- Browser app is now the only supported UI; the Tauri/Rust shell and its separate setup are removed.
- Empty Project / Managed File / key states, Sync and save loading, dark mode, and a commit message prefilled from dirty keys.
