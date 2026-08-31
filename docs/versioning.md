# Versioning

Sopsdeck versions are [Epoch SemVer](https://antfu.me/posts/epoch-semver): still `MAJOR.MINOR.PATCH` on GitHub and npm. Read the first number as `EPOCH * 1000 + MAJOR`.

| Part      | Meaning                                                                                     |
| --------- | ------------------------------------------------------------------------------------------- |
| **EPOCH** | A named product era. Rare. `1000.0.0` is a marketing/overhaul event, not a broken CLI flag. |
| **MAJOR** | Incompatible behavior users must notice (0–999 inside an epoch).                            |
| **MINOR** | Compatible features.                                                                        |
| **PATCH** | Compatible fixes.                                                                           |

Until a named era, EPOCH stays **0**, so versions look like ordinary `1.2.3`.

The unreleased tree may stay on **`0.1.0`**. The **first public GitHub Release** is **`v1.0.0`** (epoch 0) when the public npm/browser package ships (phase 8). Do not stay on a leading-zero version forever to hide breaks.

`CHANGELOG.md` is the source of truth (Keep a Changelog: Unreleased plus version sections). GitHub Releases copy that section; the landing page and in-app What’s new are generated from the same file. Do not dump git subjects onto the site.

CLI `sopsdeck --version` and the npm launcher share one version string (`internal/version` and root `package.json`). `./scripts/check` fails if they drift. Tag `vX.Y.Z` CI fails if `CHANGELOG.md` has no matching section.
