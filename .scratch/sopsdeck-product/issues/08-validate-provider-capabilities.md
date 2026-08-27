# Validate GitHub and Infisical-like sync capabilities

Type: research
Status: resolved
Blocked by: None

## Question

Using first-party APIs, CLIs, and documentation, which GitHub repository and environment secret operations are available, what authentication and metadata they expose, and how Infisical-style prefixing and optional prune of provider secrets not present in the project actually work — and what limitations constrain Sopsdeck's first-release GitHub Sync Target contract?

## Answer

GitHub Actions secrets are a write-only Sync Target: REST list/get return `name`, `created_at`, and `updated_at` only, never plaintext or ciphertext, so a client cannot know whether a GitHub secret matches local plaintext unless it stores its own fingerprint or always overwrites. Create and update are a single Libsodium-sealed-box `PUT` (`201` vs `204` only distinguish new vs existing names); delete is by name; GraphQL has no secret CRUD. Repository Actions secrets need Secrets write (`repo` on a classic PAT); environment secrets are a separate API under Environments write, require a pre-existing environment, and the product UI requires owner/admin. Dependabot, Codespaces, and organization Actions secrets are different stores/scopes. Names are stored uppercase with a 100-secret / 48 KB cap per repo and per environment. Infisical’s GitHub sync always `PUT`s every selected key (GitHub cannot import), optionally prefixes via a Handlebars key schema so prune only touches matching names, and gates deletion with `disableSecretDeletion` because GitHub has no ownership metadata — without a prefix, prune means delete every destination secret not in the current set. Findings with citations: [research/08-github-sync-capabilities.md](../research/08-github-sync-capabilities.md).
