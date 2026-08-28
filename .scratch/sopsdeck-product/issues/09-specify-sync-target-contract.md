# Specify Sync Target mapping, ownership, and pruning

Type: grilling
Status: resolved
Blocked by: 05, 08

## Question

How are Environments and keys mapped to provider scopes; how do prefixing, selection, ownership tracking, dry-run review, updates, missing values, provider drift, and opt-in pruning behave without deleting unowned secrets or making a provider canonical?

## Answer

v1 Publish target is GitHub Actions **repository** secrets and **environment** secrets. Mappings live in `.sopsdeck.toml`: Managed File → repo and optional GitHub environment, optional prefix, selected keys (default all). Publish always `PUT`s (GitHub is write-only). Last-published **names** may be recorded in the manifest; value fingerprints stay machine-local.

Auth is existing `gh` / GitHub CLI / OS-backed token — not a Sopsdeck cloud OAuth. Dry-run is the default preview. Prune defaults **off**. When on, delete only names that match the prefix **and** were previously published by Sopsdeck, and only after preview. Never delete unprefixed extras. Missing local keys skip that GitHub name (no implicit delete unless prune). GitHub is never canonical. Dependabot, Codespaces, and org secrets are out of v1.

## Implementation (2026-08-28)

`sopsdeck publish -f FILE [--prefix] [--yes] [--prune]` talks to `SOPSDECK_GITHUB_API` (studio uses `internal/githubfake`). Dry-run unless `--yes`. `--mapping` prints the resolved repo, environment, prefix, and keys without calling GitHub. Drive invoke `publish_managed_file` and `get_publish_mapping`. Inspector Publish shows mapping, prefix, and opt-in prune. `.sopsdeck.toml` maps a Managed File to `repo`, optional `environment`, `prefix`, `keys`, and last-published `published` names; CLI `--prefix` overrides the mapping; prune deletes only previously published prefixed names. Environment mappings `PUT` `/repos/{owner}/{repo}/environments/{environment}/secrets/{name}`. Auth is `GH_TOKEN`, `GITHUB_TOKEN`, or `gh auth token` (Bearer). Fake PUT bodies are not GitHub-encrypted values — fine for the local fake, not for real GitHub.

