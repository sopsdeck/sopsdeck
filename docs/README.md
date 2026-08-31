# Sopsdeck documentation

This folder is the readable layer over the spec, glossary, and tests. Agents should treat these files as the source of truth for _what the product is_ and _what is currently proved_. Style rules still live in the quality scripts, not here.

## Read in this order

1. **Domain language** — [CONTEXT.md](../CONTEXT.md). Use these words in tests, UI copy, and new docs. Do not invent synonyms.
2. **Product decisions** — [map.md](../.scratch/sopsdeck-product/map.md). Resolved tickets, not code. If a build slice would invent policy, stop and read the map.
3. **What to implement next** — [build.md](../.scratch/sopsdeck-product/build.md). Phase status plus ready tickets.
4. **Public seams** — [seams.md](seams.md) (generated). Where tests must live, and which delivery phases still have none.
5. **Living features** — [features.md](features.md) (generated from test names). If a behavior is not named here, it is not specified by a test yet.
6. **Product stills and clips** — [assets.md](assets.md) (generated catalog). Created by `./scripts/demo`; `./scripts/docs --check` fails when files are missing or unlinked. `./scripts/demo --check` also fails sub-second clips. The public site serves the same pages from [site/src/pages/docs](../site/src/pages/docs/).
7. **Versioning** — [versioning.md](versioning.md). Epoch SemVer; `CHANGELOG.md` is canonical.

CLI casts (get, set, commit, Sync) live next to the desktop clips in [assets.md](assets.md).

Regenerate the living files with `./scripts/docs`. `./scripts/check` fails when they are stale.

## Quality scripts

| Script             | What it proves                                                                   |
| ------------------ | -------------------------------------------------------------------------------- |
| `./scripts/fmt`    | Formatters (gofumpt, prettier)                                                   |
| `./scripts/lint`   | gofumpt, golangci-lint, prettier, markdownlint, xo                               |
| `./scripts/test`   | Go and browser JavaScript tests                                                  |
| `./scripts/cover`  | Go coverage floor                                                                |
| `./scripts/docs`   | Regenerates features, seams, assets catalog, public asset copies, and What’s new |
| `./scripts/smoke`  | Local teammates + Playwright against `sopsdeck drive`                            |
| `./scripts/demo`   | Product stills, clips, walkthrough, and CLI casts into `docs/assets/`            |
| `./scripts/scan`   | `govulncheck` and `bun audit` (not in `./scripts/check`)                         |
| `./scripts/dev`    | Builds `./sopsdeck` and launches the browser app with that binary                |
| `./scripts/hooks`  | Opt-in `core.hooksPath=.githooks`; `git commit --no-verify` still skips          |
| `./scripts/mutate` | Mutation testing (slow; not in `./scripts/check`)                                |
| `./scripts/check`  | lint + test + cover + docs --check                                               |
