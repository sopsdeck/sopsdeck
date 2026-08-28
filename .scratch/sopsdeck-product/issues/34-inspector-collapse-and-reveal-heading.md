# Inspector collapse and value-column reveal

Type: build
Status: done
Blocked by: None

## What to build

The right inspector should collapse by section, like VS Code panels. Toolbar **Add secret** can go; reveal/hide values becomes a show/hide icon on the values column heading (next to “Value”). Composer stays the add path ([24](24-editor-key-row-actions.md)).

Do not change the three-pane IA. Do not add a clipboard modal ([31](31-deferred-product-ideas.md)).

## Acceptance criteria

- [x] Each inspector section is collapsible; state can persist for the session (localStorage is fine).
- [x] Add secret toolbar button is gone; composer remains.
- [x] Values column heading has a reveal/hide icon that matches today’s Reveal values behavior.

## Seams

- Playwright chrome tests (`data-testid` on heading reveal and inspector sections).

## Implementation (2026-08-28)

Inspector File / Access / Publish / Git headings toggle collapse; collapsed ids live in `localStorage` `sopsdeck-inspector`. Toolbar Add secret and Reveal values are gone. The Value column heading uses `data-testid="reveal"`. Composer is unchanged.

## Comments

Captured 2026-08-28 from human-found review. Kind: idea. Spec: [04](04-validate-folder-first-workspace.md), [24](24-editor-key-row-actions.md).
