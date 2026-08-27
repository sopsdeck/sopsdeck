# Dotenvx-shaped CLI contract (research)

Pinned sources (2026-08-27):

- Dotenvx npm package `@dotenvx/dotenvx` **2.21.0** on `main` ([package.json](https://raw.githubusercontent.com/dotenvx/dotenvx/main/package.json)).
- Official docs: [dotenvx.com/docs/cli](https://dotenvx.com/docs/cli/), [help](https://dotenvx.com/docs/cli/help), [run](https://dotenvx.com/docs/cli/run/), [encryption](https://dotenvx.com/docs/quickstart/encryption).
- CLI source of truth: [`src/cli/dotenvx.js`](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/cli/dotenvx.js).
- README: [github.com/dotenvx/dotenvx](https://github.com/dotenvx/dotenvx/blob/main/README.md).
- SOPS **3.11.0** local binary (`sops --version`) plus [README.rst](https://github.com/getsops/sops/blob/v3.11.0/README.rst), [exec.go](https://raw.githubusercontent.com/getsops/sops/v3.11.0/cmd/sops/subcommand/exec/exec.go), [dotenv store](https://raw.githubusercontent.com/getsops/sops/v3.11.0/stores/dotenv/store.go), [exit codes](https://raw.githubusercontent.com/getsops/sops/v3.11.0/cmd/sops/codes/codes.go).

This note inventories what Dotenvx actually does, where SOPS already occupies the same verbs, and where claiming “the same CLI” would be a false compatibility promise. It does **not** decide Sopsdeck command names, binary aliases, or product policy.

## 1. Compatibility posture

Dotenvx’s public usage string is `dotenvx run -- yourcommand` ([help docs](https://dotenvx.com/docs/cli/help), [dotenvx.js](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/cli/dotenvx.js)). The binary is `dotenvx`; npm also registers `dx` ([package.json `bin`](https://raw.githubusercontent.com/dotenvx/dotenvx/main/package.json)).

SOPS’s analogue is `sops exec-env [file] [command]` — one file, one shell string, not `run -- argv` ([`sops exec-env --help`](https://github.com/getsops/sops/blob/v3.11.0/README.rst), local 3.11.0 help).

A CLI that “feels like dotenvx” can match **workflow verbs and injection semantics**. It cannot match **ciphertext, key files, or `DOTENV_PRIVATE_KEY*`** while remaining SOPS-only. Those are Dotenvx’s encryption product, not its run-a-command UX.

Recommended framing (evidence, not a name decision):

| Claim | Evidence |
| --- | --- |
| Drop-in `dotenvx` / `dx` shim | Would imply ciphertext, `.env.keys`, and `DOTENV_PRIVATE_KEY*` work. They will not under SOPS. |
| “Compatible with dotenvx encrypted files” | False. Format is `KEY="encrypted:…"` plus `DOTENV_PUBLIC_KEY` ([encrypted files](https://dotenvx.com/docs/learn/encrypting/encrypted-files)). SOPS dotenv uses `ENC[AES256_GCM,…]` plus flattened `sops_*` metadata ([dotenv store](https://raw.githubusercontent.com/getsops/sops/v3.11.0/stores/dotenv/store.go)). |
| “Inspired by dotenvx `run` / `get` / `set`” | Accurate if argv, `--` separator, `-f` composition, and first-wins vs `--overload` match. |
| “Same as `sops exec-env`” | Also false as a drop-in: shell string vs argv, default override polarity, one file vs many. |

## 2. Command inventory (Dotenvx 2.21.0)

Advertised top-level commands from [help docs](https://dotenvx.com/docs/cli/help) and [CLI index](https://dotenvx.com/docs/cli/):

| Command | What it does | Key flags / args | Notes |
| --- | --- | --- | --- |
| `run` | Decrypt (if needed) and inject env into a child process | `-e/--env`, `-f/--env-file`, `-fk/--env-keys-file`, `--redact`, `-o/--overload`, `--validate`, `--strict`, `--convention`, `--ignore`, `--token`, `--mask`, `--no-armor`, `--no-native`, `--no-1password`, `--no-bitwarden` | Primary UX. Requires `--` or a heuristic to find the command ([run.js](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/cli/actions/run.js)). |
| `get [KEY]` | Print one value, or all values in a format | Same load flags as run, plus `-a/--all`, `-ik/--include-key`, `-ek/--exclude-key`, `--mask`, `--pretty-print`, `--format` (`json` default, `shell`, `colon`, `eval`, `eval-export`) | Missing key logs `[MISSING_KEY]` ([README](https://github.com/dotenvx/dotenvx/blob/main/README.md)). |
| `set <KEY> [value]` | Write a key; **encryption on by default** | `-f`, `-fk`, `-c/--encrypt` (default true), `-p/--plain`, `--no-create`, `--no-armor`, `--no-native` | Omitting value prompts on TTY ([set.js](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/cli/actions/set.js)). Docs: “encryption defaults to on” ([set KEY value](https://dotenvx.com/docs/cli/set-key-value)). |
| `del <KEY>` | Delete a key from env file(s) | `-f` | Top-level on current CLI; older README snapshots sometimes omit it. |
| `encrypt` / hidden `enc` | Encrypt file(s) in place or stdout | `-f`, `-fk`, `-k/--key`, `-ek/--exclude-key`, `--stdout`, `--token`, `--no-create`, `--no-armor`, `--no-native` | Writes `encrypted:` values and `DOTENV_PUBLIC_KEY`. |
| `decrypt` / hidden `dec` | Decrypt file(s) in place or stdout | `-f`, `-fk`, `-k`, `-ek`, `--stdout`, `--mask`, `--no-armor`, `--no-native` | Inverse of encrypt. |
| `keypair [KEY]` | Print public/private keys | `-f`, `-fk`, `--format` (`json`, `shell`, `colon`), `--pretty-print` | Dotenvx keypair product. |
| `ls [directory]` | Tree of `.env*` files | `-f` glob (default `.env*`), `-ef/--exclude-env-file`, `--json` | Discovery, not encryption. |
| `gitignore` | Append patterns to `.gitignore` | `--pattern` (default `.env*`) | Default would ignore committed SOPS dotenv files if copied blindly. |
| `genexample [directory]` | Generate `.env.example` | `-f` (default `.env`) | Key names only. |
| `validate` | Check loaded env against `.env.example` | Load flags like `run` | Also available as `run --validate`. |
| `precommit [directory]` | Block committing `.env` files | `-i/--install` | Installs `.git/hooks/pre-commit`. |
| `prebuild [directory]` | Block `.env` files in Docker builds | directory | Docker-oriented. |
| `help [command]`, `-V/--version` | Help / version | | |

Hidden / security-product commands (registered in [dotenvx.js](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/cli/dotenvx.js), advertised as “Better Security” / “For Security Teams”):

- `lock` (passphrase-protect private keys), `native` (OS secret store), `armor` (hosted key service), `curl` (Armor API), `login`/`logout` aliases.
- `ext` namespace: `ls`, `genexample`, `gitignore`, `prebuild`, `precommit`, plus **`ext scan`** (leaked-secret scan) which is **not** promoted to top-level.
- `doctor` (scan for dotenv loaders), `update`.

**`rotate` is not on current `main`.** It existed through ~1.57 and was withdrawn from the open-source CLI in favor of Armor; maintainers documented a manual decrypt / strip public key / re-encrypt workaround ([issue #717 comment](https://github.com/dotenvx/dotenvx/issues/717#issuecomment-4937880580)). SOPS still has `sops rotate` (new data key, re-encrypt values; optionally add/remove recipients). Matching a Dotenvx `rotate` verb today would match **removed** Dotenvx behavior, not current Dotenvx.

**There is no `exec` command.** Injection is `run`. SOPS uses `exec-env` / `exec-file`.

Global log flags on the program: `-l/--log-level`, `-q/--quiet`, `-v/--verbose`, `-d/--debug` ([dotenvx.js](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/cli/dotenvx.js)). Quiet can also be `DOTENV_CONFIG_QUIET=true` ([README](https://github.com/dotenvx/dotenvx/blob/main/README.md)).

## 3. Shell semantics for `run -- cmd`

### 3.1 How injection works

1. Parse `run` options; collect `-f` / `-e` into an ordered `envs` list ([dotenvx.js `collectEnvs`](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/cli/dotenvx.js)).
2. Resolve files (`determine` + conventions).
3. Decrypt `encrypted:` values using `.env.keys` and/or `DOTENV_PRIVATE_KEY*` ([encryption docs](https://dotenvx.com/docs/quickstart/encryption)).
4. Expand `${VAR}`, `${VAR:-default}`, `${VAR:+alt}`, and `$(command)` ([interpolation summary](https://dotenvx.com/docs/cli/run-interpolation-syntax-summary), [command substitution](https://dotenvx.com/docs/cli/run-command-substitution)). Single quotes disable expansion ([variable expansion](https://dotenvx.com/docs/cli/run-variable-expansion)).
5. Mutate `process.env` (unless `--mask`), then `execa` the argv with `stdio: inherit` (or piped if `--redact`) ([executeCommand.js](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/lib/helpers/executeCommand.js)).
6. Child env is `{ ...process.env, ...env }` ([executeCommand.js](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/lib/helpers/executeCommand.js)).

The child is **not** launched via `/bin/sh -c`. The first argv token is resolved with `which.sync`; remaining args are passed through. This is argv-preserving and cross-platform via execa ([README “run anywhere” / Windows winget](https://github.com/dotenvx/dotenvx/blob/main/README.md)).

### 3.2 The `--` separator

`run.js` takes `this.args` after `--`. If empty, it scans `process.argv` after `run`: skip `-f/--env-file` values, skip other flags, then treat the rest as the command. If still empty:

- with `--`: `missing command after [dotenvx run --]` → **exit 1**
- without `--`: `ambiguous command due to missing '--' separator` → **exit 1**

([run.js](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/cli/actions/run.js))

PowerShell historically ate `--`; current Dotenvx docs claim 2.0+ works under PowerShell ([issue #434](https://github.com/dotenvx/dotenvx/issues/434)).

### 3.3 POSIX vs Windows vs SOPS

| | Dotenvx `run` | SOPS `exec-env` (3.11.0) |
| --- | --- | --- |
| Command shape | `run [opts] -- argv…` | `exec-env [opts] FILE 'shell string'` |
| Shell | No shell; execa argv | Unix: `/bin/sh -c`; Windows: `cmd.exe /C` ([exec_unix.go](https://raw.githubusercontent.com/getsops/sops/v3.11.0/cmd/sops/subcommand/exec/exec_unix.go), [exec_windows.go](https://raw.githubusercontent.com/getsops/sops/v3.11.0/cmd/sops/subcommand/exec/exec_windows.go)) |
| Variable expansion in the command line | Shell expands **before** dotenvx; docs require `sh -c 'echo $HELLO'` with **single quotes** ([shell expansion](https://dotenvx.com/docs/cli/run-shell-expansion)) | Shell string is expanded **inside** the child shell, so `$VAR` in the quoted command sees injected env (README example `sops exec-env out.json 'echo secret: $database_password'`) |
| stdio | inherit (redact pipes stdout/stderr) | inherit |
| Signals | Forwards SIGINT/SIGTERM (TTY: first SIGINT not forwarded; second → SIGTERM then SIGKILL) ([executeCommand.js](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/lib/helpers/executeCommand.js)) | Child process by default; `--same-process` uses `execve` on Unix only; **not supported on Windows** |
| Extra | `--redact` PTY + log redaction | `--pristine`, `--user` (Unix), `--same-process` |

Matching Dotenvx here means: **argv after `--`, no implicit shell, document `sh -c '…'`**. Matching SOPS here means: **one shell string**. Those two contracts cannot both be “exactly the same.”

### 3.4 Multiline values

Dotenvx keeps quoted multiline PEMs as real newlines in `process.env` ([run multiline](https://dotenvx.com/docs/cli/run-multiline); fixture [`tests/.env.multiline`](https://github.com/dotenvx/dotenvx/blob/main/tests/.env.multiline)).

SOPS dotenv **cannot** store raw newlines: `EmitPlainFile` replaces `\n` with the two-character sequence `\\n` ([store.go](https://raw.githubusercontent.com/getsops/sops/v3.11.0/stores/dotenv/store.go)). A “same as dotenvx” multiline fixture against a SOPS dotenv file would fail unless Sopsdeck unwraps `\\n` on inject (SOPS already does this on load). Quoted block PEMs as in the Dotenvx docs are a **parser** difference, not just encryption.

## 4. File and environment precedence

### 4.1 Historic dotenv: existing env wins

Documented as the default for containers: already-set process env beats `.env` files. `--overload` forces files to win; with multiple `-f`, **last file wins** under `--overload`, **first file wins** without it.

Sources: [environment variable precedence](https://dotenvx.com/docs/cli/run-environment-variable-precedence), [run -f](https://dotenvx.com/docs/cli/run-f), [run --overload](https://dotenvx.com/docs/cli/run-overload), CLI help text: “by default, existing env vars take precedence over .env files” ([dotenvx.js](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/cli/dotenvx.js)).

Example from docs:

```
dotenvx run -f .env.local -f .env -- node index.js   # HELLO from .env.local
dotenvx run -f .env.local -f .env --overload -- …    # HELLO from .env
```

### 4.2 Default file when `-f` is omitted

`determine()` defaults to `.env`. If any `DOTENV_PRIVATE_KEY*` is set in the process environment, defaults become the filenames implied by those key names instead (`DOTENV_PRIVATE_KEY` → `.env`, `DOTENV_PRIVATE_KEY_PRODUCTION` → `.env.production`) ([determine.js](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/lib/helpers/envResolution/determine.js), [guessPrivateKeyFilename.js](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/lib/helpers/guessPrivateKeyFilename.js)). Explicit `-f` wins over that heuristic.

**Matching `DOTENV_PRIVATE_KEY*` file selection would be a false compatibility promise** under SOPS-only encryption: those variables are Dotenvx private keys, not SOPS age/PGP/KMS material.

### 4.3 Conventions

`--convention=nextjs` (or `DOTENV_CONFIG_CONVENTION=nextjs`) loads, first-wins:

`.env.${NODE_ENV}.local`, `.env.local` (skipped when `NODE_ENV=test`), `.env.${NODE_ENV}`, `.env`

`NODE_ENV` is taken from `DOTENV_ENV || NODE_ENV || 'development'`, and nextjs only treats `development|test|production` as canonical ([conventions.js](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/lib/helpers/conventions.js), [docs](https://dotenvx.com/docs/cli/run-convention-nextjs)).

`--convention=flow` loads: `.env.${env}.local`, `.env.${env}`, `.env.local`, `.env`, `.env.defaults`.

Passing `-f directory` with a convention resolves those names under that directory ([buildCommandEnvs.js](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/lib/helpers/buildCommandEnvs.js)).

### 4.4 SOPS polarity (contrast)

`exec-env` without `--pristine` starts from `os.Environ()` then **appends** decrypted `KEY=value` pairs ([exec.go `ExecWithEnv`](https://raw.githubusercontent.com/getsops/sops/v3.11.0/cmd/sops/subcommand/exec/exec.go)). Go `os/exec` uses the **last** duplicate key. So SOPS default is **file overrides existing env**. `--pristine` drops the parent environment entirely.

That is the inverse of Dotenvx’s default. A Sopsdeck `run` that copied SOPS `exec-env` defaults would **not** match Dotenvx; one that copied Dotenvx `--overload` polarity would **not** match SOPS.

## 5. Exit codes

Dotenvx does **not** publish a codes table. Observed from source and docs:

| Situation | Exit | Source |
| --- | --- | --- |
| Child process non-zero | child’s `exitCode`, else `1` | [executeCommand.js](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/lib/helpers/executeCommand.js) |
| Missing command / missing `--` | `1` | [run.js](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/cli/actions/run.js) |
| Resolver throw (decrypt, etc.) | `1` | `catchAndLog` + `process.exit(1)` |
| `--strict` and any processed error (missing file, decrypt) | `1`, **does not run the command** | [run.js](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/cli/actions/run.js), [run --strict](https://dotenvx.com/docs/cli/run-strict) |
| Missing `.env` **without** `--strict` | logs `[MISSING_ENV_FILE]`, **still runs** the command (typically 0 if child succeeds) | [issue #484](https://github.com/dotenvx/dotenvx/issues/484), run.js error loop |
| Interactive prompt cancelled | `130` | run.js / get.js / set.js `PROMPT_CANCELLED` |
| `get` with errors not in `--ignore` | `1` after printing | [get.js](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/cli/actions/get.js) |
| `get KEY --strict` missing key | `1` | [get --strict](https://dotenvx.com/docs/cli/get-key-strict) |
| `set` / `encrypt` / `precommit` operational errors | `1` | respective actions |

Error **codes** (strings, not exit statuses) include `MISSING_ENV_FILE`, `MISSING_KEY`, `MISSING_ENV_EXAMPLE`, `MISSING_ENV_KEYS_FILE`, `DECRYPTION_FAILED`, `COMMAND_SUBSTITUTION_FAILED`, `VALIDATION_FAILED`, `COMMAND_EXITED_WITH_CODE`, plus many key-crypto codes ([errors.js `ISSUE_BY_CODE`](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/lib/helpers/errors.js)). `--ignore=MISSING_ENV_FILE` (or `DOTENV_CONFIG_IGNORE`) suppresses listed codes.

SOPS **does** publish numeric statuses: generic `1`, read `2`, write `3`, MAC mismatch `51`, no file `100`, could not retrieve key `128`, already encrypted `203`, keyboard interrupt `85`, etc. ([codes.go](https://raw.githubusercontent.com/getsops/sops/v3.11.0/cmd/sops/codes/codes.go)). Copying SOPS numbers while claiming Dotenvx compatibility (or vice versa) would be a third false promise.

## 6. Encryption models — where “same CLI” would lie

| Surface | Dotenvx | SOPS dotenv / YAML |
| --- | --- | --- |
| Ciphertext in-file | `HELLO="encrypted:<base64 ECIES blob>"` ([encrypted files](https://dotenvx.com/docs/learn/encrypting/encrypted-files)) | `HELLO=ENC[AES256_GCM,data:…,iv:…,aad:…,tag:…]` ([README ENC examples](https://github.com/getsops/sops/blob/v3.11.0/README.rst)) |
| Public material in file | `DOTENV_PUBLIC_KEY=…` (secp256k1) | Flattened `sops_mac`, `sops_age__…`, `sops_lastmodified`, … with prefix `sops_` ([`SopsPrefix`](https://raw.githubusercontent.com/getsops/sops/v3.11.0/stores/dotenv/store.go)) |
| Private key file | `.env.keys` with `DOTENV_PRIVATE_KEY` / `DOTENV_PRIVATE_KEY_PRODUCTION` ([.env.keys docs](https://dotenvx.com/docs/env-keys-file)) | Age identity, PGP, KMS, etc. — **not** `.env.keys` |
| CI decrypt env | `DOTENV_PRIVATE_KEY=… dotenvx run -- …` | `SOPS_AGE_KEY` / `SOPS_AGE_KEY_FILE` / cloud creds / `SOPS_PGP_FP` (SOPS help text) |
| Crypto | ECIES, ephemeral key per secret, AES-256-GCM, secp256k1 ([whitepaper](https://dotenvx.com/dotenvx.pdf)) | AES-256-GCM data key wrapped by age/PGP/KMS; MAC over tree ([sops.go](https://github.com/getsops/sops/blob/v3.11.0/sops.go)) |
| `encrypt` CLI | Mutates `.env` in place; creates `.env.keys`; `--no-create` optional | Writes ciphertext to **stdout** unless `-i/--in-place`; recipients via `-a/-p/-k` or `.sops.yaml` |
| `set` | Encrypts that one key with Dotenvx public key | `sops set` is JSON-path mutation of an already-SOPS file, not dotenv `KEY value` |
| `keypair` | Prints Dotenvx public/private hex | No equivalent; age/PGP keys live outside the env file |
| `lock` / `native` / `armor` | Dotenvx private-key custody | Unrelated hosted/OS products |

Any flag named `--env-keys-file` / `-fk`, any help text mentioning `DOTENV_PRIVATE_KEY`, any output containing `encrypted:`, or any promise that existing Dotenvx-encrypted `.env` files decrypt, is a **false compatibility promise** given SOPS-only encryption.

`gitignore --pattern .env*` is also hazardous: Dotenvx’s default is “do not commit `.env*`”; Sopsdeck’s pitch is that encrypted Managed Files **are** committed.

## 7. Recommended compatibility subset vs “inspired by”

This is a proposed **contract shape** with evidence. Command *names* remain a founder decision.

### 7.1 Match intentionally (Dotenvx UX, SOPS crypto)

These are what “basically exactly the same as dotenvx” reasonably covers without lying about encryption:

1. **`run -- argv` injection** — argv after `--`, no implicit shell, inherit stdio, forward child exit code, document `sh -c 'echo $HELLO'` for POSIX expansion.
2. **`-f` composition** — multiple files, first-wins; `--overload` last-wins and overrides process env; default **process env wins** (Dotenvx historic dotenv, not SOPS `exec-env`).
3. **`-e/--env KEY=value`** inline pairs in the same precedence chain.
4. **`-q/--quiet` / `--verbose` / `--debug`** so `run` does not pollute captured stdout ([issue #470](https://github.com/dotenvx/dotenvx/issues/470)).
5. **Missing `--` → exit 1** with an explicit message (Dotenvx’s two messages are a good fixture).
6. **`--strict` vs default missing-file** — default: warn `[MISSING_ENV_FILE]`-style and still run; `--strict`: exit 1 and do not run.
7. **`get KEY` / `get` dump** — print plaintext after SOPS decrypt; `--format json|shell|eval` is useful and not crypto-specific. Do not default to implying Dotenvx `encrypted:` round-trip.
8. **`set KEY value` as upsert** — write one key into a Managed File, encrypting **with SOPS**, not with `encrypted:`. Default-on encryption matches Dotenvx *intent* only if the ciphertext is SOPS.
9. **Quoted multiline / `\\n` inject** — child `process.env` should contain real newlines for PEM-like values (Dotenvx fixture), even if on-disk SOPS dotenv stores `\\n`.
10. **`--convention=nextjs` (optional)** — file *order* is independent of Dotenvx crypto; only implement if Sopsdeck wants Next.js load order.

### 7.2 Inspired by, not compatibility

Rename or clearly diverge; same *job*, different flags/files:

| Dotenvx | SOPS already | Suggested stance |
| --- | --- | --- |
| `encrypt` / `decrypt` | `sops encrypt` / `sops decrypt` (`-i`, stdout default, recipients) | Inspired: in-place SOPS encrypt of a Managed File. Do not accept `-fk` / Armor `--token`. |
| `rotate` (removed from Dotenvx CLI) | `sops rotate` | Follow SOPS (new data key / recipients), not vanished Dotenvx keypair rotate. |
| `del` | `sops unset` | Inspired: delete a dotenv key then re-encrypt. |
| `ls` / `genexample` | none | Inspired: list Managed Files; emit example with key names. |
| `validate` / `run --validate` | none | Inspired: `.env.example` presence check. |
| `precommit` | none (Sopsdeck has its own scanner ticket) | Do **not** copy “prevent committing `.env*`”; that fights committed SOPS files. Inspired: prevent committing **plaintext**. |
| `gitignore` | none | Do **not** default `--pattern .env*`. |
| `--redact` / `--mask` | none | Inspired security UX; not a compatibility requirement. |
| `ext scan` | none | Overlaps Sopsdeck local scanning; keep as product feature, not a dotenvx shim. |

### 7.3 Do not match (false compatibility)

- `keypair`, `lock`, `native`, `armor`, `curl`, `login`/`logout`, `--token`, `--no-armor`, `--no-native`.
- `-fk` / `.env.keys` / `DOTENV_PRIVATE_KEY*` as decryption or as file-selection signals.
- Reading or writing `encrypted:` ciphertext or `DOTENV_PUBLIC_KEY`.
- `--no-1password` / `--no-bitwarden` / `op://` / `bw://` unless Sopsdeck independently implements those resolvers (Dotenvx-specific).
- Binary name `dotenvx` or `dx` as a silent shim (npm `bin` of Dotenvx). Map already notes `sopsdeck` vs shim is undecided; evidence says a shim would be a compatibility lie.
- SOPS `exec-env` flag names (`--pristine`, `--same-process`, `--user`) as if they were Dotenvx.
- SOPS numeric exit codes as if they were Dotenvx’s `process.exit(1)` / child passthrough.
- `precommit --install` that blocks encrypted dotenv commits.

## 8. Fixtures a later TDD slice should steal (behavior, not ciphertext)

Lift **behavioral** cases from Dotenvx; replace ciphertext with SOPS dotenv/YAML.

From Dotenvx tests and docs (all under [github.com/dotenvx/dotenvx/tree/main/tests](https://github.com/dotenvx/dotenvx/tree/main/tests)):

| Fixture / case | Why |
| --- | --- |
| [`tests/.env.multiline`](https://github.com/dotenvx/dotenvx/blob/main/tests/.env.multiline) + [run multiline docs](https://dotenvx.com/docs/cli/run-multiline) | Newlines survive into the child env. |
| [`tests/.env.expand`](https://github.com/dotenvx/dotenvx/blob/main/tests/.env.expand) + [interpolation summary](https://dotenvx.com/docs/cli/run-interpolation-syntax-summary) | `${A}`, `${A:-}`, `${A:+}`, `${A-}` / `${A+}`. Decide whether Sopsdeck implements expansion or documents “SOPS does not expand.” Dotenvx *does*. |
| [shell expansion](https://dotenvx.com/docs/cli/run-shell-expansion) | `run -- echo $HELLO` vs `run -- sh -c 'echo $HELLO'`. |
| [run -f first-wins](https://dotenvx.com/docs/cli/run-f) / [--overload](https://dotenvx.com/docs/cli/run-overload) | Two files, same key. |
| [precedence vs process env](https://dotenvx.com/docs/cli/run-environment-variable-precedence) | `HELLO=fromenv run -f .env -- …` vs `--overload`. |
| Missing `--` and missing command ([run.js](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/cli/actions/run.js)) | Exit 1, stable stderr. |
| Child `exit 3` ([executeCommand.js](https://raw.githubusercontent.com/dotenvx/dotenvx/main/src/lib/helpers/executeCommand.js)) | Wrapper exits 3, not 1. |
| Missing file without `--strict` vs with `--strict` ([run --strict](https://dotenvx.com/docs/cli/run-strict), [#484](https://github.com/dotenvx/dotenvx/issues/484)) | |
| `get KEY` missing ([get --strict](https://dotenvx.com/docs/cli/get-key-strict)) | |
| `--quiet` does not prefix child stdout ([#470](https://github.com/dotenvx/dotenvx/issues/470)) | |
| [`tests/cli/actions/run.test.js`](https://github.com/dotenvx/dotenvx/blob/main/tests/cli/actions/run.test.js) | Largest behavioral spec for `run` (~38KB). Use as a checklist, not as a ciphertext oracle. |
| Encoding: [`tests/.env.latin1`](https://github.com/dotenvx/dotenvx/blob/main/tests/.env.latin1), [`tests/.env.utf16le`](https://github.com/dotenvx/dotenvx/blob/main/tests/.env.utf16le) | Only if Sopsdeck claims Dotenvx encoding parity. |

**Do not** treat as compatibility fixtures:

- Any file containing `encrypted:` or `DOTENV_PUBLIC_KEY`.
- `.env.keys`.
- Armor / native / lock tests.
- Dotenvx `rotate` golden files from ≤1.57.

SOPS contrast fixtures (to prove divergence, not compatibility):

- Encrypted dotenv with `sops_*` keys and `ENC[AES256_GCM,…]`.
- `sops exec-env file 'echo $HELLO'` vs argv `run -- echo` (shell vs no-shell).
- Duplicate key: parent env vs file (SOPS last-wins vs Dotenvx first-wins).

## 9. Practical recommended contract (summary)

If Sopsdeck wants Dotenvx muscle memory **and** SOPS-only encryption, the honest contract is:

> **Workflow-compatible with Dotenvx `run` / `get` / `set` for plaintext injection and file composition; encryption-compatible with SOPS, not with Dotenvx.**

Concretely:

- Implement a `run --` argv runner whose precedence, `--overload`, `--strict`, quiet logging, multiline inject, and child exit codes match Dotenvx 2.21 docs/source.
- Implement `get`/`set`/`del` as dotenv-oriented operations on SOPS-encrypted Managed Files.
- Treat `encrypt`/`decrypt`/`rotate` as SOPS-shaped (or thin wrappers), never as writers of `encrypted:` / `.env.keys`.
- Refuse Dotenvx private-key flags and env vars; if a user passes `DOTENV_PRIVATE_KEY` or `-fk .env.keys`, fail loudly rather than ignoring them (silent ignore is also a compatibility lie).
- Do not ship a `dotenvx` binary name unless a later ticket explicitly accepts that lie.
- Do not copy `gitignore .env*` or `precommit` “block all .env commits.”
- Keep `--redact`, conventions, 1Password, Armor, and `ext` out of the v1 compatibility set unless separately specified.

Binary name (`sopsdeck` vs `sd` vs shim) is out of scope for this ticket; the evidence only constrains **what must not be claimed**.
