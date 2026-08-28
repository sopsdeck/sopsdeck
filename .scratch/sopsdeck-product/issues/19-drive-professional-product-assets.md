# Drive professional product assets and testable docs

Type: build
Status: ready
Blocked by: None (20 and 21 are done)

## What to build

Turn the existing driver (`sopsdeck drive`, studio, Playwright, `./scripts/demo`) into **public** assets: landing page, GitHub, X / HN / Reddit, and docs. Not one long recording.

Quality bar: Screen Studio–like if a scriptable path exists (window-only frame, cursor, zooms, no OS desktop, no `/var/folders` or `desktop/../testdata` paths). If that is not reachable from code, a clean recording of the driven UI is the fallback — still short, still composed, still usable in public.

Need:

- One full walkthrough (open Project, edit, save, commit, Sync, grant Access, Publish) against **studio** — no extra machine or GitHub account.
- Short per-feature clips and stills, one idea each (reveal, encrypt & save, Sync, recipient add, Publish dry-run, …).
- Docs that **embed those generated files** and fail a check when they are missing or stale. Hand-copied screenshots that rot are out.

`./scripts/demo` writing `docs/assets/editor.png` and `editor-revealed.png` is the current floor, not the bar. `./scripts/check` stays fast: asset generation stays in `./scripts/demo` / a dedicated assets script, with an optional `--check` that only asserts files exist and docs link them.

Do not invent product behavior in a clip that is not already proved in [docs/features.md](../../docs/features.md).

## Already there

- `sopsdeck drive --demo` seeds `checkout` + Bob Access + local origin + fake GitHub.
- Playwright smoke (`e2e/smoke.spec.js`) and stills (`e2e/demo.spec.js`).
- Living feature/seams markdown from tests (`./scripts/docs`).

## Acceptance criteria

- [ ] A full walkthrough video (or scripted recording) covers the studio teammate + Publish path without a second GitHub account or machine.
- [ ] Separate short clips **and** stills exist per proved editor action (at least: open Managed File, reveal, save, commit, Sync, recipient add, Publish).
- [ ] Public frames do not show raw filesystem junk (`/var/folders`, `../testdata`, unexpanded `desktop/../…`) — depends on 21 display paths.
- [ ] Walkthrough does not show raw `git-pull(1)` (or equivalent) as the user-visible failure — depends on 20.
- [ ] Docs that show the product (README, `docs/`, landing page as applicable) reference the generated files; a script fails when those files or links are stale.
- [ ] Asset jobs are not part of `./scripts/check`.

## Seams

- Playwright against `sopsdeck drive` (existing).
- Studio + fake GitHub (existing).
- Generated docs/asset index (extend `./scripts/docs` or `./scripts/demo`).

## Comments

Captured 2026-08-28; triaged 2026-08-28. Kind: idea → build.
