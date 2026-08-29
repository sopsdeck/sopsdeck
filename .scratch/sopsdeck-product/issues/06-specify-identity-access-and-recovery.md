# Specify identity, Access, removal, and recovery

Type: grilling
Status: resolved
Blocked by: None

## Question

What exact onboarding, key storage, recipient policy, per-scope Access, add/remove User, rotation, and lost-key workflows make public-key-only Access safe and understandable? Private keys are never shared; recovery means adding another Recipient, not a team password.

## Answer

Human Users get an Age keypair on first run. The private key lives in the OS keychain (`SOPS_AGE_KEY_CMD`). Creation is blocked until they confirm that same private key is saved in **their** password manager (copy/download in v1, no PM API). Restore imports that identity into the keychain. Existing Age files and PGP remain usable in place; private material is never copied into Sopsdeck. KMS is not a day-one human setup.

Access is per **Managed File** (SOPS recipient list). Nested Environments in one file share Recipients. Different people for staging vs production means different files. The creating User is a Recipient on files they encrypt. There are no Sopsdeck roles; Git permissions plus Access decide who can merge a re-encrypt PR.

Join without Access: Sopsdeck opens a **request** PR (name + public key, no ciphertext). Out-of-band is the same public key in chat. No P2P key-exchange network. An existing User accepts in Sopsdeck, which re-encrypts the named files (or all Managed Files in the Project) and opens the real PR. Adding a teammate you already have Access with is only the re-encrypt PR.

CI is a User with its own Age keypair. Public key is a Recipient; private key lives only in GitHub Secrets / `SOPS_AGE_KEY`.

Remove User: drop their Recipient, rotate the data key, re-encrypt HEAD, PR. Warn that Git history and copies they already have still decrypt. No history rewrite or force-push.

Lost keychain + lost PM backup: Access is gone. Lost device with PM backup intact: restore the same identity. Stolen device with PM intact: **Replace my key** — new Age identity, mandatory PM backup, add new Recipient, remove the old one (same rotate/PR as remove). No Sopsdeck project recovery key.

## Comments

- First-run Age keypair for each human User; private key in OS keychain via `SOPS_AGE_KEY_CMD`. Existing Age files and PGP stay usable in place; private material is never copied in. KMS is not a day-one human setup.
- Join/share channels: Sopsdeck-created Git pull request, or out-of-band public key (chat/email). No Sopsdeck P2P/key-exchange network.
- Join without Access: request PR (name + public key, no ciphertext). Join with Access (adding a teammate): re-encrypt PR. Accepting a request is done by an existing User in Sopsdeck, which then re-encrypts.
- Access is per Managed File. Different recipient sets require different files. Nested Environments in one file share Recipients. Join requests name files or all Managed Files in the Project.
- Remove User: drop Recipient on chosen files, rotate the data key, re-encrypt HEAD, PR. Warn that Git history and copies they already have still decrypt. No history rewrite or force-push.
- Identity backup is mandatory (password manager). Losing the device without that backup means losing Access. No Sopsdeck-owned project recovery key.
- Backup is the same Age identity, not a second Recipient. Creation is blocked until the user confirms the private key is saved in their password manager (copy/download in v1, not a PM API). Restore imports that identity into the keychain.
- CI is a User with its own Age keypair. Public key is a Recipient; private key lives only in GitHub Secrets / `SOPS_AGE_KEY`, never in Git and never as a shared team key. Adding CI is the re-encrypt-PR path.
- Stolen device with PM backup intact: Replace my key (new identity, backup, add Recipient, remove old with data-key rotate).

## Implementation (2026-08-28)

Identity create/import persist the Age private key in the OS keychain (`go-keyring`; tests use `SOPSDECK_KEYCHAIN_DIR`) after `--confirmed-backup`. `sopsdeck identity key` prints it for `SOPS_AGE_KEY_CMD`. Existing `SOPS_AGE_KEY_FILE` and `$SOPSDECK_STATE_DIR/age.txt` still decrypt. Studio Users are throwaway files, not keychain. `recipient add` re-wraps the data key so a second Age identity can decrypt. `recipient remove` drops the Recipient, rotates the data key, and warns that Git history still decrypts (CLI, drive, inspector). `recipient request` opens a metadata-only PR for named or all Managed Files. `recipient grant` re-encrypts the same selection for the new Recipient and opens the Access PR. Both require a clean worktree and return to the original branch.
