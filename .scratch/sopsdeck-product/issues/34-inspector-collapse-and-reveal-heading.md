# Inspector collapse and value-column reveal

Type: build
Status: open
Blocked by: None

## What to build

The right inspector should collapse by section, like VS Code panels. Toolbar **Add secret** can go; reveal/hide values becomes a show/hide icon on the values column heading (next to “Value”). Composer stays the add path ([24](24-editor-key-row-actions.md)).

Do not change the three-pane IA. Do not add a clipboard modal ([31](31-deferred-product-ideas.md)).

## Acceptance criteria

- [ ] Each inspector section is collapsible; state can persist for the session (localStorage is fine).
- [ ] Add secret toolbar button is gone; composer remains.
- [ ] Values column heading has a reveal/hide icon that matches today’s Reveal values behavior.

## Seams

- Playwright chrome tests (`data-testid` on heading reveal and inspector sections).

## Comments

Captured 2026-08-28 from human-found review. Kind: idea. Spec: [04](04-validate-folder-first-workspace.md), [24](24-editor-key-row-actions.md).
