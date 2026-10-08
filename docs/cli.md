# CLI

Sopsdeck’s CLI lives in the same local runner as the browser app. Files stay in your project. Age keys stay in the OS keychain.

Run `sopsdeck --help` to list commands, or `sopsdeck .` to open the browser app for the current Project.

Standard SOPS commands and flags pass directly to the `sops` executable on your PATH. Install the [SOPS CLI](https://getsops.io/docs/) to use them. The browser app and Sopsdeck-specific commands still use embedded SOPS.

```bash
sopsdeck encrypt --in-place --age age1... .env.production
sopsdeck decrypt .env.production
sopsdeck decrypt --extract '["build"]["env"]["TOKEN"]' eas.json
sopsdeck set eas.json '["build"]["env"]["TOKEN"]' '"new-value"'
sopsdeck unset eas.json '["build"]["env"]["TOKEN"]'
sopsdeck updatekeys eas.json
sopsdeck exec-env .env.production 'your-command'
sopsdeck --decrypt .env.production
sopsdeck sops --help
```

Arguments, stdin, stdout, stderr, and exit codes are preserved. The local keychain identity is available when `SOPS_AGE_KEY`, `SOPS_AGE_KEY_FILE`, and `SOPS_AGE_KEY_CMD` are unset. Standard SOPS configuration, formats, key providers, and flags work as usual.

Native SOPS stderr is passed through without being saved to Sopsdeck's error log, since commands such as `exec-env` may print secret values.

Sopsdeck adds the following conveniences. `set` with `-f` or `--env-file` uses Sopsdeck's key/value and paste syntax; `set FILE PATH VALUE` uses native SOPS syntax. `sopsdeck sops ...` always selects SOPS, including its help and version commands.

```bash
sopsdeck get KEY -f path/to/.env.production
sopsdeck set KEY VALUE -f path/to/.env.production
sopsdeck del KEY -f path/to/.env.production
sopsdeck lock -f path/to/.env.production
sopsdeck unlock -f path/to/.env.production
```

`get` without a key dumps the file. `--output json` prints every leaf. `--at REV` decrypts a Git revision.

When Sopsdeck resolves dotenv values for `get KEY`, `get --output json`, `run`, the browser editor, or Secret Sync, it expands references in file order: earlier file keys take precedence, then the process environment, and missing names become empty. Single-quoted values and escaped dollar signs stay literal. A `$(...)` value runs through POSIX `sh` with earlier file keys in its environment; each command has a 30-second limit, and a failed or timed-out command stops resolution. On Windows, install a compatible `sh.exe` on `PATH`; Sopsdeck does not provide a shell. Opening a trusted Project can execute its stored commands. For `run`, existing process environment values remain in effect when names overlap.

## Access

```bash
sopsdeck recipient add AGE1... -f FILE --name "Ada <ada@example.com>"
sopsdeck recipient list -f FILE
sopsdeck recipient remove AGE1... -f FILE
sopsdeck recipient request AGE1... --name NAME --all
sopsdeck recipient grant AGE1... --name NAME -f FILE
```

`request` opens a metadata-only PR. `grant` re-encrypts and opens the Access PR. Anyone who can decrypt a file can add or remove its Recipients.

`recipient remove` re-encrypts the current file with a fresh SOPS data key. Run it for every Managed File a departing person could read, then rotate the actual provider credentials they previously knew. It cannot revoke old Git clones, history, or values they already copied.

## Git

Use Git to commit, pull, and push Managed Files and `.sopsdeck.toml`. These commands read Git history and restore values into the working tree:

```bash
sopsdeck review -f FILE
sopsdeck history -f FILE
sopsdeck restore -f FILE --at REV
```

## Project

```bash
sopsdeck project init FOLDER --file eas.json --keys build.env.EXPO_TOKEN
sopsdeck project add FOLDER --file compose.yaml --keys services.db.environment.POSTGRES_PASSWORD
sopsdeck project remove FOLDER --file compose.yaml
sopsdeck project encrypt FILE --keys build.env.EXPO_TOKEN,build.env.SECRET
sopsdeck files FOLDER
```

JSON and YAML encrypt only the paths you pass to `--keys`. Dotenv files encrypt every key.

## Scan

`sopsdeck scan` inspects the Git index, so unstaged worktree changes do not replace staged content in the scan. It blocks high-confidence credentials, Age private identities, and nonempty protected values from plaintext staged Managed Files. `public_keys` and structured fields outside `encrypted_keys` remain allowed; an allowlisted path is skipped. Test-token matches warn without blocking. Diagnostics show paths and key names, never matched values.

`sopsdeck scan --install` creates a pre-commit hook at Git's active hooks path only when one is absent. It preserves an existing hook and prints the manual integration step. It also refuses to write to a hooks path configured outside the Project, which could affect other repositories. The command records `scan.hook` in `.sopsdeck.toml`. Run `sopsdeck scan --uninstall` to remove the unchanged Sopsdeck hook; a modified hook is preserved for manual cleanup. `git commit --no-verify` still bypasses Git hooks.

## Secret Sync

```bash
sopsdeck sync -f FILE
sopsdeck sync -f FILE --mapping
```

`sync` writes selected values to the configured Sync Target immediately. `--mapping` prints the resolved target configuration without sending secrets. Configure mappings in `.sopsdeck.toml` or the browser app. Set `SOPSDECK_GITHUB_API` to the GitHub API root; authentication uses `GH_TOKEN`, `GITHUB_TOKEN`, or `gh auth token`.

## Other commands

```bash
sopsdeck identity create --confirmed-backup
sopsdeck identity key
sopsdeck identity import -f age-identity.txt --confirmed-backup
sopsdeck identity remove --yes
sopsdeck run -f FILE -- your-command
sopsdeck run --no-redact -- your-command
sopsdeck rename OLD NEW -f FILE
sopsdeck unused -f FILE
sopsdeck scan
```

`identity key` prints the Age private key for SOPS; save the entire output in a password manager and never commit it. `identity remove --yes` clears only this machine’s OS-keychain identity; it does not remove the public key from files. `SOPSDECK_STATE_DIR` is optional: when set, failed commands append exit summaries to `$SOPSDECK_STATE_DIR/errors.json`; child stderr is not retained.

`run` masks every nonempty protected value it captures, including short values. Add exact dotenv keys or structured paths to a Managed File's `public_keys` list in `.sopsdeck.toml` when their values should stay visible in command output. Structured parent paths opt out their descendants. This list changes output redaction only; it does not change encryption or Secret Sync. Names such as `PUBLIC`, `VITE_*`, or `_PLAIN` do not opt values out automatically.

For a locked structured file, `run` temporarily writes a mode-`0600` plaintext copy at the Managed File path, then restores the original encrypted bytes and permissions. Sopsdeck writes and another `run` for that path are refused while it is active. If the child changes the file, `run` leaves that change in place and exits unsuccessfully rather than replacing it. A forced kill or machine crash can leave plaintext and a `.sopsdeck-run.lock` file behind. Inspect the file, remove the stale lock (`rm FILE.sopsdeck-run.lock` on macOS/Linux or `Remove-Item FILE.sopsdeck-run.lock` in PowerShell), then recover with `sopsdeck lock -f FILE`; the required Age identity must be available. Commands that require a terminal on stdout may not work interactively through `run` yet.

```toml
[[managed_file]]
path = "eas.json"
public_keys = ["build.env.APP_VERSION"]
```
