# Agent and IDE protection

Sopsdeck can keep an application's Managed File values out of an agent's environment and captured tool output. Run the application as a child of `sopsdeck run`:

```sh
sopsdeck run -f .env -- npm run dev
```

The application child receives the values it needs. Sopsdeck redacts captured stdout and stderr before they reach the terminal or task output. This does not protect direct file access, the child application's own file or network writes, or values derived from the secrets.

Do not wrap the whole coding agent with `sopsdeck run`. That gives the agent process the application's secret environment; terminal masking alone does not remove values from internal tool results, transcripts, or telemetry.

## Examples

The opt-in examples are in [`examples/agent-protection`](../examples/agent-protection):

- `.vscode/tasks.json` starts an app using a process task, with arguments kept separate from shell quoting.
- `.claude/settings.json` blocks Claude Code's `Read` tool for paths listed in `.sopsdeck.toml` and denies the example dotenv paths to sandboxed shell commands.
- `.cursor/hooks.json` blocks Cursor Agent and Tab reads of Managed Files. Cursor's `beforeReadFile` hook receives the complete file contents; the Sopsdeck adapter discards those fields and never logs the payload.
- `.codex/config.toml` shows a Codex filesystem profile that denies the example dotenv paths to sandboxed tool calls. Add every Managed File path to the profile.

Copy or merge the relevant entries into your Project. Replace `.env` and `.env.*` with the paths of all Managed Files, including structured JSON and YAML files. A path denial covers the whole file. It does not let the agent read only public leaves of a structured Managed File.

## Host behavior and limits

| Host               | Native control and event                                                                                                                     | Where the control applies                                                                                                                                            | Current limit                                                                                                                                                                                                         |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Codex CLI          | Trusted project `default_permissions` filesystem profile                                                                                     | Sandboxed local tool calls, including file reads and shell processes                                                                                                 | Configure exact Managed File paths. Project config is skipped until the project is trusted. Start the app task outside the Codex sandbox because the profile also blocks the runner from reading the file.            |
| Claude Code        | `PreToolUse` matcher `Read`; input uses `tool_name` and `tool_input.file_path`; denial uses `hookSpecificOutput.permissionDecision = "deny"` | The Read hook runs before the file tool can return content. `sandbox.filesystem.denyRead` separately constrains sandboxed shell processes on macOS, Linux, and WSL2. | Shell commands that cannot run in the sandbox may prompt for an unsandboxed retry; decline it. Native Windows shell commands are unsandboxed by Claude Code, so use WSL2 or keep shell commands under human approval. |
| Cursor Agent / CLI | `beforeReadFile` and `beforeTabFileRead`; input has `file_path` (and `content` for Agent reads); output uses `permission`                    | Before Agent or Tab file content is sent to its consumer                                                                                                             | The hook does not deny shell reads from files inside the workspace. Keep normal command approvals enabled and reject direct read/export requests. The hook cannot make that human decision for you.                   |
| VS Code task       | `type: "process"`, `command`, and argument array                                                                                             | The protected app child and its captured terminal output                                                                                                             | Tasks are launchers, not access-control policy. The task does not prevent an agent from asking to read or export a file through another tool.                                                                         |

The hook adapter matches paths against the nearest `.sopsdeck.toml` `managed_file` entries, resolves symlink aliases, and emits only a fixed denial reason. It never parses shell text, grants permissions, or records hook input. If a hook payload is malformed or the manifest cannot be checked, it denies the read. Cursor hooks should use `failClosed: true` so a missing or timed-out adapter also blocks the read.

Host controls still depend on user trust and approval settings. Keep the host's ordinary approval prompts. Do not pre-approve commands that print dotenv values, use `get` to export them, run `printenv`, or source a Managed File. When direct access is needed, stop and ask the project owner to perform the operation. A human-approved shell command can disclose secrets by design.

## Install and remove

1. Install Sopsdeck so `sopsdeck` is on the host hook process's `PATH`. Run `sopsdeck --version` from the IDE's integrated terminal first. If the host process does not inherit that `PATH`, use the absolute Sopsdeck executable path in the hook command.
2. Copy the matching example configuration into your Project, merging existing settings instead of replacing them. Add every Managed File path to the Codex profile or Claude shell deny list.
3. Trust the Project when Codex or Claude Code asks. In Codex, inspect project hooks/settings with `/hooks`; in Claude Code, review the project settings trust prompt. Cursor reloads project hooks from `.cursor/hooks.json`.
4. Run the “Start app with Managed File” task from VS Code's **Tasks: Run Task** menu. It launches `sopsdeck run -f .env -- npm run dev`; edit the app arguments to match your Project.
5. To uninstall, remove only the Sopsdeck hook entries and filesystem denial entries. For Codex, restore the previous `default_permissions` value if the Project already had one. Preserve unrelated host settings and hooks.

## Verification status

Offline protocol fixtures cover Claude and Cursor allow/deny output, malformed payloads, a path containing spaces, symlink aliases, and ensuring fake input content never appears in adapter output. The installed versions inventoried on macOS arm64 on 2026-10-08 were Codex CLI 0.154.0, Claude Code 2.1.259, Cursor 3.23.23, and VS Code 1.137.0. The host integrations have not yet been exercised through live agent sessions. Do not interpret the offline fixtures as proof that a particular host build honors its configuration. Current official docs: [Codex hooks and permissions](https://developers.openai.com/codex/hooks), [Codex configuration](https://developers.openai.com/codex/config-reference), [Claude Code hooks](https://code.claude.com/docs/en/hooks), [Claude Code sandbox](https://code.claude.com/docs/en/sandboxing), [Cursor hooks](https://cursor.com/docs/hooks), [Cursor run modes](https://cursor.com/docs/agent/security/run-modes), and [VS Code tasks](https://code.visualstudio.com/docs/debugtest/tasks).
