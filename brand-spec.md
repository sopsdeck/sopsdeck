# Sopsdeck brand spec

Canonical domain: **sopsdeck.com**

Positioning: Sopsdeck helps individual developers and teams browse and edit SOPS-encrypted secrets.

Headline: **A visual browser and editor for your secrets.**

Subheadline: A simple, flexible way to manage SOPS-encrypted dotenv, JSON, and YAML files. Edit locally, share access with your team, and track changes in Git.

Audience: **For individual developers and teams.**

Attributes: precise, calm, developer-native.

Product UI principle: **Your filesystem is the workspace.** Projects appear as local folders in one persistent sidebar, secret-bearing configuration files appear as nested leaves, and selecting a file opens one focused editor with access, encryption, and Git state in context.

Format principle: dotenv is one adapter, not the product model. The initial interface should accommodate SOPS-friendly dotenv, JSON (including Expo `eas.json`), and YAML files while preserving each format’s native structure.

Recommended direction: **Structural monogram**. Two opposing ledges form a solid S with the weight and stability of a deck, giving the name a compact mark without relying on padlock or shield clichés.

Wordmark: keep the monogram one color; render `sops` in the surrounding text color and `deck` in Commit Blue on light backgrounds or Decrypt Mint on Vault Ink and other dark backgrounds.

## Color

| Color        | HEX       | Job                                   |
| ------------ | --------- | ------------------------------------- |
| Vault Ink    | `#101828` | Core identity, dark UI, primary text  |
| Commit Blue  | `#3157F6` | Brand recognition and primary actions |
| Decrypt Mint | `#46D6A8` | Safe, synced, and successful states   |
| Local Paper  | `#F7F9FC` | Main light canvas and inverse text    |
| Diff Slate   | `#475467` | Body text and secondary information   |
| Drift Signal | `#FF6B57` | Warnings and changed-state emphasis   |

Verified contrast: Ink/Paper 16.83:1, Slate/Paper 7.29:1, Paper/Blue 5.21:1, and Ink/Mint 9.67:1. These pairs exceed WCAG AA for body text.

## Typography

- Headline and UI: **Manrope**, with a generic `sans-serif` fallback.
- Technical labels and code: **IBM Plex Mono**, with `ui-monospace, SFMono-Regular, Consolas, monospace` fallbacks.
- Type scale: 12, 15, 19, 24, 30, 38, 48, 60px (1.25 ratio).
- Both families are freely licensed under the SIL Open Font License 1.1.

## Logo concepts

1. **Structural monogram — selected:** two opposing ledges form a solid S, expressing the deck as a stable platform and the name as a compact signature.
2. **Cipher seam:** two file planes meet at a controlled seam, representing the move from local editing to encrypted output.
3. **Terminal cut:** `sops//deck` uses a familiar code marker as the boundary between plaintext work and encrypted storage.
4. **Open S:** a broken letterform suggests visibility without exposure and gives the wordmark a compact signature.

## Assets

Sources and ready-to-upload files live in `brand/`. Preview: [brand/preview.html](brand/preview.html). Regenerate with `bun scripts/brand-export.mjs`.

Square icon (monochrome structural monogram on Vault Ink): use for the GitHub org profile image, macOS/Windows app icon, Apple touch icon, and PWA icons. GitHub crops a circle; the OS crops a squircle — padding is sized for both.

Display-weight structural monogram plus wordmark: use on the site header, lockups, and wide heroes. GitHub organizations have no cover photo; put `brand/export/github-readme-hero.png` at the top of the org profile README, and `github-social-preview.png` as a repository social preview.

| File                                     | Use                                  |
| ---------------------------------------- | ------------------------------------ |
| `brand/export/github-org-avatar.png`     | GitHub org profile image (1024×1024) |
| `brand/export/github-readme-hero.png`    | Org README banner (1280×400)         |
| `brand/export/github-social-preview.png` | GitHub social preview (1280×640)     |
| `brand/export/og-image.png`              | Site Open Graph (1200×630)           |
| `brand/export/social-banner.png`         | Wide header (1500×500)               |
| `brand/export/app-icon.png`              | Desktop icon master (1024×1024)      |
| `brand/export/lockup-dark.png`           | Transparent lockup for ink           |
| `brand/export/lockup-light.png`          | Transparent lockup for paper         |
| `site/favicon.svg`                       | Browser tab icon                     |

## Don'ts

1. Don’t use generic padlocks, shields, keys, or hacker-green imagery.
2. Don’t use Drift Signal coral as decoration; reserve it for states requiring attention.
3. Don’t set long prose in IBM Plex Mono; it is for technical labels and data only.
