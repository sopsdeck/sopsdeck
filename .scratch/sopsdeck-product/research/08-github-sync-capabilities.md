# GitHub Sync Target capabilities (research)

Date: 2026-08-27  
Ticket: `.scratch/sopsdeck-product/issues/08-validate-provider-capabilities.md`

This note records first-party API, CLI, and documentation facts about GitHub Actions secrets and Infisical’s GitHub Secret Sync. Infisical is a UX/capability reference, not a Sopsdeck dependency. This note does not decide Sopsdeck product policy.

## Sources

GitHub:

- [REST API endpoints for GitHub Actions Secrets](https://docs.github.com/en/rest/actions/secrets)
- [Encrypting secrets for the REST API](https://docs.github.com/en/rest/guides/encrypting-secrets-for-the-rest-api)
- [Secrets (Actions concepts)](https://docs.github.com/en/actions/concepts/security/secrets)
- [Secrets reference](https://docs.github.com/en/actions/reference/security/secrets)
- [Using secrets in GitHub Actions](https://docs.github.com/en/actions/how-tos/write-workflows/choose-what-workflows-do/use-secrets)
- [Understanding GitHub secret types](https://docs.github.com/en/code-security/reference/secret-security/secret-types)
- [REST API endpoints for Codespaces repository secrets](https://docs.github.com/en/rest/codespaces/repository-secrets)
- [REST API endpoints for Dependabot secrets](https://docs.github.com/en/rest/dependabot/secrets)
- [Permissions required for fine-grained personal access tokens](https://docs.github.com/en/rest/authentication/permissions-required-for-fine-grained-personal-access-tokens)
- [Permissions required for GitHub Apps](https://docs.github.com/en/rest/authentication/permissions-required-for-github-apps)
- [Rate limits for the REST API](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api)
- [GraphQL Actions reference](https://docs.github.com/en/graphql/reference/actions)
- [GraphQL Repos reference](https://docs.github.com/en/graphql/reference/repos)
- [`gh secret set`](https://cli.github.com/manual/gh_secret_set), [`gh secret list`](https://cli.github.com/manual/gh_secret_list), [`gh secret delete`](https://cli.github.com/manual/gh_secret_delete)
- [Libsodium sealed boxes](https://libsodium.gitbook.io/doc/public-key_cryptography/sealed_boxes)

Infisical:

- [GitHub Secret Sync](https://infisical.com/docs/integrations/secret-syncs/github)
- [Secret Syncs overview](https://infisical.com/docs/integrations/secret-syncs/overview)
- [Create GitHub Sync API](https://infisical.com/docs/api-reference/endpoints/secret-syncs/github/create)
- [Delete GitHub Sync API](https://infisical.com/docs/api-reference/endpoints/secret-syncs/github/delete)
- [GitHub App Connection](https://infisical.com/docs/integrations/app-connections/github)
- First-party implementation: [`github-sync-fns.ts`](https://github.com/Infisical/infisical/blob/main/backend/src/services/secret-sync/github/github-sync-fns.ts), [`secret-sync-fns.ts`](https://github.com/Infisical/infisical/blob/main/backend/src/services/secret-sync/secret-sync-fns.ts), [`github-sync-constants.ts`](https://github.com/Infisical/infisical/blob/main/backend/src/services/secret-sync/github/github-sync-constants.ts), [`github-sync-types.ts`](https://github.com/Infisical/infisical/blob/main/backend/src/services/secret-sync/github/github-sync-types.ts)

## 1. GitHub secret stores are separate

GitHub documents three independent secret types. Writing an Actions repository secret does not create a Codespaces or Dependabot secret of the same name.

| Store | Scope | Used by | REST prefix |
| --- | --- | --- | --- |
| Actions | organization, repository, environment | Actions workflows | `/actions/secrets` |
| Dependabot | organization, repository (no environment) | Dependabot and Dependabot-triggered workflows | `/dependabot/secrets` |
| Codespaces | user, organization, repository (no environment) | Codespaces | `/codespaces/secrets` |

Evidence:

- [Understanding GitHub secret types](https://docs.github.com/en/code-security/reference/secret-security/secret-types): Actions secrets are only available in Actions workflows; Dependabot cannot read Actions secrets; GitHub Actions cannot access Codespaces secrets.
- `gh secret set --app {actions|agents|codespaces|dependabot}` selects which store to write ([`gh secret set`](https://cli.github.com/manual/gh_secret_set)). Default for a repository secret is Actions.

`gh` also documents an `agents` application. That is a fourth CLI-selectable store; it is not the Actions repository/environment API Sopsdeck would use for a first-release Actions Sync Target.

A first-release GitHub Actions Sync Target that only calls `/repos/{owner}/{repo}/actions/secrets` (and optionally environment Actions secrets) will not populate Dependabot or Codespaces.

## 2. Actions REST operations (repository, environment, organization)

All list/get endpoints are documented as returning metadata **without revealing encrypted values**. Create and update are a single `PUT`. There is no PATCH, no value GET, and no compare endpoint.

### Repository Actions secrets

| Operation | Method | Path | Success |
| --- | --- | --- | --- |
| List | `GET` | `/repos/{owner}/{repo}/actions/secrets` | `200` `{ total_count, secrets[] }` |
| Get one | `GET` | `/repos/{owner}/{repo}/actions/secrets/{secret_name}` | `200` |
| Get public key | `GET` | `/repos/{owner}/{repo}/actions/secrets/public-key` | `200` `{ key_id, key }` |
| Create or update | `PUT` | `/repos/{owner}/{repo}/actions/secrets/{secret_name}` | `201` create / `204` update |
| Delete | `DELETE` | `/repos/{owner}/{repo}/actions/secrets/{secret_name}` | `204` |

List/get item fields: `name`, `created_at`, `updated_at`. No `value`, no hash, no `key_id` of the stored ciphertext.

Classic PAT / OAuth: `repo` scope. REST text: authenticated users need collaborator access to create, update, or read secrets. Public-key GET: anyone with read access to the repository.

### Environment Actions secrets

Same operations under `/repos/{owner}/{repo}/environments/{environment_name}/secrets` (and `.../public-key`, `.../{secret_name}`). Environment names in the path must be URL-encoded (`/` → `%2F`).

List/get item fields are the same as repository secrets: `name`, `created_at`, `updated_at`.

The environment is a path parameter. These endpoints do not create the environment. GitHub’s environment how-to requires repository owner (personal repos) or `admin` (org repos) to configure environments, and states that only repository admins can configure an environment even though anyone who can edit workflows can cause one to be created by referencing it ([Managing environments](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments)).

Fine-grained PAT / GitHub App: environment secret CRUD is under **Environments** write, not under **Secrets** write ([PAT permissions](https://docs.github.com/en/rest/authentication/permissions-required-for-fine-grained-personal-access-tokens), [GitHub App permissions](https://docs.github.com/en/rest/authentication/permissions-required-for-github-apps)).

### Organization Actions secrets

Same create/update/delete/list/get/public-key pattern under `/orgs/{org}/actions/secrets`. Extra fields: `visibility` (`all` | `private` | `selected`) and `selected_repositories_url`. Create/update requires `visibility`; `selected` also requires `selected_repository_ids`. Separate endpoints add/remove selected repositories.

Classic PAT: `admin:org` (plus `repo` if the repository is private). Fine-grained PAT / GitHub App: organization **Secrets** write.

A repository can list org secrets that are shared with it via `GET /repos/{owner}/{repo}/actions/organization-secrets` (names + timestamps only). That list is not a write API for org secrets.

GitHub Free: organization-level secrets and variables are not accessible by private repositories ([Using secrets](https://docs.github.com/en/actions/how-tos/write-workflows/choose-what-workflows-do/use-secrets)).

### Pagination

List endpoints take `per_page` (max 100, default 30) and `page`. A repository or environment can hold 100 secrets, so two pages can be required. An organization can hold 1,000, so listing for prune needs pagination.

## 3. Metadata vs values: what GitHub will and will not tell a client

After a write, GitHub will not return the plaintext or the stored ciphertext.

Documented list/get payload for a repository or environment secret:

```json
{
  "name": "SECRET_NAME",
  "created_at": "2019-08-10T14:59:22Z",
  "updated_at": "2020-01-10T14:59:22Z"
}
```

Implications:

- **Last-updated time is visible.** `updated_at` is a required date-time on list and get.
- **Values are not readable after write.** Every list/get description says “without revealing their encrypted value.” Infisical encodes this as `canImportSecrets: false` and `getSecrets` throws “GitHub does not support importing secrets.”
- **`PUT` 201 vs 204 only distinguishes name create vs name already present**, not whether the new plaintext differs from the old one.
- **`updated_at` cannot prove value equality.** A client that `PUT`s the same plaintext again still performs a write. GitHub does not document that `updated_at` is left unchanged when the value is identical. Even if it were, a matching timestamp would not prove the remote value equals local plaintext.
- **Ciphertext cannot be used as a fingerprint.** Writes must be Libsodium sealed boxes of the plaintext under the repository/environment public key ([Encrypting secrets](https://docs.github.com/en/rest/guides/encrypting-secrets-for-the-rest-api)). Libsodium sealed boxes create a new ephemeral key pair per message, so encrypting the same plaintext twice yields different ciphertext ([Sealed boxes](https://libsodium.gitbook.io/doc/public-key_cryptography/sealed_boxes)). A client that does not hold GitHub’s private key cannot decrypt, and cannot compare two sealed boxes to test equality.

Therefore a client **cannot know whether a GitHub secret matches local plaintext unless it stores its own hash/fingerprint of the last successfully written value** (or always overwrites). GitHub does not provide that hash.

## 4. Encryption contract

GitHub Actions secrets use Libsodium sealed boxes so the value is encrypted before it reaches GitHub ([Secrets concepts](https://docs.github.com/en/actions/concepts/security/secrets)). REST create/update bodies require:

- `encrypted_value`: Base64 sealed box under the public key from the matching `.../public-key` endpoint
- `key_id`: the `key_id` from that public-key response

The public key is **not secret**. Repository and environment public-key GETs are allowed for anyone with read access to the repository (private repos still need `repo` on a classic PAT). The private key never leaves GitHub.

`gh secret set` documents: “Secret values are locally encrypted before being sent to GitHub.”

## 5. Naming, limits, precedence

From the [Secrets reference](https://docs.github.com/en/actions/reference/security/secrets):

- Names: alphanumeric and underscore only; no spaces; must not start with a number; must not start with `GITHUB_`.
- Names are case-insensitive when referenced. **GitHub stores secret names as uppercase** regardless of how they are entered.
- Uniqueness is per level (repository, organization, or environment).
- Limits: 1,000 organization secrets, **100 repository secrets**, **100 environment secrets**, **48 KB per secret**.
- A workflow can use all 100 repository secrets, all 100 environment secrets, and at most 100 organization secrets (first 100 alphabetically if more are granted).
- Precedence when the same name exists at multiple levels: environment > repository > organization.

Infisical’s GitHub sync uppercases keys before `PUT` and refuses maps larger than 100 (repo/environment) or 1,000 (org) ([`github-sync-fns.ts`](https://github.com/Infisical/infisical/blob/main/backend/src/services/secret-sync/github/github-sync-fns.ts)).

## 6. Authentication and required permissions

### Who can manage secrets in the product UI

[Using secrets](https://docs.github.com/en/actions/how-tos/write-workflows/choose-what-workflows-do/use-secrets):

- Repository secrets: `write` access on an organization repository; collaborator on a personal repository.
- Environment secrets: repository **owner** (personal) or **`admin`** (organization).
- Organization secrets: organization owners.

[Secret types](https://docs.github.com/en/code-security/reference/secret-security/secret-types) states that users with **admin** access create and manage repository/environment Actions secrets, while collaborators can **use** them. That is stricter than the REST endpoint boilerplate (“collaborator access”). A client should treat **admin/owner for environment secrets** and **write/collaborator for repository secrets** as the product rules, and treat REST “collaborator” as the API’s looser wording.

### Personal access token (classic)

- Repository and environment Actions secrets: `repo` (REST). `gh` defaults to `repo` and `read:org`.
- Organization Actions secrets: additional `admin:org`. `gh auth login --scopes "admin:org"` ([Using secrets](https://docs.github.com/en/actions/how-tos/write-workflows/choose-what-workflows-do/use-secrets)).

### Fine-grained PAT

From [permissions for fine-grained PATs](https://docs.github.com/en/rest/authentication/permissions-required-for-fine-grained-personal-access-tokens) and the PAT permission catalog:

| Target | Permission | Access for write |
| --- | --- | --- |
| Repository Actions secrets | Repository **Secrets** | `write` (`read` for list/get/public-key) |
| Environment Actions secrets | Repository **Environments** | `write` |
| Organization Actions secrets | Organization **Secrets** | `write` |
| Dependabot repository secrets | Repository **Dependabot secrets** | `write` |
| Codespaces repository secrets | Repository **Codespaces secrets** | `write` |

A token with only Secrets write cannot manage environment secrets.

### GitHub App

Same permission split as fine-grained PATs: repository **Secrets** for repo Actions secrets, **Environments** for environment secrets, organization **Secrets** for org secrets. Tokens: user-to-server (UAT) or installation (IAT).

Infisical’s documented GitHub App permissions for its connection: Metadata read-only, Secrets read and write, Environments read and write, Actions read; organization Secrets read and write ([GitHub connection](https://infisical.com/docs/integrations/app-connections/github)).

### `gh` CLI

`gh secret set|list|delete` covers repository (default), `--env`, `--org`, and `--user` (Codespaces). `--app` selects Actions / Agents / Codespaces / Dependabot. List JSON fields: `name`, `updatedAt`, `visibility`, `numSelectedRepos`, `selectedReposURL` — **no values**. `createdAt` is not in the documented JSON field list even though REST returns `created_at`.

`gh` is a convenience wrapper over the same REST encrypt-then-PUT/DELETE contract, not a way to read values back.

### Rate limits

Actions secret endpoints are ordinary REST `core` traffic ([Rate limits](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api)):

- Authenticated user / PAT / user-to-server GitHub App: **5,000 requests/hour** (15,000 if a GitHub Enterprise Cloud org owns/approves the app).
- GitHub App installation: **5,000/hour** minimum, scaling with repos/users up to 12,500; 15,000 on GHEC.
- Secondary limits: ≤100 concurrent requests; REST `PUT`/`DELETE` cost **5 points** each vs 1 for most `GET`; ≤900 REST points/minute.

A sync that lists secrets, fetches one public key, then `PUT`s N secrets and `DELETE`s extras is one GET-list (possibly paged) + one GET-key + 5N write points. Infisical’s GitHub sync file includes `// TODO: rate limit handling`.

There is no documented secret-specific rate limit separate from `core`.

## 7. GraphQL

GitHub’s GraphQL [Actions](https://docs.github.com/en/graphql/reference/actions) and [Repos](https://docs.github.com/en/graphql/reference/repos) references expose workflows and workflow runs, not Actions/environment secret CRUD. A search of the Repos GraphQL reference for `secret` returns no fields. Secret management is REST-only (plus `gh` as a REST client).

## 8. Infisical GitHub sync: prefixing and prune

Infisical Secret Syncs are one-way from Infisical to a destination. Docs state Infisical is the source of truth for the connected service: secrets not present (or imported) in Infisical will be overwritten, and changes made directly at the destination may be overwritten ([Secret Syncs overview](https://infisical.com/docs/integrations/secret-syncs/overview)). GitHub cannot import, so the only initial-sync behavior is `overwrite-destination` ([Create GitHub Sync](https://infisical.com/docs/api-reference/endpoints/secret-syncs/github/create); UI: “GitHub does not support importing secrets”).

Destination scopes: `organization`, `repository`, `repository-environment` — matching GitHub’s three Actions secret levels.

Auth methods Infisical supports: GitHub App, OAuth App, PAT ([GitHub connection](https://infisical.com/docs/integrations/app-connections/github)). Implementation uses the same REST paths documented above, Libsodium `crypto_box_seal`, API version `2022-11-28` ([`github-sync-fns.ts`](https://github.com/Infisical/infisical/blob/main/backend/src/services/secret-sync/github/github-sync-fns.ts)).

### Key schema (prefix / template)

[Overview](https://infisical.com/docs/integrations/secret-syncs/overview):

- Key schemas apply a prefix, suffix, or format via Handlebars: `{{secretKey}}`, `{{environment}}`.
- Example: Infisical `SECRET_1` + schema `INFISICAL_{{secretKey}}` → destination `INFISICAL_SECRET_1`.
- **“Any destination secrets which do not match the schema will not get deleted or updated by Infisical.”**
- GitHub UI: “We highly recommend using a Key Schema to ensure that Infisical only manages the specific keys you intend, keeping everything else untouched.”

Implementation ([`secret-sync-fns.ts`](https://github.com/Infisical/infisical/blob/main/backend/src/services/secret-sync/secret-sync-fns.ts)):

1. Before sync, `addSchema` rewrites each Infisical key through the template.
2. GitHub sync then uppercases those names.
3. Every resulting name is `PUT` (always overwrite; no skip-if-unchanged — GitHub cannot confirm values).
4. On prune, `matchesSchema` tests destination names against the compiled prefix/suffix. **If `keySchema` is unset, `matchesSchema` returns `true` for every name.**

### Disable secret deletion (opt out of prune)

UI: “If enabled, Infisical will not remove secrets from the sync destination. Enable this option if you intend to manage some secrets manually outside of Infisical.”

API: `syncOptions.disableSecretDeletion` boolean — “Enable this flag to prevent removal of secrets from the GitHub destination when syncing.”

Implementation: after all `PUT`s, if `disableSecretDeletion` is set, `syncSecrets` returns. Otherwise it lists destination secrets and `DELETE`s any name that (a) matches the key schema and (b) is **not** in the current Infisical map.

That is name-based prune, not value-based. Infisical never sees destination values.

Without a key schema, prune deletes **every** destination secret whose name is not in the current Infisical map — including secrets the user created in GitHub by hand. That is why Infisical warns to use a schema and/or disable deletion.

### Overwrite destination vs disable deletion

These are two different knobs:

- `initialSyncBehavior: overwrite-destination` is the only GitHub option (no import). The UI copy says it “Removes any secrets at the destination endpoint not present in Infisical.”
- `disableSecretDeletion` is the ongoing opt-out of that removal.

If deletion is disabled, Infisical still `PUT`s/overwrites names it manages; it just does not delete extras.

### Removing secrets when deleting the sync

`DELETE /api/v1/secret-syncs/github/{syncId}?removeSecrets=true|false` (default `false`): “Whether previously synced secrets should be removed prior to deletion.” GitHub is flagged `canRemoveSecretsOnDeletion: true`.

Implementation `removeSecrets` lists destination secrets and deletes names that **are** in the current (schema-applied) Infisical map — cleanup of keys Infisical would have written, not a second prune of unrelated extras.

## 9. Hard limitations that constrain a first-release GitHub Sync Target contract

These are GitHub (and Infisical-pattern) facts, not product decisions:

1. **GitHub cannot be a source of truth for values.** List/get never return plaintext. A Sync Target is write-only for values. Drift detection of values requires a client-side fingerprint of last successful write, or an always-overwrite policy.

2. **Create and update are the same `PUT`.** There is no “set only if missing” or “set only if changed” API.

3. **Prune can only be name-based.** A client can list names (and timestamps) and delete names not in the selected local set. It cannot confirm the remote value before delete.

4. **Prefix/schema is the only first-party-safe way to avoid deleting unowned GitHub secrets.** GitHub has no ownership metadata on a secret. Infisical’s schema match is a client-side convention: extras that do not match the prefix/template are left alone. Without a prefix, optional prune means “delete every Actions secret in that repo/environment that is not in this sync set.”

5. **`disableSecretDeletion` / opt-in prune is required if users also keep manual GitHub secrets.** Infisical documents this explicitly.

6. **Always-overwrite is the only value-sync strategy GitHub supports without a local hash.** Infisical always `PUT`s every selected secret.

7. **Repository Actions secrets, environment Actions secrets, Dependabot, and Codespaces are different APIs and stores.** First-release “GitHub repository secrets” does not imply the others. Environment secrets need Environments write plus a pre-existing environment and stricter human permissions.

8. **Organization Actions secrets are a third scope** with visibility policies, `admin:org` / org Secrets write, and a 1,000-secret cap. GitHub Free private repos cannot use org secrets.

9. **Names are coerced to uppercase and must match GitHub’s charset.** Local keys with hyphens, spaces, or a leading digit cannot be sent as-is. Prefixes must themselves be valid GitHub secret names (no `GITHUB_` prefix).

10. **Hard caps: 100 repo secrets, 100 environment secrets, 48 KB each.** A selected local set larger than that cannot fully sync.

11. **GraphQL cannot replace REST** for this contract.

12. **Auth must cover the chosen scope.** Secrets write is not enough for environment secrets. `gh`/`PAT`/`GitHub App` are all viable; none of them can read values back.

13. **Rate limits are the shared REST `core` budget.** A large prune+write loop is many 5-point `PUT`/`DELETE`s.

14. **Libsodium sealed-box encryption is mandatory** for REST writes. Sending plaintext in `encrypted_value` is not a documented option.

15. **Public-key GET is world-readable on public repos.** Encryption protects values in transit to GitHub; it does not hide that a client is about to write a secret, and it does not authenticate the writer (the PAT/App token does).
