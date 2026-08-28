# Public site: download, Cloudflare, hero video

Type: build
Status: open
Blocked by: [14](14-define-release-and-support-contract.md) (signed artifacts for a real download), [29](29-docs-site.md)

## What to build

The public site should let people download the app, be deployed on Cloudflare, and show a teaser video in the hero. Canonical domain stays sopsdeck.com ([29](29-docs-site.md)). Signing/notarization stays [14](14-define-release-and-support-contract.md); do not ship an unsigned “download” that pretends to be the release.

## Acceptance criteria

- [ ] Landing has a download path that points at GitHub Releases (or a documented placeholder until 14 lands).
- [ ] Site deploy is Cloudflare (Pages or equivalent) with a documented command/workflow.
- [ ] Hero includes a teaser video from the existing demo catalog ([19](19-drive-professional-product-assets.md) / [28](28-usable-product-recordings.md)), not a new product recording pipeline.

## Seams

- `site/`, release workflow, Cloudflare config. No new product policy.

## Comments

Captured 2026-08-28 from human-found review. Kind: idea.
