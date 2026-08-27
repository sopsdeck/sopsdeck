# Specify MCP and AI skill security contracts

Type: grilling
Status: resolved
Blocked by: None

## Question

Which local MCP tools and Codex/Claude skill workflows exist; what metadata, plaintext, mutation, command-injection, approval, logging, and capability boundaries let agents help without turning the model into a secret exfiltration path?

## Answer

Local MCP only, plus thin Codex/Claude skills that call it. Default tools return metadata: Projects, Managed Files, key names, Recipients, Git state. Reading a value or mutating a Managed File requires per-call approval. Prefer `run` (inject into a child command) over returning plaintext to the model. Log tool name and outcome, never values. No remote MCP, no cloud processing, no skills that dump files into the prompt.
