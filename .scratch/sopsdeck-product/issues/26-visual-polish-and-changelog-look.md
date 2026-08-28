# Visual polish and changelog surfaces

Type: build
Status: done
Blocked by: [21](21-desktop-chrome-polish.md) (done; this is the next taste pass)

## What to build

Human QA: branding is fine; components still feel unfinished — missing icons, hover, modest motion. Changelog (site + in-app What’s new) works and is ugly.

Raise the existing vanilla UI to the concept/brand bar ([brand-spec.md](../../brand-spec.md), [sopsdeck-ui-concept.html](../../sopsdeck-ui-concept.html)). **Do not** treat shadcn/Tailwind as required. Adopt a kit only if vanilla cannot hit the bar without a pile of one-off CSS — and only if the result still matches brand, not a generic dashboard theme.

Changelog: keep `CHANGELOG.md` as the source ([22](22-epoch-semver-and-changelog.md)); restyle `site/changelog.html` and the What’s new dialog so they look like product, not a dump.

## Already there

- Primary actions use the same icon set as key rows; `withBusy` updates a `.btn-label` so icons survive loading text.
- What’s new is a card list with a short open animation. Public notes page matches the landing header/hero and numbered note cards.
- No component kit.

## Acceptance criteria

- [x] Primary actions and key rows have consistent iconography and hover/focus (extends 21).
- [x] Modest motion on selection and dialogs; no gratuitous animation.
- [x] Public changelog page and in-app What’s new are readable and on-brand (typography, spacing, hierarchy).
- [x] If a component kit is introduced, Playwright still runs against drive; brand tokens stay the source of color/type.

## Seams

- `desktop/src` CSS/HTML/JS; `site/changelog.html` via `./scripts/docs`.
- Playwright chrome tests still pass.

## Implementation (2026-08-28)

- `scripts/docs.mjs` `changelogPage` restyled. What’s new items use `whats-new-item`.

## Comments

Captured 2026-08-28 from human-found review. Kind: idea. 21 shipped the first chrome pass; this is polish + changelog look. Done 2026-08-28.
