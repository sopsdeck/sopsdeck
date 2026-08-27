use std::path::{Path, PathBuf};
use std::process::Command;

use serde::Serialize;
use tauri_plugin_dialog::DialogExt;

#[derive(Serialize)]
struct ManagedFile {
    name: String,
    path: String,
    rel: String,
}

#[derive(Serialize)]
struct Pair {
    key: String,
    value: String,
}

fn sopsdeck_bin() -> PathBuf {
    if let Ok(p) = std::env::var("SOPSDECK_BIN") {
        return PathBuf::from(p);
    }
    let repo = PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("../..");
    let candidate = repo.join("sopsdeck");
    if candidate.exists() {
        return candidate;
    }
    PathBuf::from("sopsdeck")
}

fn run_sopsdeck(args: &[&str]) -> Result<String, String> {
    let out = Command::new(sopsdeck_bin())
        .args(args)
        .output()
        .map_err(|e| e.to_string())?;
    if !out.status.success() {
        return Err(String::from_utf8_lossy(&out.stderr).trim().to_string());
    }
    Ok(String::from_utf8_lossy(&out.stdout).to_string())
}

fn is_dotenv_name(name: &str) -> bool {
    name == ".env" || name.starts_with(".env.") || name.to_ascii_lowercase().ends_with(".env")
}

fn is_structured_name(name: &str) -> bool {
    let lower = name.to_ascii_lowercase();
    lower.ends_with(".json") || lower.ends_with(".yaml") || lower.ends_with(".yml")
}

fn skip_dir(name: &str) -> bool {
    matches!(
        name,
        ".git" | "node_modules" | "target" | "dist" | "vendor" | ".scratch"
    )
}

fn looks_sops(path: &Path) -> bool {
    let Ok(data) = std::fs::read(path) else {
        return false;
    };
    let n = data.len().min(16_384);
    let sample = String::from_utf8_lossy(&data[..n]);
    sample.contains("\"sops\"") || sample.contains("sops:")
}

fn walk_managed(root: &Path, prefix: &Path, out: &mut Vec<ManagedFile>) {
    let entries = match std::fs::read_dir(root) {
        Ok(e) => e,
        Err(_) => return,
    };
    for entry in entries.flatten() {
        let path = entry.path();
        let name = entry.file_name().to_string_lossy().into_owned();
        if path.is_dir() {
            if skip_dir(&name) {
                continue;
            }
            walk_managed(&path, prefix, out);
            continue;
        }
        let take = is_dotenv_name(&name) || (is_structured_name(&name) && looks_sops(&path));
        if !take {
            continue;
        }
        let rel = path
            .strip_prefix(prefix)
            .map(|p| p.to_string_lossy().into_owned())
            .unwrap_or_else(|_| path.to_string_lossy().into_owned());
        out.push(ManagedFile {
            name,
            path: path.to_string_lossy().into_owned(),
            rel,
        });
    }
}

#[tauri::command]
fn list_managed_files(path: String) -> Result<Vec<ManagedFile>, String> {
    let root = PathBuf::from(&path);
    if !root.is_dir() {
        return Err("not a folder".into());
    }
    let mut files = Vec::new();
    walk_managed(&root, &root, &mut files);
    files.sort_by(|a, b| a.rel.cmp(&b.rel));
    Ok(files)
}

#[tauri::command]
fn get_managed_file(path: String) -> Result<Vec<Pair>, String> {
    let out = run_sopsdeck(&["get", "-f", &path, "--output", "json"])?;
    let map: serde_json::Map<String, serde_json::Value> =
        serde_json::from_str(out.trim()).map_err(|e| e.to_string())?;
    let mut pairs: Vec<Pair> = map
        .into_iter()
        .map(|(key, v)| Pair {
            value: match v {
                serde_json::Value::String(s) => s,
                other => other.to_string(),
            },
            key,
        })
        .collect();
    pairs.sort_by(|a, b| a.key.cmp(&b.key));
    Ok(pairs)
}

#[tauri::command]
fn set_managed_key(path: String, key: String, value: String) -> Result<(), String> {
    run_sopsdeck(&["set", &key, &value, "-f", &path]).map(|_| ())
}

#[tauri::command]
async fn pick_project_folder(app: tauri::AppHandle) -> Result<Option<String>, String> {
    let folder = app
        .dialog()
        .file()
        .set_title("Add project folder")
        .blocking_pick_folder();
    Ok(folder.and_then(|p| p.into_path().ok()).map(|p| p.to_string_lossy().into_owned()))
}

#[tauri::command]
fn boot_project() -> Option<String> {
    std::env::var("SOPSDECK_DEV_PROJECT").ok().filter(|s| !s.is_empty())
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_dialog::init())
        .invoke_handler(tauri::generate_handler![
            list_managed_files,
            get_managed_file,
            set_managed_key,
            pick_project_folder,
            boot_project
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
