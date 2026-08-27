mod managed;
mod sidecar;

use std::path::PathBuf;

use serde::Serialize;
use tauri_plugin_dialog::DialogExt;

use managed::ManagedFile;
use sidecar::{git_root, run_sopsdeck, run_sopsdeck_in};

#[derive(Serialize)]
struct Pair {
    key: String,
    value: String,
}

#[tauri::command]
fn list_managed_files(path: String) -> Result<Vec<ManagedFile>, String> {
    managed::list_in(&PathBuf::from(path))
}

#[tauri::command]
fn get_managed_file(path: String) -> Result<Vec<Pair>, String> {
    let out = run_sopsdeck(&["get", "-f", &path, "--output", "json"])?;
    let map: serde_json::Map<String, serde_json::Value> =
        serde_json::from_str(out.trim()).map_err(|error| error.to_string())?;
    let mut pairs: Vec<Pair> = map
        .into_iter()
        .map(|(key, value)| Pair {
            value: match value {
                serde_json::Value::String(text) => text,
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
fn commit_managed_file(path: String, message: String) -> Result<(), String> {
    let message = message.trim();
    if message.is_empty() {
        return Err("commit message is required".into());
    }
    run_sopsdeck(&["commit", "-m", message, "-f", &path]).map(|_| ())
}

#[tauri::command]
fn sync_project(path: String) -> Result<(), String> {
    let root = git_root(&path)?;
    run_sopsdeck_in(&root, &["sync"]).map(|_| ())
}

#[tauri::command]
fn pick_project_folder(app: tauri::AppHandle) -> Option<String> {
    let folder = app
        .dialog()
        .file()
        .set_title("Add project folder")
        .blocking_pick_folder();
    folder
        .and_then(|picked| picked.into_path().ok())
        .map(|path| path.to_string_lossy().into_owned())
}

#[tauri::command]
fn boot_project() -> Option<String> {
    std::env::var("SOPSDECK_DEV_PROJECT")
        .ok()
        .filter(|value| !value.is_empty())
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let result = tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_dialog::init())
        .invoke_handler(tauri::generate_handler![
            list_managed_files,
            get_managed_file,
            set_managed_key,
            commit_managed_file,
            sync_project,
            pick_project_folder,
            boot_project
        ])
        .run(tauri::generate_context!());
    if let Err(error) = result {
        eprintln!("error while running tauri application: {error}");
        std::process::exit(1);
    }
}
