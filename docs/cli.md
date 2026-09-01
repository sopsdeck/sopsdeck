# CLI

Sopsdeck’s CLI lives in the same local runner as the browser app. Files stay in your project. Age keys stay in the OS keychain.

```bash
sopsdeck get KEY -f path/to/.env.production
sopsdeck set KEY VALUE -f path/to/.env.production
sopsdeck del KEY -f path/to/.env.production
sopsdeck lock -f path/to/.env.production
sopsdeck unlock -f path/to/.env.production
```

`get` without a key dumps the file. `--output json` prints every leaf. `--at REV` decrypts a Git revision.

## Access

```bash
sopsdeck recipient add AGE1... -f FILE --name "Ada <ada@example.com>"
sopsdeck recipient list -f FILE
sopsdeck recipient remove AGE1... -f FILE
sopsdeck recipient request AGE1... --name NAME --all
sopsdeck recipient grant AGE1... --name NAME -f FILE
```

`request` opens a metadata-only PR. `grant` re-encrypts and opens the Access PR. Only Project owners can grant once owners are recorded in `.sopsdeck.toml`.

`recipient remove` re-encrypts the current file with a fresh SOPS data key. Run it for every Managed File a departing person could read, then rotate the actual provider credentials they previously knew. It cannot revoke old Git clones, history, or values they already copied.

## Git

```bash
sopsdeck commit -m "rotate stripe" -f FILE
sopsdeck review -f FILE
sopsdeck history -f FILE
sopsdeck restore -f FILE --at REV
sopsdeck sync
```

`sync` is `git fetch`, `git pull --ff-only`, then `git push`. It never force-pushes.

## Project

```bash
sopsdeck project init FOLDER --file eas.json --keys build.env.EXPO_TOKEN
sopsdeck project add FOLDER --file compose.yaml --keys services.db.environment.POSTGRES_PASSWORD
sopsdeck project remove FOLDER --file compose.yaml
sopsdeck project encrypt FILE --keys build.env.EXPO_TOKEN,build.env.SECRET
sopsdeck files FOLDER
```

JSON and YAML encrypt only the paths you pass to `--keys`. Dotenv files encrypt every key.

## Other commands

```bash
sopsdeck identity create --confirmed-backup
sopsdeck identity key
sopsdeck identity import -f age-identity.txt --confirmed-backup
sopsdeck identity remove --yes
sopsdeck run -f FILE -- your-command
sopsdeck rename OLD NEW -f FILE
sopsdeck unused -f FILE
sopsdeck publish -f FILE --yes
sopsdeck scan
sopsdeck mcp
```

`identity key` prints the Age private key for SOPS; save the entire output in a password manager and never commit it. `identity remove --yes` clears only this machine’s OS-keychain identity; it does not remove the public key from files. `SOPSDECK_STATE_DIR` is optional: when set, failed commands append redacted messages to `$SOPSDECK_STATE_DIR/errors.json`.
