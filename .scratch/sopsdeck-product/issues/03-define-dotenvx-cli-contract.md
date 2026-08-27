# Define the Dotenvx-shaped CLI compatibility contract

Type: research
Status: resolved
Blocked by: None

## Question

Which Dotenvx commands, flags, precedence rules, shell semantics, exit codes, and fixtures should `sopsdeck` intentionally match while retaining SOPS-only encryption and avoiding false compatibility promises?

## Answer

Match Dotenvx **workflow**, not Dotenvx **crypto**. The honest contract, from Dotenvx 2.21.0 source/docs and SOPS 3.11.0, is: argv `run -- cmd` injection, `-f` first-wins composition, process-env-beats-file unless `--overload`, `--strict` vs warn-and-run on missing files, child exit-code passthrough, `--quiet`, and `get`/`set`/`del` as dotenv operations — while encrypting only with SOPS. Do not match `encrypted:` values, `.env.keys`, `DOTENV_PRIVATE_KEY*`, `-fk`, `keypair`, Armor/`lock`/`native`, or a silent `dotenvx`/`dx` shim; those would claim ciphertext compatibility Sopsdeck is not offering. Do not copy `gitignore .env*` or “block all `.env` commits” precommit, which fights committed SOPS files. Do not copy SOPS `exec-env` defaults either: that command is a `/bin/sh -c` string whose decrypted keys **override** existing env (last-wins), the inverse of Dotenvx. Treat `encrypt`/`decrypt`/`rotate` as SOPS-shaped; Dotenvx `rotate` is gone from current `main`. Fixtures should replay Dotenvx behavior (multiline inject, shell `sh -c '…'`, first-wins vs overload, missing `--`, child exit 3) against SOPS ciphertext, never against `encrypted:` golden files. Full inventory, citations, and the match / inspired-by / do-not-match tables are in [research/03-dotenvx-cli-contract.md](../research/03-dotenvx-cli-contract.md). Binary name remains a later policy choice; this ticket only constrains what must not be claimed.

## Implementation (2026-08-28)

`get`/`set`/`del`/`run` against SOPS dotenv/JSON/YAML are tested. Extra commands beyond this contract: `identity`, `commit`, `sync`, `recipient add`, `publish`, `files`, `drive`.

