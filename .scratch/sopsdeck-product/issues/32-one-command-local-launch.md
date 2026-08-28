# One command to build the CLI and launch the app

Type: build
Status: open
Blocked by: None

## What to build

Local run is several steps (build CLI, `cd desktop`, `bun install`, set `SOPSDECK_BIN`, `tauri dev`). Human QA wants **one command** that builds a fresh `sopsdeck` and launches the desktop app against that binary.

Do not change product behavior. Do not auto-open testdata unless the user already set `SOPSDECK_DEV_PROJECT`.

## Acceptance criteria

- [x] From the repo root, one script builds `./sopsdeck` and starts Tauri `dev` with `SOPSDECK_BIN` pointing at that binary.
- [x] README / desktop README tell humans to use that command.
- [x] `--build-only` exits after the CLI build (no Tauri) so the path is checkable without a GUI.

## Seams

- `./scripts/dev`, sidecar `SOPSDECK_BIN` (existing). `TestDevScriptBuildOnlyWritesCLI`.

## Implementation (2026-08-28)

`./scripts/dev` runs `go build -o sopsdeck ./cmd/sopsdeck`, then `bun run tauri -- dev` in `desktop/` with `SOPSDECK_BIN` set. `--build-only` prints the binary path and exits. Does not set `SOPSDECK_DEV_PROJECT`.

## Comments

Captured 2026-08-28 from human-found review. Kind: idea.
