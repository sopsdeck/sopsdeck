# Specify Expo eas.json handling

Type: grilling
Status: resolved
Blocked by: None

## Question

Given that SOPS can wrap `eas.json` as JSON but an encrypted file is not valid EAS CLI input, and Expo’s own secret story is EAS environment variables plus `credentials.json` (with `eas.json` `env` meant for committable values): should Sopsdeck treat `eas.json` as a Managed File, leave it plaintext and map secrets to an EAS Sync Target, decrypt a gitignored working copy for local EAS, or drop first-release `eas.json` encryption entirely?

## Answer

`eas.json` is JSON. It can be a Managed File. If it is encrypted, Sopsdeck warns that EAS CLI will not read SOPS ciphertext. No gitignored decrypted working copy (threat model). Prefer secrets in dotenv Managed Files and `sopsdeck run`. `eas.json` `env` stays for committable values. An EAS API Sync Target is later, not v1.
