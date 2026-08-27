# Specify coexistence with existing SOPS projects

Type: grilling
Status: resolved
Blocked by: None

## Question

How does Sopsdeck relate to `.sops.yaml` creation rules and already-SOPS-encrypted trees: coexist, generate, wrap, replace, or import? What happens when a user adds a folder that already has SOPS metadata, mixed encrypted/unencrypted files, or path-based creation rules Sopsdeck did not write?

## Answer

Coexist. `.sops.yaml` stays the SOPS encryption policy (creation rules, recipients, path regexes). Sopsdeck does not wrap, replace, or fork it. Adding or removing Access updates `.sops.yaml` (when Sopsdeck is the one changing Recipients) and re-encrypts the affected files the SOPS way.

Add a folder that already has SOPS files: those files are Managed Files; existing `.sops.yaml` is honored as-is; Sopsdeck does not rewrite rules it did not change. Mixed trees: only files with SOPS metadata are Managed by default; unencrypted candidates can be encrypted on explicit action, never as a silent scan-and-encrypt. If there is no `.sops.yaml` yet, Sopsdeck writes a minimal one when it first encrypts or first grants Access.
