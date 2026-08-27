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
    let out = Command::new(sopsdeck_bin())
        .args(args)
        .current_dir(dir)
        .output()
        .map_err(|error| error.to_string())?;
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
