# Specify paste, import, and editing workflows

Type: prototype
Status: resolved
Blocked by: None

## Question

How should single-value paste, dotenv/JSON/YAML bulk paste, format detection, Environment selection, duplicate keys, overwrites, multiline values, type preservation, raw/structured editing, validation, undo, and write confirmation behave in concrete desktop and CLI scenarios?

## Answer

The approved three-pane editor is the surface; no extra prototype. Clipboard is sniffed as a lone value, dotenv, JSON, or YAML. A lone value pastes into the focused key, or prompts for a key. Bulk paste opens a preview of adds / changes / conflicts and writes only on confirm. Multiline is preserved. JSON/YAML keep native types. Dotenv follows SOPS dotenv grammar; warn when Node/dotenvx quoting would be treated as literals. Structured key/value rows are the default; raw buffer is available. Undo is in-editor until save; after save, Git. CLI: `set` / stdin for the same operations, preview on TTY unless `--yes`.

## Implementation (2026-08-28)

CLI: non-empty stdin to `sopsdeck set -f FILE` is paste. Detection order: JSON object, dotenv `KEY=value`, YAML map, else lone value (needs KEY). Without `--yes`, preview lists add/change **names** only (no values) and does not write. `--yes` applies in one decrypt/encrypt. Desktop: `paste` in the open Managed File sniffs the same formats, shows an in-editor names-only preview, and writes to rows on confirm (Encrypt & save still required). Not done: clipboard sniff on app focus / preview **modal** (12 + 31); Node/dotenvx quoting warning.
