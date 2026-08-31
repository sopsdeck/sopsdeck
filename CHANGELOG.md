# Changelog

All notable user-facing changes are listed here. Versioning is [Epoch SemVer](docs/versioning.md) (still `MAJOR.MINOR.PATCH` on the wire).

## Unreleased

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
