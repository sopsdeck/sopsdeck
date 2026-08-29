use std::path::{Path, PathBuf};
use std::process::Command;

pub fn sopsdeck_bin() -> PathBuf {
    if let Ok(path) = std::env::var("SOPSDECK_BIN") {
        return PathBuf::from(path);
    }
    let repo = PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("../..");
    let candidate = repo.join("sopsdeck");
    if candidate.exists() {
        return candidate;
    }
    PathBuf::from("sopsdeck")
}

pub fn run_sopsdeck(args: &[&str]) -> Result<String, String> {
    run_sopsdeck_in(Path::new("."), args)
}

pub fn run_sopsdeck_in(dir: &Path, args: &[&str]) -> Result<String, String> {
    let mut cmd = Command::new(sopsdeck_bin());
    cmd.args(args).current_dir(dir).envs(default_env());
    let out = cmd.output().map_err(|error| error.to_string())?;
    if !out.status.success() {
        return Err(String::from_utf8_lossy(&out.stderr).trim().to_string());
    }
    Ok(String::from_utf8_lossy(&out.stdout).to_string())
}

pub fn git_root(path: &str) -> Result<PathBuf, String> {
    let dir = Path::new(path)
        .parent()
        .filter(|parent| !parent.as_os_str().is_empty())
        .unwrap_or_else(|| Path::new("."));
    let out = Command::new("git")
        .args(["rev-parse", "--show-toplevel"])
        .current_dir(dir)
        .output()
        .map_err(|error| error.to_string())?;
    if !out.status.success() {
        return Err(String::from_utf8_lossy(&out.stderr).trim().to_string());
    }
    Ok(PathBuf::from(String::from_utf8_lossy(&out.stdout).trim()))
}

// default_env returns env overrides the sidecar adds on top of the inherited
// parent env, so the desktop app is self-sufficient when launched from a GUI
// (no shell env): SOPSDECK_STATE_DIR logs failed commands, and SOPS_AGE_KEY_CMD
// lets a keychain identity decrypt Managed Files.
fn default_env() -> Vec<(&'static str, String)> {
    let mut env: Vec<(&'static str, String)> = Vec::new();
    if let Some(dir) = default_state_dir() {
        env.push(("SOPSDECK_STATE_DIR", dir));
    }
    let age_key_file = std::env::var("SOPS_AGE_KEY_FILE").unwrap_or_default();
    let age_key_cmd = std::env::var("SOPS_AGE_KEY_CMD").unwrap_or_default();
    if age_key_file.is_empty() && age_key_cmd.is_empty() {
        let bin = sopsdeck_bin().to_string_lossy().into_owned();
        env.push(("SOPS_AGE_KEY_CMD", format!("{bin} identity key")));
    }
    env
}

fn default_state_dir() -> Option<String> {
    if let Ok(dir) = std::env::var("SOPSDECK_STATE_DIR") {
        if !dir.is_empty() {
            return Some(dir);
        }
    }
    let home = std::env::var("HOME")
        .or_else(|_| std::env::var("USERPROFILE"))
        .unwrap_or_default();
    if home.is_empty() {
        return None;
    }
    Some(
        PathBuf::from(home)
            .join(".config")
            .join("sopsdeck")
            .to_string_lossy()
            .into_owned(),
    )
}
