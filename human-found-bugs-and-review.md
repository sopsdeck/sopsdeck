# Review

This file is where I will capture any issues, features, bugs, improvements from
testing and using the app. When you review this, you should process the
findings into tickets to start fixing and implementing them, and crossing them
off from this file so it is clear they have been processed and will get
addressed.

## Findings

_Processed into tickets 32–38._

## Feature Ideas

- [ ] feature 1

---

### Archived

- [x] need a better way for running it locally, it should just be one command to
      launch the app, build the cli, make sure the app is using the freshly
      built cli → [32](.scratch/sopsdeck-product/issues/32-one-command-local-launch.md)
- [x] when running and testing it locally and any errors occur, they should be
      captured into a file and properly logged, such that the same error simpley
      just increments the count instead of duplicating the error, then these
      errors can be discovered and fixed like tickets or bugs, and the logging
      and observability system should work effectively to help debug and fix
      issues → [33](.scratch/sopsdeck-product/issues/33-deduped-local-error-log.md)
- [x] The right panel should work kind of like how panels in vscode work, where
      each section is collapsible → [34](.scratch/sopsdeck-product/issues/34-inspector-collapse-and-reveal-heading.md)
- [x] The add secret button can be remove, and the reveal values button can be a
      show/hide icon in the heading for the values column next to the "value"
      text → [34](.scratch/sopsdeck-product/issues/34-inspector-collapse-and-reveal-heading.md)
- [x] The left file tree needs to be more advanced and robust, it should support
      nested folders, i should be able to collapse folders, have recents and
      truncation with a show more button like codex → [35](.scratch/sopsdeck-product/issues/35-nested-project-file-tree.md)
- [x] Changelog still needs to be refined, adding change types with nice tags
      and badges along with grouping by change type (bug fix, feature, performance)
      following conventional commits, grouping by date and version, as well as
      platform information (macos, windows, linux etc) → [36](.scratch/sopsdeck-product/issues/36-changelog-type-tags.md)
- [x] Should be able to download the app from the website → [37](.scratch/sopsdeck-product/issues/37-public-site-download-and-hosting.md)
- [x] Website should be deployed on cloudflare → [37](.scratch/sopsdeck-product/issues/37-public-site-download-and-hosting.md)
- [x] Website should display a teaser video in the hero → [37](.scratch/sopsdeck-product/issues/37-public-site-download-and-hosting.md)
- [x] Demo screenshots and assets aren't seeded properly with realistic data, they
      should have a sidebar filled with projects, some open some closed, real looking
      projects with real looking files and secrets ranging from docker compose
      files to dot env files to eas.json files, and whatever other common frameworks
      use → [38](.scratch/sopsdeck-product/issues/38-multi-project-demo-seed.md)
- [x] Revealing secrets that aren't used anywhere → [31](.scratch/sopsdeck-product/issues/31-deferred-product-ideas.md) (needs a reference rule; adjacent to scan)
- [x] Smart renaming of secrets, if you rename a secret key that is used, it
      should offer to update any references to use the new name → [31](.scratch/sopsdeck-product/issues/31-deferred-product-ideas.md) (in-file rename is 24)
- [x] Smart clipboard detection, if you focus the app and have secret in
      your clipboard it should offer to capture it, same for recipient keys, by
      showing a modal with where you want to add the secret or which
      file/projects you want to give the key access to → [31](.scratch/sopsdeck-product/issues/31-deferred-product-ideas.md) (this is issue 12 + 06, not a side channel)
- [x] OpenBao? → [31](.scratch/sopsdeck-product/issues/31-deferred-product-ideas.md) (later Sync Target; needs specify first)
- [x] an easy way for me to copy an absolute path to a project from my terminal
      or nvim editor or something, the go to the sopsdeck app and just open that
      path from my clipboard or paste it into the file tree or something → [31](.scratch/sopsdeck-product/issues/31-deferred-product-ideas.md) (issue 05; blocked on 23)
- [x] I don't like the dark mode button, it should be an icon or something cleaner → [25](.scratch/sopsdeck-product/issues/25-sidebar-window-and-scroll.md)
- [x] Secret value inputs should have click to reveal, copy icons, delete icons → [24](.scratch/sopsdeck-product/issues/24-editor-key-row-actions.md)
- [x] I can't add any more secret files from the sidebar → [25](.scratch/sopsdeck-product/issues/25-sidebar-window-and-scroll.md)
- [x] I can't delete a secret → [24](.scratch/sopsdeck-product/issues/24-editor-key-row-actions.md)
- [x] The UI for adding a secret and hiding/showing values is lazy, should be
      a clean icon for revealing/hiding and for adding it should be a clean hover
      effect at the bottom, or maybe a text input like a chat or something where
      you can just paste in a secret key=value line, or just type in a key or
      something, and currently when you add a secret you can't see the key input
      anyway which is a clear UI bug, but either way i think this flow should be
      replaced → [24](.scratch/sopsdeck-product/issues/24-editor-key-row-actions.md)
- [x] I should be able to copy a key via revealed copy icon on hover, and i should
      be able to simple just edit the key by clicking on it → [24](.scratch/sopsdeck-product/issues/24-editor-key-row-actions.md)
- [x] There should be markdown linting and formatting → [30](.scratch/sopsdeck-product/issues/30-deterministic-quality-gates.md)
- [x] All of the assets are 0.01s long, basically unusable recordings, should
      probably use something like [webreel](https://webreel.dev/) → [28](.scratch/sopsdeck-product/issues/28-usable-product-recordings.md)
- [x] Poor scroll handling, ugly scrollbars, i can scroll down to see the body
      which is not a good look → [25](.scratch/sopsdeck-product/issues/25-sidebar-window-and-scroll.md)
- [x] App just appears as "desktop" and doesn't have the logo → [25](.scratch/sopsdeck-product/issues/25-sidebar-window-and-scroll.md)
- [x] UI components, while look good and the branding is good, don't feel very
      polished, perhaps something like shadcn/tailwind should be used. The
      just seems to be a general lack of polish, lack of icons, lack of
      tasteful animations, hover effects, etc. → [26](.scratch/sopsdeck-product/issues/26-visual-polish-and-changelog-look.md) (kit is optional, not mandated)
- [x] I want to seem more realistic examples in the test data, like an eas.json,
      a docker-compose file, Dockerfile, and other config files you would see
      in the wild, as well as some examples of realistic multiline secrets. → [27](.scratch/sopsdeck-product/issues/27-realistic-managed-file-fixtures.md)
- [x] Changelog though working, is boring, and ugly. → [26](.scratch/sopsdeck-product/issues/26-visual-polish-and-changelog-look.md)
- [x] Need to add some security tools, something like sonarqube for example,
      some form of lint/code security checks that are deterministic, easy to run → [30](.scratch/sopsdeck-product/issues/30-deterministic-quality-gates.md)
- [x] Need to add more deterministic checks for things we need to keep remember
      to do, for example keeping the changelog up to date with clean, human readable
      and use friendly messages, one thing i can think of is a commit hook or
      something that enforces updating the change logs and any other
      documentation, as well as enforcing passing tests, linting and formatting
      checks → [30](.scratch/sopsdeck-product/issues/30-deterministic-quality-gates.md)
- [x] need demo videos of the cli tool as well, using something like [castkit](https://github.com/deeflect/castkit) → [28](.scratch/sopsdeck-product/issues/28-usable-product-recordings.md)
- [x] need a docs page as will, I'm honestly kind of considering just making the
      landing page a docusaurus site that has the landing page, changelog, and
      documentation all in one → [29](.scratch/sopsdeck-product/issues/29-docs-site.md) (Docusaurus is a candidate)
- [x] app beach balls if i try opening a folder → [23](.scratch/sopsdeck-product/issues/23-fix-folder-open-hang.md) (P0)
