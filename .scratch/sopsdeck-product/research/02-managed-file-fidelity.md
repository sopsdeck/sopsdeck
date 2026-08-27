# Managed File fidelity: SOPS, dotenv, JSON, Expo `eas.json`, YAML

Question: What can SOPS and candidate parsers reliably preserve for dotenv, JSON, Expo `eas.json`, and YAML—including comments, ordering, quoting, multiline values, native types, partial encryption, and round trips—and where must Sopsdeck expose explicit limitations?

This note is evidence, not product policy. It names facts Sopsdeck would have to surface; it does not choose defaults.

## Sources

- SOPS docs: [References](https://getsops.io/docs/reference/), [Common operations](https://getsops.io/docs/usage/common-operations/), [Config file](https://getsops.io/docs/usage/identities/config-file/)
- SOPS source (`getsops/sops`): `sops.go`, `aes/cipher.go`, `stores/dotenv/store.go`, `stores/json/store.go`, `stores/yaml/store.go`
- Maintainer statements: [getsops/sops#1435](https://github.com/getsops/sops/issues/1435), [#2130](https://github.com/getsops/sops/issues/2130), [#1271](https://github.com/getsops/sops/issues/1271), [#1127](https://github.com/getsops/sops/issues/1127)
- JSON: [RFC 8259](https://www.rfc-editor.org/rfc/rfc8259)
- YAML: [YAML 1.2.2](https://yaml.org/spec/1.2.2/), [go.yaml.in/yaml/v3](https://pkg.go.dev/go.yaml.in/yaml/v3)
- Dotenv: [Node.js Environment Variables](https://nodejs.org/api/environment_variables.html) (v26), [motdotla/dotenv](https://github.com/motdotla/dotenv), [dotenvx .env format](https://dotenvx.com/docs/env-file)
- Expo: [eas.json schema](https://docs.expo.dev/eas/json/), [EAS environment variables](https://docs.expo.dev/eas/environment-variables.md), [Expo environment variables](https://docs.expo.dev/guides/environment-variables.md), [local credentials.json](https://docs.expo.dev/app-signing/local-credentials)

## SOPS encryption model (all structured stores)

SOPS treats YAML, JSON, ENV, and INI as trees. Keys stay in cleartext; leaf values are encrypted. The tree path is additional authenticated data (AEAD AAD), so substituting a ciphertext under a different key fails authentication. Binary files are a single encrypted blob under `data` in a JSON envelope, not in scope for v1 Managed File formats. ([SOPS References — types](https://getsops.io/docs/reference/); [sops.go package comment](https://github.com/getsops/sops/blob/main/sops.go))

Encrypted leaves are strings of the form:

```text
ENC[AES256_GCM,data:<b64>,iv:<b64>,tag:<b64>,type:<t>]
```

`type` is one of `str`, `int`, `float`, `bool`, `bytes`, `time`, `comment`. Encrypt serializes bools as Python-titlecase `True`/`False`, ints with `strconv.Itoa`, floats with `strconv.FormatFloat(..., 'f', -1, 64)`. Empty strings, empty byte slices, and empty comments are **not** encrypted: `Cipher.Encrypt` returns `""` for them, so they appear as plaintext empty values in an otherwise encrypted file. `nil` is walked without encryption. Ciphertext length equals plaintext length (GCM does not hide length). Unchanged values reused across decrypt→re-encrypt keep the same IV via an in-process stash, so ciphertext is stable for unmodified leaves. ([aes/cipher.go](https://github.com/getsops/sops/blob/main/aes/cipher.go); [getsops/sops#815](https://github.com/getsops/sops/issues/815))

A MAC (SHA-512 over leaf values, stored encrypted under `sops.mac`) covers **all** values by default, including unencrypted ones. `--mac-only-encrypted` / `mac_only_encrypted: true` restricts the MAC to encrypted leaves. Comments are excluded from the MAC. ([SOPS References — MAC](https://getsops.io/docs/reference/); [sops.go `Encrypt`](https://github.com/getsops/sops/blob/main/sops.go))

SOPS maintainer (felixfontein): SOPS aims to preserve **structure**, not formatting. Comments are the large exception. Perfect round-trip of presentation is not a SOPS goal, and the YAML library itself cannot do it. ([getsops/sops#2130](https://github.com/getsops/sops/issues/2130))

## Partial encryption and `.sops.yaml` creation rules

By default every leaf value is encrypted. Six mutually exclusive selectors can restrict that (CLI flags or `.sops.yaml` creation-rule fields). Using more than one in the same file is rejected. ([Common operations — Encrypting only parts of a file](https://getsops.io/docs/usage/common-operations/); [References — Creation rule settings](https://getsops.io/docs/reference/))

| Selector | Encrypts when | Comments |
|---|---|---|
| default (no selector) | every leaf | encrypted |
| `unencrypted_suffix` (default `_unencrypted`) | key does **not** end with suffix | always encrypted |
| `encrypted_suffix` | key **does** end with suffix | always encrypted |
| `unencrypted_regex` | key does **not** match | always encrypted |
| `encrypted_regex` | key **does** match | always encrypted |
| `unencrypted_comment_regex` | preceding or same-line trailing comment does **not** match | matching comments/values left plaintext |
| `encrypted_comment_regex` | preceding or same-line trailing comment **does** match | the matching comment itself is **not** encrypted |

`encrypted_suffix` / `encrypted_regex` / comment-regex selectors are the mechanism for “partial encryption”: non-matching keys stay plaintext. Unencrypted content still feeds the MAC unless `mac_only_encrypted` is set, so editing those plaintext values outside SOPS still fails integrity. Comment-regex selectors are YAML-oriented in the docs (`--encrypted-comment-regex` is introduced as “For YAML files”); dotenv has whole-line comments only; JSON has no comments.

`.sops.yaml` (not `.sops.yml`) is discovered from CWD upward, first file wins. `creation_rules` are evaluated sequentially; first `path_regex` match wins; a rule with no `path_regex` matches all remaining files. `path_regex` is relative to the config file. CLI/env identity flags ignore the config file. Store options under `stores:`: JSON/YAML `indent` only; `dotenv` and `ini` currently accept no keys. ([Config file](https://getsops.io/docs/usage/identities/config-file/); [References — Config file format](https://getsops.io/docs/reference/))

## Dotenv

### There is no single dotenv spec

Node.js documents that `.env` files have **no formal specification** and then defines its own. motdotla/dotenv and dotenvx each document a richer grammar than SOPS implements. ([Node.js Environment Variables — DotEnv](https://nodejs.org/api/environment_variables.html); [motdotla/dotenv README](https://github.com/motdotla/dotenv); [dotenvx .env](https://dotenvx.com/docs/env-file))

### What SOPS actually implements

The dotenv store splits on `\n`. Empty lines are skipped. A line starting with `#` is a comment (`Comment{Value: line[1:]}`). Any other line must contain `=`; key is everything before the first `=`, value is everything after, with the literal two-character sequence `\n` replaced by a newline. Quotes are **not** parsed. Whitespace around keys/values is **not** trimmed. Inline `#` is part of the value. `export KEY=...` is a key named `export KEY`. Complex (nested) values are rejected on emit. ([stores/dotenv/store.go](https://github.com/getsops/sops/blob/main/stores/dotenv/store.go); maintainer restatement in [#1127](https://github.com/getsops/sops/issues/1127) and [#1271](https://github.com/getsops/sops/issues/1271))

On emit, comments become `#` + value + `\n`. String values have newlines re-escaped as `\n`. Non-string values are stringified. There is no quoting of values that contain `#`, spaces, or `=`.

CRLF: a `\r`-only “blank” line (Windows CRLF split on `\n`) is not empty, does not start with `#`, and has no `=`, so load fails (`invalid dotenv input line`). Unix LF is required. ([getsops/sops#779](https://github.com/getsops/sops/issues/779))

### Round-trip: not safe

Maintainer (felixfontein), still-open architecture issue: “INI and DotEnv stores are not roundtrip-safe, and quoting is generally a problem.” Because `\n` is escaped but `\` itself is not, a literal backslash-n in plaintext and a real newline cannot be distinguished; both become newlines after decrypt. A dotenv-v2 format was discussed; it has not shipped. An earlier quoting-aware parser was reverted as a breaking change (#706). ([getsops/sops#1435](https://github.com/getsops/sops/issues/1435))

Empty lines are dropped on load, so blank-line grouping is lost. User key order is preserved (the tree is an ordered slice). SOPS metadata is flattened to `sops_*` keys, sorted alphabetically on emit (stable since [PR #1101](https://github.com/getsops/sops/pull/1101)). Whole-line comments are preserved as comments and encrypted by default (`type:comment`). Inline comments are not a SOPS dotenv feature.

### Candidate parsers diverge from SOPS

| Feature | SOPS dotenv store | Node.js spec / motdotla/dotenv | dotenvx |
|---|---|---|---|
| Formal spec | Primitive line grammar | Node.js defines its own; states there is no formal spec | Documents its own cheat sheet |
| Values | Always strings; quotes are literal characters | Always strings; quotes are delimiters | Always strings; quotes are delimiters |
| Inline `#` comments | Not recognized; become part of the value | Yes, unless quoted | Yes, when `#` follows a quoted value or is its own line; `VAL#not` is not a comment |
| Quote stripping | No | Yes (`'quoted'` → `quoted`) | Yes |
| Unquoted trim | No | Yes (Node.js and motdotla) | Interpolation rules; hash after unquoted value is a comment only with space (`VAL # comment`) |
| Real-newline multiline | No (only `\n` escape) | Yes, quoted values may span lines (Node.js; motdotla ≥ v15) | Backtick multiline; `\n` in double quotes |
| `\n` vs `\\n` | Ambiguous (not round-trip safe) | Double-quoted `\n` expands | `\n` `\r` `\t` `\\` in double quotes |
| Interpolation | None | motdotla: use dotenvx | Unquoted and double-quoted |
| `export` prefix | Becomes part of the key | Ignored (Node.js) | Not in the SOPS grammar |
| Empty lines | Dropped | Skipped at parse (lost if rewritten) | Ignored |

`sops exec-env` therefore exports quoted strings **including the quote characters**, which matches the SOPS grammar and surprises users of Node/shell dotenv. ([getsops/sops#1127](https://github.com/getsops/sops/issues/1127))

If Sopsdeck parses dotenv with motdotla/dotenv or dotenvx, then encrypts with SOPS (or the reverse), quotes, `#`, whitespace, and multiline will not mean the same thing on both sides.

## JSON (including Expo `eas.json`)

### JSON the format

RFC 8259: objects are **unordered** collections of name/value pairs; arrays are ordered; values are string, number, `true`, `false`, `null`, object, or array. Names SHOULD be unique. Insignificant whitespace is only space, tab, LF, CR. The grammar has **no comments**. Strings are double-quoted. Numbers have no Infinity/NaN; leading zeros are not allowed. Implementations differ on whether member order is visible. ([RFC 8259 §§1, 2, 4, 6](https://www.rfc-editor.org/rfc/rfc8259))

JSONC / JSON5 comments are extensions, not JSON. Go `encoding/json` (what SOPS uses) rejects them.

### What SOPS preserves

The JSON store tokenizes with `json.Decoder` (`UseNumber()` then normalize to `int` if the value fits `int64`, else `float64`). That keeps object member **order** in the tree (a slice, not a Go map). On emit it rebuilds JSON and runs `json.Indent`. Default indent is one tab (`indent: -1`); `0` is compact; positive is that many spaces. A trailing newline is always appended. Comment tree items are **skipped** on emit. Top-level arrays, numbers, strings, and `null` are rejected: SOPS needs a top-level object so it can insert a `sops` metadata key. ([stores/json/store.go](https://github.com/getsops/sops/blob/main/stores/json/store.go); [References — Top-level arrays, JSON indentation](https://getsops.io/docs/reference/))

Lost on any encrypt or decrypt rewrite:

- Original whitespace and indent style (rewritten to configured indent)
- Original quoting of numbers vs strings (types are native; strings always `"..."`)
- Key quoting alternatives (JSON only allows double quotes anyway)
- Comments, trailing commas, single quotes (invalid JSON; load fails)
- Top-level arrays
- Integer values that do not fit `int64` become `float64` (the `UseNumber` path exists specifically to avoid silent `float64` precision loss for integers that *do* fit `int64`)

Preserved:

- Object key order in the SOPS tree (then re-emitted in that order)
- Native types: string / int / float / bool / null (null stays unencrypted)
- Nested objects and arrays under a top-level object
- Partial encryption by key suffix or regex (same as YAML)

Multiline: JSON strings may contain escaped `\n` or (in the source) escaped Unicode; after round-trip they are ordinary JSON strings as `encoding/json` emits them, not source-layout multiline.

### Expo `eas.json`

`eas.json` is documented as JSON. The schema is a top-level object (`cli`, `build`, `submit`, …). That shape **is** something SOPS can wrap: it is not a top-level array. After encryption it is no longer valid input for EAS CLI: leaf strings/bools/numbers become `ENC[...]` strings and a `sops` object is inserted. Consumers must decrypt first. ([EAS: Configuration with eas.json](https://docs.expo.dev/eas/json/))

What is secret vs config in Expo’s own model:

- **`build.*.env`**: object of environment variables set during the build. Docs: “It should only be used for values that you would commit to your git repository and not for passwords or [secrets](https://docs.expo.dev/eas/environment-variables.md#visibility-settings-for-environment-variables).” ([eas.json schema — `env`](https://docs.expo.dev/eas/json/))
- **`build.*.environment`**: selects an EAS environment (`development` / `preview` / `production`) whose variables live **on EAS**, not in the file. ([eas.json schema — `environment`](https://docs.expo.dev/eas/json/); [EAS environment variables](https://docs.expo.dev/eas/environment-variables.md))
- **EAS environment variables** (dashboard/CLI): visibility `plaintext`, `sensitive`, or `secret`. Secret values are not readable outside EAS servers. These are not fields inside `eas.json`.
- **`EXPO_PUBLIC_` variables**: inlined into the client bundle; Expo says do not store sensitive info there. ([Environment variables in Expo](https://docs.expo.dev/guides/environment-variables.md))
- **Signing secrets**: `credentials.json` (keystore passwords, distribution cert passwords) is a **separate** JSON file, not `eas.json`. `eas.json` only has `credentialsSource: local | remote`. ([local credentials](https://docs.expo.dev/app-signing/local-credentials))

`eas.json` therefore mixes non-secret build config (node version, profiles, `environment` name, committable `env`) with optional `env` values that Expo tells people not to treat as secrets. SOPS can encrypt any JSON leaf; Expo’s schema does not mark which keys are secrets. Partial encryption (`encrypted_regex` on names like `env`, or leaving profile metadata plaintext) is a creation-rule choice, not something the eas.json schema encodes.

`eas.json` cannot carry JSON comments. Native types in the schema include booleans (`developmentClient`, `autoIncrement`), numbers (`rollout`), nested objects, and string maps (`env`). SOPS will type-tag those leaves (`type:bool`, `type:int`/`float`, `type:str`). After decrypt, types round-trip if the JSON store’s int/float split matches what was encrypted.

## YAML

### YAML the format

YAML 1.2.2: comments, scalar style (plain / single-quoted / double-quoted / literal `|` / folded `>`), indentation, and node style are **presentation** details. Parsing discards presentation. Construction of native data must not depend on comments, key order, or style. Mapping keys are unordered in the representation graph; order is a serialization detail. Comments begin with `#` and run to end of line; they are not part of scalars. ([YAML 1.2.2 §§3.1–3.2, comments, scalar styles](https://yaml.org/spec/1.2.2/))

So “preserve quoting and whitespace” is not a YAML-representation guarantee even before SOPS. Any library that round-trips through native values will pick a style on emit.

### What SOPS preserves

The YAML store uses `go.yaml.in/yaml/v3`. It walks `yaml.Node`, keeping mapping order in `TreeBranch` slices and storing comments as `sops.Comment` items. Since [PR #2131](https://github.com/getsops/sops/pull/2131) (merged 2026-04-17), `Comment.Inline` distinguishes line comments from head comments so trailing comments can be restored as `LineComment` instead of being collapsed to the next key’s `HeadComment`. That fix is in current mainline; SOPS 3.12.2 still collapsed inline comments (the bug report). Foot comments already round-tripped. ([stores/yaml/store.go](https://github.com/getsops/sops/blob/main/stores/yaml/store.go); [#2130](https://github.com/getsops/sops/issues/2130) / [#2131](https://github.com/getsops/sops/pull/2131))

Documented limits:

- **Anchors/aliases**: not supported. Paths used as AEAD AAD break if anchors rewrite structure at load time. ([References — YAML anchors](https://getsops.io/docs/reference/))
- **Top-level sequences or scalars**: not supported; same `sops` metadata-key reason as JSON. Nested sequences under a mapping are fine. ([References — Top-level arrays](https://getsops.io/docs/reference/); `yaml/store.go` error `YAML documents that are sequences are not supported`)
- **Multi-document streams**: supported; MAC is over the physical file, not per document. ([References — YAML Streams](https://getsops.io/docs/reference/))
- **Indent**: default 4 spaces; emitter only accepts 2–9; 1 or ≥10 become 2. ([References — YAML indentation](https://getsops.io/docs/reference/))
- **Key order**: generally preserved; `sops` metadata is appended at the bottom of the first document with SOPS’s internal (not alphabetical) field order. ([getsops/sops#1445](https://github.com/getsops/sops/issues/1445))

Scalar style (`"foo"` vs `'foo'` vs `foo` vs `|`) is **not** stored in the tree. Emit uses `yaml.Node.Encode` / the v3 encoder, which chooses a style. Literal block scalars become ordinary strings in the tree; re-emit may use a different style. Duplicate keys are rejected (an extra decode-into-map uniqueness check).

### Native types

Scalars are `node.Decode(&result)` into `interface{}`. `true`/`false` become bool; integers/floats become native numbers; `null` stays nil (unencrypted). YAML 1.1 `yes`/`no`/`on`/`off` decoded into `interface{}` remain **strings** in go-yaml v3 (1.1 bools apply only when the target is a typed `bool`). Octals still follow YAML 1.1 `0777` as well as YAML 1.2 `0o777`. ([go.yaml.in/yaml/v3 compatibility](https://pkg.go.dev/go.yaml.in/yaml/v3))

Quoted `"true"` vs unquoted `true` therefore matter: the former is `type:str`, the latter `type:bool`. After round-trip the bool will be emitted as a YAML boolean, not necessarily with the original quotes.

Multiline: `|` / `>` content is a string leaf and encrypts as `type:str`. Newlines survive in the value; block vs folded vs quoted style does not.

## Encrypt / decrypt / edit round trips — what breaks

A typical SOPS cycle is load → tree → encrypt/decrypt → emit. Emit is a serializer, not a copy of the original bytes.

| Concern | dotenv | JSON / `eas.json` | YAML |
|---|---|---|---|
| Byte-exact file | No | No | No |
| Comments | Whole-line only; encrypted by default; inline `#` is data | None (invalid JSON) | Yes, as structure; inline vs head needs SOPS with #2131; formatting/whitespace around comments not exact |
| Key order | User keys yes; `sops_*` metadata sorted | Yes in tree, re-emitted in that order | Yes; `sops` block appended with internal order |
| Quoting | Quotes are data; never added/stripped by SOPS | Always JSON double quotes; style rewritten | Style chosen by encoder, not original |
| Whitespace / blank lines | Blank lines dropped; no trim | Fully rewritten (`json.Indent`) | Indent rewritten (2–9 spaces); blank-line layout not a tree field |
| Multiline | Only `\n` escape; ambiguous with literal `\n` | JSON string escapes | Value preserved; block style not preserved |
| Native types | Strings only on emit | str/int/float/bool/null | str/int/float/bool/null; quote-sensitive for `true`/`123` |
| Empty values | Left plaintext `KEY=` | `""` and `null` left plaintext | `""` and `null` left plaintext |
| Top-level shape | Flat `KEY=value` only | Must be `{...}` | Must be a mapping; no top-level sequence |
| CRLF | Breaks | JSON whitespace, but emit is LF | Emit is LF |
| Partial encryption | Suffix/regex on keys; no comment-regex in practice | Suffix/regex on keys | Suffix/regex and comment-regex |
| Downstream consumer | Node/dotenvx parse quotes/`#` differently | EAS CLI will not accept encrypted `eas.json` | Anchors, tags, merge keys not SOPS-safe |

Decrypt → edit in an editor → encrypt additionally depends on the editor/parser. If the editor is SOPS itself (`sops edit`), the tree is the source of truth. If Sopsdeck uses a different parser (JSONC, yaml, dotenvx) and writes back through SOPS, every divergence in the tables above is a user-visible mutation.

## Limitations Sopsdeck would have to surface

These are capabilities and non-capabilities of the formats and of SOPS. They are not product decisions.

1. **SOPS is not a pretty-printer of the original file.** Structure (keys, nested collections, typed leaves, and — for YAML/dotenv — comments it understands) is what it keeps. Indent, quote style, blank lines, and JSON/YAML layout are rewritten.
2. **Dotenv in SOPS is a different language from Node dotenv / dotenvx / Node.js’s documented `.env` grammar.** Quotes, inline comments, trimming, `export`, interpolation, and real multiline will not mean the same thing unless Sopsdeck uses the SOPS grammar end-to-end — or documents that it does not.
3. **SOPS dotenv is not round-trip safe for `\n` vs newline**, and drops empty lines. CRLF files can fail to parse. That is an open SOPS architecture issue (#1435), not a local bug.
4. **JSON cannot have comments.** JSONC `eas.json` or commented `package`-style JSON is not SOPS JSON. Trailing commas fail.
5. **Top-level JSON arrays and YAML sequences cannot be SOPS files.** They must be wrapped in an object/mapping.
6. **`eas.json` is ordinary JSON SOPS can encrypt, but an encrypted `eas.json` is not a valid EAS config until decrypted.** Expo’s own docs put secrets in EAS environment variables / `credentials.json`, and tell people not to put passwords in `eas.json` `env`. SOPS will still encrypt whatever leaves a creation rule selects.
7. **Empty strings are not encrypted** in any store. They remain visible `""` / `KEY=`.
8. **Unencrypted values still sit under the MAC** unless `mac_only_encrypted` is on. Editing plaintext leaves with a non-SOPS tool invalidates the file.
9. **Comment-based partial encryption is a YAML feature.** Dotenv only has whole-line comments; JSON has none. Suffix/regex partial encryption applies to all three structured stores.
10. **Creation-rule selectors are mutually exclusive** and first `path_regex` match wins. Coexistence with an existing `.sops.yaml` is a file SOPS will honor from CWD, not from the target file’s directory (documented CWD lookup, Issue 242).
11. **YAML anchors, aliases, and merge keys are unsupported.** YAML 1.1-ish `yes`/`on` into `interface{}` stay strings; quoted vs unquoted `true`/`1` change encrypted type.
12. **YAML inline-comment position** requires a SOPS build that includes #2131. Older SOPS turns trailing comments into preceding-line comments on the first encrypt.
13. **Parser choice is a fidelity choice.** Using motdotla/dotenv, dotenvx, `JSON.parse`, JSONC, or a non-v3 YAML library alongside SOPS will disagree with SOPS on comments, types, and quoting. The SOPS tree is the only representation SOPS will MAC and re-encrypt.
14. **Ciphertext length reveals plaintext length.** Diffs of encrypted files show which keys changed; they also leak value length.

## Citations (compact)

1. https://getsops.io/docs/reference/
2. https://getsops.io/docs/usage/common-operations/
3. https://getsops.io/docs/usage/identities/config-file/
4. https://github.com/getsops/sops/blob/main/sops.go
5. https://github.com/getsops/sops/blob/main/aes/cipher.go
6. https://github.com/getsops/sops/blob/main/stores/dotenv/store.go
7. https://github.com/getsops/sops/blob/main/stores/json/store.go
8. https://github.com/getsops/sops/blob/main/stores/yaml/store.go
9. https://github.com/getsops/sops/issues/1435
10. https://github.com/getsops/sops/issues/2130
11. https://github.com/getsops/sops/pull/2131
12. https://github.com/getsops/sops/issues/1271
13. https://github.com/getsops/sops/issues/1127
14. https://github.com/getsops/sops/issues/1445
15. https://github.com/getsops/sops/issues/779
16. https://github.com/getsops/sops/pull/1101
17. https://www.rfc-editor.org/rfc/rfc8259
18. https://yaml.org/spec/1.2.2/
19. https://pkg.go.dev/go.yaml.in/yaml/v3
20. https://nodejs.org/api/environment_variables.html
21. https://github.com/motdotla/dotenv
22. https://dotenvx.com/docs/env-file
23. https://docs.expo.dev/eas/json/
24. https://docs.expo.dev/eas/environment-variables.md
25. https://docs.expo.dev/guides/environment-variables.md
26. https://docs.expo.dev/app-signing/local-credentials
