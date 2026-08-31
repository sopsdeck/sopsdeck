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

// Tauri runs a synchronous #[tauri::command] on the main UI thread, so any
// blocking I/O freezes the window (the macOS beachball). Each command below
// shells out to the sopsdeck CLI or walks the filesystem, so they are async
// and move that work onto the blocking pool via spawn_blocking.

#[tauri::command]
async fn list_managed_files(path: String) -> Result<Vec<ManagedFile>, String> {
    tauri::async_runtime::spawn_blocking(move || managed::list_in(&PathBuf::from(path)))
        .await
        .map_err(|e| e.to_string())?
}

#[tauri::command]
async fn get_managed_file(path: String, at: Option<String>) -> Result<Vec<Pair>, String> {
    tauri::async_runtime::spawn_blocking(move || load_pairs(&path, at))
        .await
        .map_err(|e| e.to_string())?
}

fn load_pairs(path: &str, at: Option<String>) -> Result<Vec<Pair>, String> {
    let mut args = vec![
        "get".to_string(),
        "-f".to_string(),
        path.to_string(),
        "--output".to_string(),
        "json".to_string(),
    ];
    if let Some(rev) = at.filter(|value| !value.is_empty()) {
        args.push("--at".to_string());
        args.push(rev);
    }
    let refs: Vec<&str> = args.iter().map(String::as_str).collect();
    let out = run_sopsdeck(&refs)?;
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
async fn set_managed_key(path: String, key: String, value: String) -> Result<(), String> {
    run_blocking(move || run_sopsdeck(&["set", &key, &value, "-f", &path]).map(|_| ())).await
}

#[tauri::command]
async fn del_managed_key(path: String, key: String) -> Result<(), String> {
    run_blocking(move || run_sopsdeck(&["del", &key, "-f", &path]).map(|_| ())).await
}

#[tauri::command]
async fn create_managed_file(path: String) -> Result<(), String> {
    run_blocking(move || run_sopsdeck(&["set", "-f", &path]).map(|_| ())).await
}

#[tauri::command]
async fn commit_managed_file(path: String, message: String) -> Result<(), String> {
    let message = message.trim().to_string();
    if message.is_empty() {
        return Err("commit message is required".into());
    }
    run_blocking(move || run_sopsdeck(&["commit", "-m", &message, "-f", &path]).map(|_| ())).await
}

#[tauri::command]
async fn sync_project(path: String) -> Result<(), String> {
    run_blocking(move || {
        let root = git_root(&path)?;
        run_sopsdeck_in(&root, &["sync"]).map(|_| ())
    })
    .await
}

#[tauri::command]
async fn add_recipient(path: String, public_key: String) -> Result<(), String> {
    run_blocking(move || run_sopsdeck(&["recipient", "add", &public_key, "-f", &path]).map(|_| ()))
        .await
}

#[tauri::command]
async fn remove_recipient(path: String, public_key: String) -> Result<(), String> {
    run_blocking(move || {
        run_sopsdeck(&["recipient", "remove", &public_key, "-f", &path]).map(|_| ())
    })
    .await
}

#[tauri::command]
async fn review_managed_file(path: String) -> Result<String, String> {
    run_blocking(move || run_sopsdeck(&["review", "-f", &path])).await
}

#[tauri::command]
async fn history_managed_file(path: String) -> Result<String, String> {
    run_blocking(move || run_sopsdeck(&["history", "-f", &path])).await
}

#[tauri::command]
async fn restore_managed_file(path: String, at: String) -> Result<(), String> {
    let at = at.trim().to_string();
    if at.is_empty() {
        return Err("pick a revision from History".into());
    }
    run_blocking(move || run_sopsdeck(&["restore", "-f", &path, "--at", &at]).map(|_| ())).await
}

#[tauri::command]
async fn publish_managed_file(
    path: String,
    prefix: String,
    yes: bool,
    prune: bool,
) -> Result<String, String> {
    run_blocking(move || publish(&path, &prefix, yes, prune)).await
}

fn publish(path: &str, prefix: &str, yes: bool, prune: bool) -> Result<String, String> {
    let mut args = vec!["publish".to_string(), "-f".to_string(), path.to_string()];
    if !prefix.is_empty() {
        args.push("--prefix".to_string());
        args.push(prefix.to_string());
    }
    if yes {
        args.push("--yes".to_string());
    }
    if prune {
        args.push("--prune".to_string());
    }
    let refs: Vec<&str> = args.iter().map(String::as_str).collect();
    run_sopsdeck(&refs)
}

#[tauri::command]
async fn get_publish_mapping(path: String) -> Result<serde_json::Value, String> {
    run_blocking(move || {
        let out = run_sopsdeck(&["publish", "-f", &path, "--mapping"])?;
        serde_json::from_str(out.trim()).map_err(|error| error.to_string())
    })
    .await
}

#[allow(clippy::unused_async)]
#[tauri::command]
async fn pick_project_folder(app: tauri::AppHandle) -> Option<String> {
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

#[tauri::command]
fn whats_new() -> Result<serde_json::Value, String> {
    serde_json::from_str(include_str!("../../src/whats-new.json"))
        .map_err(|error| error.to_string())
}

#[tauri::command]
fn app_version() -> String {
    env!("CARGO_PKG_VERSION").to_string()
}

// run_blocking runs a blocking closure on Tauri's blocking pool and flattens
// the JoinError into the command's String error type.
async fn run_blocking<F, T>(f: F) -> Result<T, String>
where
    F: FnOnce() -> Result<T, String> + Send + 'static,
    T: Send + 'static,
{
    tauri::async_runtime::spawn_blocking(f)
        .await
        .map_err(|e| e.to_string())?
}

pub fn run() {
    let result = tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_dialog::init())
        .invoke_handler(tauri::generate_handler![
            list_managed_files,
            get_managed_file,
            set_managed_key,
            del_managed_key,
            create_managed_file,
            commit_managed_file,
            sync_project,
            add_recipient,
            remove_recipient,
            review_managed_file,
            history_managed_file,
            restore_managed_file,
            publish_managed_file,
            get_publish_mapping,
            pick_project_folder,
            boot_project,
            whats_new,
            app_version
        ])
        .run(tauri::generate_context!());
    if let Err(error) = result {
        eprintln!("error while running tauri application: {error}");
        std::process::exit(1);
    }
}

#[cfg(test)]
mod version_tests {
    #[test]
    fn cargo_pkg_version_matches_whats_new() {
        let parsed: serde_json::Value =
            serde_json::from_str(include_str!("../../src/whats-new.json")).expect("json");
        let version = parsed
            .get("version")
            .and_then(serde_json::Value::as_str)
            .expect("version");
        assert_eq!(version, env!("CARGO_PKG_VERSION"));
    }
}
