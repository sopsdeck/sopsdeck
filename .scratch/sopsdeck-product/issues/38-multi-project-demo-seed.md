# Multi-project demo seed for stills

Type: build
Status: done
Blocked by: None

## What to build

Demo screenshots still look sparse. Seed should show a sidebar with several Projects (some expanded, some collapsed), realistic names, and mixed Managed Files (compose, dotenv, eas.json, other common framework configs we already support). Issue [27](27-realistic-managed-file-fixtures.md) added file types; this ticket is **how the demo tree is arranged on camera**.

Do not encrypt unsupported formats. Do not invent a second discovery rule.

## Acceptance criteria

- [x] `sopsdeck drive --demo` (or demo Playwright seed) registers more than one Project.
- [x] Stills/clips that show the sidebar include nested/realistic files already in testdata plus extra Projects.
- [x] `./scripts/demo --check` still passes; no sub-second clips.

## Seams

- Studio demo seed, `e2e/demo.spec.js`, [27](27-realistic-managed-file-fixtures.md).

## Comments

Captured 2026-08-28 from human-found review. Kind: idea.

`drive --demo` seeds `checkout` (expanded, nested `apps/web/.env`), `atlas-web`, and `docs-site` (collapsed). Same Age identity. Discovery stays `list_managed_files`.
