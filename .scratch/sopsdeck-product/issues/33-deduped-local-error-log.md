# Deduped local error log

Type: build
Status: done
Blocked by: None

## What to build

When running locally, errors should land in a file: same error increments a count instead of duplicating lines. That log is how humans (and agents) discover bugs to ticket.

Do not invent a hosted observability product. Keep it local, redacted (issue 13: never log secret values).

## Acceptance criteria

- [x] Desktop/CLI local errors append to a log under `SOPSDECK_STATE_DIR` (or documented equivalent).
- [x] Repeat occurrences bump a count on one record, not a new copy of the same message.
- [x] Log lines do not include secret values.

## Seams

- CLI / drive / Tauri error paths; issue [13](13-specify-failure-privacy-and-recovery-ux.md), [20](20-contextual-failure-ux.md).

## Implementation (2026-08-28)

Failed CLI commands write `$SOPSDECK_STATE_DIR/errors.json` (JSON array of `{message, count, last}`). Drive invoke uses the same writer. Age private keys and `ENC[...]` ciphertext are redacted. Usage text is skipped. Desktop UI-only JS errors are not in this slice.

## Comments

Captured 2026-08-28 from human-found review. Kind: idea.
