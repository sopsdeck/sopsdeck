# Sopsdeck documentation

This folder is the readable layer over the spec, glossary, and tests. Agents should treat these files as the source of truth for _what the product is_ and _what is currently proved_. Style rules still live in the quality scripts, not here.

## Read in this order

1. **Domain language** — [CONTEXT.md](../CONTEXT.md). Use these words in tests, UI copy, and new docs. Do not invent synonyms.
2. **Product decisions** — [map.md](../.scratch/sopsdeck-product/map.md). Resolved tickets, not code. If a build slice would invent policy, stop and read the map.
3. **What to implement next** — [build.md](../.scratch/sopsdeck-product/build.md). Phase status plus ready tickets.
4. **Public seams** — [seams.md](seams.md) (generated). Where tests must live, and which delivery phases still have none.
5. **Living features** — [features.md](features.md) (generated from test names). If a behavior is not named here, it is not specified by a test yet.
6. **Product stills and clips** — [assets.md](assets.md) (generated catalog). Created by `./scripts/demo`; `./scripts/docs --check` fails when files are missing or unlinked.

Regenerate the living files with `./scripts/docs`. `./scripts/check` fails when they are stale.

## Quality scripts

| Script             | What it proves                                             |
| ------------------ | ---------------------------------------------------------- |
| `./scripts/fmt`    | Formatters (gofumpt, rustfmt, prettier)                    |
| `./scripts/lint`   | gofumpt, golangci-lint, rustfmt, clippy, prettier, xo      |
| `./scripts/test`   | Go tests; Rust tests via cargo-nextest when installed      |
| `./scripts/cover`  | Go coverage floor; rust `cargo llvm-cov` when installed    |
| `./scripts/docs`   | Regenerates features and seams from tests                  |
| `./scripts/smoke`  | Local teammates + Playwright against `sopsdeck drive`      |
| `./scripts/demo`   | Product stills, clips, and walkthrough into `docs/assets/` |
| `./scripts/mutate` | Mutation testing (slow; not in `./scripts/check`)          |
| `./scripts/check`  | lint + test + cover + docs --check                         |

Rust local loop: `cd desktop/src-tauri && bacon clippy`. Stay on stable rustc; Tauri is the reason not to default to nightly. Clippy denies panics, unwrap, expect, and indexing in production code, and allows them in tests.
