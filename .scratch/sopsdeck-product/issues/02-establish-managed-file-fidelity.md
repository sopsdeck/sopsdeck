# Establish the Managed File fidelity contract

Type: research
Status: resolved
Blocked by: None

## Question

What can SOPS and candidate parsers reliably preserve for dotenv, JSON, Expo `eas.json`, and YAML—including comments, ordering, quoting, multiline values, native types, partial encryption, and round trips—and where must Sopsdeck expose explicit limitations?

## Answer

SOPS encrypts leaf values and leaves keys in cleartext; it preserves structure (tree key order, nested collections, typed leaves, and comments it understands), not byte-exact formatting, and empty strings/`null` stay plaintext while unencrypted leaves still feed the MAC unless `mac_only_encrypted` is set. Partial encryption is key suffix/regex on all structured stores or comment-regex on YAML; those six selectors are mutually exclusive and `.sops.yaml` creation rules are first-match. SOPS dotenv is a primitive line grammar (whole-line `#`, first `=`, literal quotes, `\n` escape only, empty lines dropped, CRLF unsafe) and is not Node dotenv, Node.js’s documented `.env` grammar, or dotenvx—maintainers document it as not round-trip safe. JSON, including Expo `eas.json`, has no comments (RFC 8259); SOPS requires a top-level object, re-indents, and keeps native types and member order, so `eas.json` is wrap-able JSON but an encrypted file is not valid EAS input until decrypted (Expo puts secrets in EAS environment variables and `credentials.json`, and says `eas.json` `env` is for committable values). YAML keeps mapping order and comments (inline vs head needs SOPS with #2131), but quote style, `|`/`>` layout, and indent are rewritten, anchors and top-level sequences are unsupported, and quoted `"true"` vs unquoted `true` changes encrypted type. Sopsdeck would have to surface: no pretty original file; dotenv parser mismatch; JSON/YAML rewrite; encrypted `eas.json` unusable by EAS CLI; empty values unencrypted; comment-regex YAML-only; `.sops.yaml` CWD lookup. Full evidence: [02-managed-file-fidelity.md](../research/02-managed-file-fidelity.md).
