# Specify failure, privacy, and operational recovery behavior

Type: grilling
Status: resolved
Blocked by: 07, 09

## Question

For crashes, interrupted writes, SOPS/Git/provider failures, locked keychains, unavailable networks, malformed files, clipboard use, logs, update checks, and partial Sync Target operations, what must be atomic, recoverable, redacted, retryable, and visible to the user?

## Answer

Encrypted writes are temp-file + rename in the same directory so a crash leaves the previous ciphertext. Malformed or undecryptable files are shown, not overwritten. Keychain locked: OS prompt, then retry. Git and GitHub errors are visible and retryable; Publish does not prune if any `PUT` failed. Logs, crash reports, and update checks carry no secret values and no private keys (key names and paths are allowed). Clipboard is user-initiated; v1 does not auto-clear. No analytics.

## Implementation (2026-08-28)

Atomic ciphertext writes exist on set. Git/GitHub failures are visible today as raw command text in a single editor `#error` — that presentation is [20](20-contextual-failure-ux.md), not a change to this answer.

