# Specify local secret scanning and commit prevention

Type: grilling
Status: resolved
Blocked by: 07

## Question

Which files and staged changes are scanned, which detectors and confidence thresholds block or warn, how are SOPS ciphertext and legitimate fixtures ignored, how do baselines/allowlists work, and how is an opt-in Git hook installed and bypassed safely?

## Answer

Local only. Opt-in pre-commit hook, installed from Sopsdeck / recorded in the manifest. Scan staged files. High-confidence detectors (cloud keys, private key PEMs, common tokens) **block**. Lower-confidence matches **warn**. SOPS ciphertext (`ENC[`, `sops` metadata) is ignored. Committed allowlist/baseline in the repo for known fixtures. `git commit --no-verify` still works; the app can warn after a bypass. No remote scanning.

## Implementation (2026-08-28)

`sopsdeck scan` reads `git diff --cached` and reports path + detector without secret values. Blocks `AKIA…`, `-----BEGIN … PRIVATE KEY-----`, and `ghp_` / `gho_` / `github_pat_` / `sk_live_` tokens; warns on `sk_test_`. Strips `ENC[…]` and `sops` / `sops_` metadata. `[scan] allowlist` paths in `.sopsdeck.toml` skip those staged files. `sopsdeck scan --install` writes `.git/hooks/pre-commit` (`sopsdeck scan`) and sets `scan.hook = true`. No `git config`. Not done: desktop warning after a `--no-verify` bypass.
