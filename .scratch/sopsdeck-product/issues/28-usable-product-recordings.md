# Usable product recordings (desktop + CLI)

Type: build
Status: ready
Blocked by: [21](21-desktop-chrome-polish.md) display paths (done)

## What to build

[19](19-drive-professional-product-assets.md) is **done** as a pipeline (Playwright webm + stills + docs check) and the files are **unusable**: clips are ~0.01s. Human QA: record with something like [webreel](https://webreel.dev/) for the UI, and [castkit](https://github.com/deeflect/castkit) for CLI demos.

Replace or wrap `./scripts/demo` so public assets are watchable. Keep: no `/var/folders` junk, studio teammate path, docs fail when assets are missing. `./scripts/check` stays fast; generation stays in demo/assets jobs.

CLI recordings are new. They must show real `sopsdeck` commands against testdata/studio, not a mocked prompt.

## Acceptance criteria

- [ ] Each public video is long enough to follow (seconds, not a single frame); a check fails sub-second clips.
- [ ] Desktop walkthrough + per-feature clips are regenerated with a recorder that produces usable motion (webreel or equivalent).
- [ ] At least a few CLI casts exist (get/set/commit/sync or the current proved subset) and are linked from docs/README.
- [ ] Asset jobs remain outside `./scripts/check`; `--check` still asserts files exist and are linked.

## Seams

- `./scripts/demo`, `docs/assets/`, README / `docs/` embeds.
- Optional webreel/castkit in CI or a documented local command.

## Comments

Captured 2026-08-28 from human-found review. Kind: bug against 19’s quality bar. 19’s check only proved files exist.
