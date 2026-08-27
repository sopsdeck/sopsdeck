# Sopsdeck

Sopsdeck is a local-first workspace for managing encrypted configuration files and projecting selected secrets into external systems.

## Language

**Project**:
A local folder registered by canonical filesystem path, usually but not necessarily a Git worktree. Separate worktrees are separate Projects even when they share a repository.
_Avoid_: Repository, vault, workspace

**Project Manifest**:
The committed plaintext `.sopsdeck.toml` that lists Managed Files, Sync Target mappings, and scan policy without credentials, secret values, or machine paths.
_Avoid_: Settings database, vault configuration

**Managed File**:
A supported secret-bearing configuration file inside a Project whose encrypted local contents are canonical.
_Avoid_: Environment file, vault

**Environment**:
A logical secret scope exposed by a Managed File's format. A dotenv file usually contains one Environment, while a structured file may contain several nested Environments.
_Avoid_: File, project

**Sync Target**:
An external system such as GitHub Actions or EAS that receives selected values from a Managed File without becoming a competing source of truth.
_Avoid_: Source of truth, remote vault

**User**:
A named person or automation identity bound to one or more Recipients.
_Avoid_: Account, seat

**Recipient**:
A SOPS public key listed on a Managed File. Sharing a Recipient is how Access is granted; the matching private key is never shared.
_Avoid_: Shared key, password, .env.keys

**Access**:
A User's ability to decrypt a Managed File because one of their Recipients is listed on it. Removing Access does not retract values already copied or present in Git history.
_Avoid_: Role, login permission

**Secret History**:
The Git-derived sequence of changes to secret keys and values in a Managed File.
_Avoid_: Audit database, backup
