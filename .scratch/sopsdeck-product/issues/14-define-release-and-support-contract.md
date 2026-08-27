# Define packaging, updates, and platform support

Type: grilling
Status: resolved
Blocked by: None

## Question

What are the CLI binary name and aliases, signing, notarization, installer, auto-update, CLI installation, versioning, license, compatibility, support, and later Windows/Linux promises for a product with no required hosted service?

## Answer

Binary is `sopsdeck` with optional `sd` alias. No `dotenvx` shim. License Apache-2.0. Versioning is semver. macOS ships first: Developer ID, notarization, Tauri signed updater via static JSON on GitHub Releases (no Sopsdeck backend). The CLI ships inside the app; Homebrew can come later. Windows/Linux use the same Go core and Tauri shell later, with no date promise. No paid support contract in v1.

## Implementation (2026-08-28)

Semver is unchanged as the wire format. Interpretation and changelog surfaces: [22 Epoch SemVer, CHANGELOG.md, and release notes](22-epoch-semver-and-changelog.md) — epoch overlay (`EPOCH * 1000 + MAJOR`), `CHANGELOG.md` canonical, GitHub Release body generated from that file, landing page and in-app What’s new from the same source. Signed artifacts and updater JSON remain this ticket’s phase 8 work.

