# Sopsdeck

Local-first workspace for SOPS-encrypted config (dotenv, JSON, YAML). Canonical site: [sopsdeck.com](https://sopsdeck.com).

## Prerequisites

- Go 1.25+
- system `git`
- for the desktop app: Rust (stable), Node 22+, and a macOS machine

## CLI

```bash
go test ./internal/cli/
go build -o sopsdeck ./cmd/sopsdeck
```

First-run identity (Age key is not saved until you confirm a password-manager backup):

```bash
export SOPSDECK_STATE_DIR="$HOME/.sopsdeck"
./sopsdeck identity create --confirmed-backup
export SOPS_AGE_KEY_FILE="$SOPSDECK_STATE_DIR/age.txt"
```

```bash
./sopsdeck get KEY -f path/to/.env.production
./sopsdeck set KEY VALUE -f path/to/.env.production
./sopsdeck run -f path/to/.env.production -- your-command
./sopsdeck commit -m "rotate stripe" -f path/to/.env.production
./sopsdeck sync
```

`sync` is `git fetch`, `git pull --ff-only`, then `git push`. It never force-pushes.

## Desktop

The app shells out to the `sopsdeck` binary. Build that first, then:

```bash
cd desktop
npm install
SOPSDECK_BIN="$(pwd)/../sopsdeck" \
SOPS_AGE_KEY_FILE="$(pwd)/../testdata/age.txt" \
SOPSDECK_DEV_PROJECT="$(pwd)/../testdata" \
npm run tauri -- dev
```

`SOPSDECK_DEV_PROJECT` auto-opens a folder (handy with `testdata`). Omit it and use **Add folder from disk**. `testdata/age.txt` is a throwaway test key, not a personal identity.

## Landing page

```bash
python3 -m http.server 4173 --directory site
```

Open http://127.0.0.1:4173/
