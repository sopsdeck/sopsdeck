# Define the threat model and security boundaries

Type: grilling
Status: resolved
Blocked by: None

## Question

What does Sopsdeck claim to protect, against whom, and where may plaintext exist? Spell the trusted computing base, threat actors (laptop theft, malicious teammate, compromised Git host, model/tooling exfiltration, accidental commit), what the product does not protect (Git history, copied values, a compromised machine), and the rule for when secret material may leave the device.

## Answer

Sopsdeck is dotenvx’s product shape — encrypted config in Git, decrypt to edit or `run`, project-as-folder — with Dotenvx’s collaboration model inverted. Dotenvx shares a private key (`.env.keys` / `DOTENV_PRIVATE_KEY`). Sopsdeck never does. Access is SOPS asymmetric encryption: teammates and CI receive **public keys only**. Private keys stay with their owner.

**Trusted computing base.** The user’s unlocked machine, OS disk encryption, signed Sopsdeck + SOPS binaries, and each User’s private key. The Tauri WebView is untrusted (no keys, no plaintext persistence). Git hosts, GitHub, Expo, AI providers, and update servers are not trusted with plaintext. Git metadata may list recipient public keys; that is intended.

**Plaintext may exist** in memory while a Managed File is open; on screen masked by default; in the clipboard only if the user copies; in a child process environment only for explicit `run`; at a Sync Target only after a reviewed push.

**Plaintext may not exist** as a decrypted working copy on disk; in logs, crash reports, or analytics; in AI/MCP context without per-call approval; in Git except as SOPS ciphertext; in any shared private-key file (no `.env.keys` analogue, no team passphrase).

**Leave-the-machine rule.** Secret material leaves the device only through explicit user action: copy, `run`, or a reviewed Sync Target push. Sharing a public key is not sharing a secret. Putting a CI identity’s private key in GitHub Secrets is that automation User’s key, not a team password.

**In scope.** Accidental plaintext commit; accidental private-key commit or private-key sharing; decrypt by anyone who is not a current recipient; silent prune of unowned GitHub secrets; models/tools seeing values by default; a Git host reading plaintext.

**Out of scope (must be said).** An unlocked or malware-ridden machine; a current recipient who copies values; Git history from when a User still had Access; values already at a Sync Target; screen share; a counterfeit binary.

Recovery is “add another Recipient,” not a shared master password. Age vs PGP vs KMS and where private keys live remain [Specify identity, Access, removal, and recovery](06-specify-identity-access-and-recovery.md).
