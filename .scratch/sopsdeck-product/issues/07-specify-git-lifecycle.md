# Specify Git review, Sync, Secret History, and conflicts

Type: grilling
Status: resolved
Blocked by: None

## Question

What exact state machine governs semantic review, encrypted diff, save, stage, commit, fetch/rebase, push, on-demand historical decryption, restore, dirty worktrees, and decryptable three-way conflicts without surprising or destructive Git behavior? What should the Git “sync” action be named so it is not confused with a Sync Target?

## Answer

Git uses system `git`. Save writes ciphertext atomically in the worktree. **Review** shows an on-demand plaintext semantic diff of uncommitted Managed Files (plus the encrypted diff if useful). **Commit** stages those paths and commits with the user’s message.

The Git action is named **Sync**. A Sync Target publish is named **Publish** (e.g. Publish to GitHub). Sync = `fetch` then `pull --ff-only` then `push`. If the branch has diverged or the Managed Files worktree is dirty, stop and say so. Never force-push, never auto-rebase, never auto-merge.

**Secret History** is `git log` on the file. Decrypt a revision only when opened; do not keep a plaintext audit store. **Restore** copies those values into the current editor and requires a new save/commit.

Conflicts: if base, ours, and theirs all decrypt and parse, show a secret-aware three-way for those keys. Otherwise leave it to Git and refuse to invent a resolution.

## Implementation (2026-08-28)

`sopsdeck commit -m … -f` and `sopsdeck sync` (fetch, `pull --ff-only`, push) are tested, including diverge refusal and dirty-worktree stop. `sopsdeck review -f` prints a plaintext semantic diff of uncommitted keys vs HEAD. Desktop Commit/Sync invoke those commands. Commit message is user-supplied; prefill is [21](21-desktop-chrome-polish.md). Not done: Secret History, Restore, three-way.

