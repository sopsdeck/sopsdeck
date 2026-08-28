# Realistic Managed File fixtures

Type: build
Status: done
Blocked by: None

## What to build

Testdata and studio/demo trees look toy. Human QA wants in-the-wild files: `eas.json`, docker-compose, Dockerfile-adjacent config, and realistic **multiline** secrets — still SOPS-encrypted Managed Files, still issue 02 fidelity (structure, not formatting; eas.json warning from issue 18).

Use these in `testdata/` and/or studio demo checkout so Playwright stills and CLI tests exercise more than a flat dotenv.

Do not change eas.json product policy (issue 18). Do not encrypt a Dockerfile as if it were dotenv unless it is actually a Managed File we support.

## Already there

- `testdata/eas.json`, `compose.yaml`, `hello.multiline.env` (ciphertext + `.plain` twins).
- `get eas.json` warns `eas.json: EAS CLI will not read SOPS ciphertext` and still prints the value. Desktop shows the same copy in `file-note`.
- Demo seed adds `eas.json` and `compose.yaml` after `.env.production` so Production stays first.

## Acceptance criteria

- [x] At least one encrypted `eas.json` fixture (JSON Managed File + EAS warning path if opened).
- [x] At least one compose/YAML-style Managed File and one dotenv with a realistic multiline value.
- [x] Existing `get`/`set`/fidelity tests still pass; new fixtures have tests or demo coverage so they do not rot.

## Seams

- `testdata/`, studio demo seed, docs/demo stills if those files appear on camera.

## Implementation (2026-08-28)

- Tests: `TestGetPrintsValueFromEASJSONAndWarnsOnStderr`, `TestGetPrintsValueFromComposeYAML`, `TestGetPrintsMultilineDotenvValueContainingANewline`, `TestFilesListsComposeYAMLAndMultilineDotenv`, `TestListFindsCommittedComposeYAMLAndMultilineDotenv`.

## Comments

Captured 2026-08-28 from human-found review. Kind: idea. Spec: [02](02-establish-managed-file-fidelity.md), [18](18-specify-eas-json-handling.md). Done 2026-08-28.
