# Using Sopsdeck

Sopsdeck is a local editor for [SOPS](https://github.com/getsops/sops)-encrypted dotenv, JSON, and YAML files. Files stay in your project. Access is a list of Age public keys on each file. Git is how you share and review changes.

Sopsdeck is under active development. Expect rough edges, and [submit a GitHub issue](https://github.com/sopsdeck/sopsdeck/issues/new) when something breaks.

## Get started

Install in a project, then open that folder:

```bash
npm install -D @sopsdeck/sopsdeck
npx sopsdeck .
```

SOPS is bundled. You need Node.js and Git. On first launch, create an Age identity, then copy its private-key backup into a password manager. The private key stays in the OS keychain; teammates never see it.

`npx sopsdeck .` opens that one Project. The sidebar is that folder’s Managed Files, not a list of recent Projects.

Projects also work outside Git repositories; Git history is unavailable there. Opening a subfolder of an initialized Project reopens its root. Monorepo files can share one root manifest, or use independent manifests in package folders. Nested Git repositories and nested Projects are excluded from their parent's scans. Project paths resolve symlinks, but managed files must be regular files inside that Project, without symlinks.

## Edit secrets

Open a Managed File, edit keys and values, then **Encrypt & save**. Values stay hidden until you reveal them. Saving encrypts the file on disk; commit and push with Git when you are ready.

## Rename keys

Rename a key in the editor. On Encrypt & save, Sopsdeck can rewrite whole-word references across the Project (`KEY`, `$KEY`, and `${KEY}`) so application code keeps matching the new name. The CLI is `sopsdeck rename OLD NEW -f FILE`.

## Lock files

A locked file is SOPS ciphertext on disk. Unlock writes plaintext so a local tool can read it; lock encrypts it again. Use Unlock only on this machine. Do not commit an unlocked file.

The badge in the editor follows that state: **Locked** or **Unlocked**.

## Unused secrets

Sopsdeck scans the Project for references to each key. Keys with zero references show an **unused** badge. `sopsdeck unused -f FILE` prints the same list. This is advisory; it does not delete anything.

## Secret history

Every save is a local change you commit with Git. File history lists those commits. Open a secret’s history to see that key at each revision. `get KEY -f FILE --at REV` decrypts one historical value. Restore copies a revision into the worktree and leaves it uncommitted.

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

Each Managed File lists Age public keys (Recipients). Adding a teammate’s key lets them decrypt. Removing a key re-encrypts the current file with a fresh SOPS data key; the removed key cannot decrypt the new version, but Git history and copies they already have can still decrypt.

When you initialize a Project, your Git identity is recorded in `.sopsdeck.toml` with your Age public key so teammates can see who you are. When you add someone, enter their name or git identity (`Bob <bob@example.com>`) with their Age public key. That label is stored in the same file.

To join a file you cannot open, open **Account**. Copy your Age public key, or copy a request message that includes it, and send that to a teammate with Access.

Initialize a shared Project once, in one teammate's checkout. That teammate grants the other public keys, then commits and pushes both the encrypted file and `.sopsdeck.toml`. Other teammates pull those changes into their own checkouts; they do not initialize it again. **Secret Sync** sends values to integrations, not other checkouts. A name in Access is only a label: replacing an Age identity requires granting its new public key.

## Managing Access

Anyone who can decrypt a file can add or remove its Recipients. Sopsdeck does not assign owner roles. Older `[[owner]]` entries are read as recipient labels and converted to `[[recipient]]` entries when the manifest is next saved.

Use GitHub `CODEOWNERS` and branch protection if your team requires review of Access changes. `sopsdeck recipient request` opens a metadata-only PR; `sopsdeck recipient grant` re-encrypts and opens the Access PR.

## Copy your key

Your Age public key is in **Account** and on the **Project** panel. Copy it and send it to a teammate with Access, or include it in a Request access message. Back up your private key to your password manager from **Account**.

## Back up and recover your identity

Open **Account**, choose **Back up private key**, and save the complete Age identity block in a password manager. It is the only way to recover access if this machine is lost or its OS keychain is cleared. Clear your clipboard after pasting it.

The CLI equivalent is `sopsdeck identity key`; it prints the private key, so never paste its output into an issue, chat, or Project file. Restore it on a replacement machine with `sopsdeck identity import -f FILE --confirmed-backup`.

## What Sopsdeck stores

- Encrypted Managed Files and `.sopsdeck.toml` live in the Project and can be committed to Git. `.sopsdeck.toml` contains public recipient keys and labels, never private keys.
- Your private Age identity is in the operating system keychain. On macOS, the keychain item uses service `sopsdeck` and account `age`; it is not a folder in the Project or home directory.
- The browser keeps only UI preferences, recent Project paths, and folder/inspector state in browser local storage for its `127.0.0.1` origin. It does not store secret values or the Age private key, and never reads your clipboard automatically.
- CLI diagnostics are optional: if you set `SOPSDECK_STATE_DIR`, it contains exit summaries in `$SOPSDECK_STATE_DIR/errors.json`. Child stderr is streamed to the caller and is not retained.

To forget browser UI state, clear site data for Sopsdeck’s local `127.0.0.1` address in your browser; Projects, keys, and encrypted files remain untouched. To remove the identity from this machine, use **Account → Remove local identity** or `sopsdeck identity remove --yes`. That does not revoke its public key from any Managed File and makes local decryption impossible until you import a backup. Delete the optional `errors.json` file if you want to clear CLI diagnostics.

## Recover an out-of-sync manifest

If `.sopsdeck.toml` lists a missing or unsafe file, Sopsdeck shows a warning and keeps the other Managed Files usable. Opening the Project does not rewrite the manifest, recreate missing files, or change your identity.

Restore an intended missing file from Git, or open **Project → Edit managed files**, uncheck the stale entry, and apply the change. This removes only the manifest entry. The CLI equivalent is `sopsdeck project remove FOLDER --file PATH`; it also works when the file is missing. Commit and share the corrected manifest with your normal Git workflow.

If the manifest contains invalid TOML, fix it on disk and choose **Retry opening project**. Do not reinitialize the Project or remove your identity. Managed Files may use any name; Sopsdeck detects dotenv, JSON, and YAML content when possible and treats other content as a binary Managed File.

## Remove someone and rotate secrets

Remove a person under **Access** for every Managed File they could read, then commit and push with Git. Sopsdeck rotates the SOPS data key for each removed file automatically.

That protects future file revisions, not secrets the person already read, copied, or has in Git history. Rotate the actual provider credentials afterward (for example, create a new Stripe key, update the Managed File, commit, and revoke the old Stripe key). If the person’s device or Age key might be compromised, treat that provider rotation as required.

To replace your own Age identity, first create and back up the replacement identity on another machine, add its public key to every Managed File from an existing authorized machine, and confirm it can decrypt. Only then remove the old public key from every file. Never remove the last usable identity before the replacement has Access.
