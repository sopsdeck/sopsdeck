# Sopsdeck brand spec

Canonical domain: **sopsdeck.com**

Positioning: For developer teams, Sopsdeck is the local-first secrets workspace that makes SOPS-encrypted environments understandable and safe to manage—without moving trust away from the developer’s machine.

Attributes: precise, calm, developer-native.

Product UI principle: **Your filesystem is the workspace.** Projects appear as local folders in one persistent sidebar, secret-bearing configuration files appear as nested leaves, and selecting a file opens one focused editor with access, encryption, and Git state in context.

Format principle: dotenv is one adapter, not the product model. The initial interface should accommodate SOPS-friendly dotenv, JSON (including Expo `eas.json`), and YAML files while preserving each format’s native structure.

Recommended direction: **Quiet Cipher**. Structured geometry and a bright encrypted seam communicate controlled movement from local edits to encrypted Git changes, without relying on padlock or shield clichés.

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

1. **Cipher seam — recommended:** two file planes meet at a controlled seam, representing the move from local editing to encrypted output.
2. **Terminal cut:** `sops//deck` uses a familiar code marker as the boundary between plaintext work and encrypted storage.
3. **Open S:** a broken letterform suggests visibility without exposure and gives the wordmark a compact signature.
4. **Small-size proof:** the cipher seam mark remains recognizable in one color, inverted, and at 24px.

## Don'ts

1. Don’t use generic padlocks, shields, keys, or hacker-green imagery.
2. Don’t use Drift Signal coral as decoration; reserve it for states requiring attention.
3. Don’t set long prose in IBM Plex Mono; it is for technical labels and data only.
