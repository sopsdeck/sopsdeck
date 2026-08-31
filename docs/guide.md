# Using Sopsdeck

Sopsdeck is a local editor for [SOPS](https://github.com/getsops/sops)-encrypted dotenv, JSON, and YAML files. Files stay in your project. Access is a list of Age public keys on each file. Git is how you share and review changes.

Sopsdeck is under active development. Expect rough edges, and [submit a GitHub issue](https://github.com/sopsdeck/sopsdeck/issues/new) when something breaks.

## Get started

Install in a project, then open that folder:

```bash
npm install -D @sopsdeck/sopsdeck
npx sopsdeck .
```

SOPS is bundled. You need Node.js and Git. On first launch, create an Age identity and confirm you stored the backup in a password manager. The private key stays in the OS keychain; teammates never see it.

`npx sopsdeck .` opens that one Project. The sidebar is that folder’s Managed Files, not a list of recent Projects.

## Edit secrets

Open a Managed File, edit keys and values, then **Encrypt & save**. Values stay hidden until you reveal them. Saving encrypts the file on disk and can create a Git commit.

## Rename keys

Rename a key in the editor. On Encrypt & save, Sopsdeck can rewrite whole-word references across the Project (`KEY`, `$KEY`, and `${KEY}`) so application code keeps matching the new name. The CLI is `sopsdeck rename OLD NEW -f FILE`.

## Lock files

A locked file is SOPS ciphertext on disk. Unlock writes plaintext so a local tool can read it; lock encrypts it again. Use Unlock only on this machine. Do not commit an unlocked file.

The badge in the editor follows that state: **Locked** or **Unlocked**.

## Unused secrets

Sopsdeck scans the Project for references to each key. Keys with zero references show an **unused** badge. `sopsdeck unused -f FILE` prints the same list. This is advisory; it does not delete anything.

## Secret history

Every Encrypt & save can commit. File history lists those commits. Open a secret’s history to see that key at each revision. `get KEY -f FILE --at REV` decrypts one historical value. Restore copies a revision into the worktree and leaves it uncommitted.

## Field encryption

JSON and YAML files do not have to encrypt every field. Open a structured file to see its tree. Lock only the leaves you choose — for example `EXPO_TOKEN` in `eas.json`. Unselected keys stay plaintext so CLIs can still read them. The File inspector lists encrypted paths so you can add or remove them later.

## Encrypted in place

SOPS encrypts values in place. Keys, comments, and structure stay readable. A locked `eas.json` looks like this:

```json
{
  "cli": { "version": "20.5.1" },
  "build": {
    "env": {
      "EXPO_PUBLIC_API_URL": "https://api.example.com",
      "EXPO_TOKEN": "ENC[AES256_GCM,data:m4nY8pQ=,iv:…,tag:…,type:str]"
    }
  },
  "sops": {
    "age": [{ "recipient": "age1…" }]
  }
}
```

`EXPO_PUBLIC_API_URL` is still a normal string. Only `EXPO_TOKEN` is ciphertext. The `sops` metadata lists who can decrypt.

## Access

Each Managed File lists Age public keys (Recipients). Adding a teammate’s key lets them decrypt. Removing a key rotates the data key; Git history can still decrypt.

When you initialize a Project, your Git identity is recorded in `.sopsdeck.toml` with your Age public key so teammates can see who you are. When you add someone, enter their name or git identity (`Bob <bob@example.com>`) with their Age public key. That label is stored in the same file.

To join a file you cannot open, open **Account**. Copy your Age public key, or copy a request message that includes it, and send that to a Project owner.

## Project owners

Anyone who can decrypt a file can technically re-encrypt it with extra keys. Sopsdeck records **owners** in `.sopsdeck.toml` when you initialize a Project. Only those owners can add Recipients in Sopsdeck. If no owners are listed, the previous behavior remains: anyone with Access can add people.

Put a GitHub `CODEOWNERS` file on `.sopsdeck.toml` (and the Managed Files) so Access PRs need owner review. `sopsdeck recipient request` opens a metadata-only PR; `sopsdeck recipient grant` re-encrypts and opens the Access PR.

## Copy your key

Your Age public key is in **Account** and on the **Project** panel. Copy it and send it to an owner, or include it in a Request access message. The private key never leaves the keychain.
