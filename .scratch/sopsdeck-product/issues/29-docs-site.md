# Docs site: landing, changelog, documentation

Type: build
Status: done
Blocked by: None

## What to build

Human QA wants a real docs surface, and is considering one Docusaurus (or similar) site for landing + changelog + documentation instead of the current split (`site/index.html`, `site/changelog.html`, `docs/*.md`).

Goal: one public site people can browse. Changelog still generated from `CHANGELOG.md` ([22](22-epoch-semver-and-changelog.md)). Living `docs/features.md` / `docs/seams.md` stay generated from tests (`./scripts/docs`).

Docusaurus is a **candidate**, not mandated. Vanilla `site/` + markdown docs is fine if the result is one coherent site. Do not duplicate three competing landing pages.

## Acceptance criteria

- [x] Public docs are reachable from the landing page (not only GitHub `docs/`).
- [x] Changelog is on that site and stays generated from `CHANGELOG.md`.
- [x] `./scripts/docs --check` still fails on stale generated pages / missing asset links.
- [x] Canonical domain remains sopsdeck.com; no new product policy in the docs copy.

## Seams

- `site/`, `docs/`, `./scripts/docs`.

## Comments

Captured 2026-08-28 from human-found review. Kind: idea.

Shipped as vanilla `site/` (no Docusaurus): landing nav **Docs** → `site/docs/` hub; living features/seams/assets plus versioning and CONTEXT glossary as HTML; notes still from `CHANGELOG.md`. Markdown sources stay the generated/hand files under `docs/`.
