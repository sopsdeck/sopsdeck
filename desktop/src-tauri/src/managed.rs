use std::path::Path;

use serde::Serialize;

#[derive(Serialize, Debug, PartialEq, Eq)]
pub struct ManagedFile {
    pub name: String,
    pub path: String,
    pub rel: String,
}

pub fn list_in(root: &Path) -> Result<Vec<ManagedFile>, String> {
    if !root.is_dir() {
        return Err("not a folder".into());
    }
    let mut files = Vec::new();
    walk_managed(root, root, &mut files);
    files.sort_by(|a, b| a.rel.cmp(&b.rel));
    Ok(files)
}

fn is_dotenv_name(name: &str) -> bool {
    name == ".env" || name.starts_with(".env.") || name.to_ascii_lowercase().ends_with(".env")
}

fn is_structured_name(name: &str) -> bool {
    Path::new(name)
        .extension()
        .and_then(|ext| ext.to_str())
        .is_some_and(|ext| matches!(ext.to_ascii_lowercase().as_str(), "json" | "yaml" | "yml"))
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
    let Some(slice) = data.get(..n) else {
        return false;
    };
    let sample = String::from_utf8_lossy(slice);
    sample.contains("\"sops\"") || sample.contains("sops:")
}

fn walk_managed(root: &Path, prefix: &Path, out: &mut Vec<ManagedFile>) {
    let Ok(entries) = std::fs::read_dir(root) else {
        return;
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
        let rel = path.strip_prefix(prefix).map_or_else(
            |_| path.to_string_lossy().into_owned(),
            |p| p.to_string_lossy().into_owned(),
        );
        out.push(ManagedFile {
            name,
            path: path.to_string_lossy().into_owned(),
            rel,
        });
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    fn write(path: &Path, body: &str) {
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent).expect("create parent");
        }
        fs::write(path, body).expect("write fixture");
    }

    #[test]
    fn list_in_rejects_a_file() {
        let dir = tempfile::tempdir().expect("tempdir");
        let file = dir.path().join("not-a-folder");
        write(&file, "");
        let err = list_in(&file).expect_err("file is not a folder");
        assert_eq!(err, "not a folder");
    }

    #[test]
    fn list_in_finds_dotenv_and_sops_structured_files() {
        let dir = tempfile::tempdir().expect("tempdir");
        let root = dir.path();
        write(&root.join(".env.production"), "HELLO=world\n");
        write(&root.join("plain.json"), "{\"HELLO\":\"world\"}\n");
        write(
            &root.join("secrets.json"),
            "{\n  \"HELLO\": \"world\",\n  \"sops\": {}\n}\n",
        );
        write(&root.join("nested/app.yaml"), "sops:\n  kms: []\n");
        write(&root.join("node_modules/skip.env"), "NO=pe\n");
        write(&root.join("target/skip.env"), "NO=pe\n");

        let files = list_in(root).expect("list");
        let rels: Vec<_> = files.iter().map(|f| f.rel.as_str()).collect();
        assert_eq!(
            rels,
            vec![".env.production", "nested/app.yaml", "secrets.json"]
        );
    }

    #[test]
    fn dotenv_names_include_env_production_suffix() {
        assert!(is_dotenv_name(".env"));
        assert!(is_dotenv_name(".env.production"));
        assert!(is_dotenv_name("local.env"));
        assert!(!is_dotenv_name("readme.md"));
        assert!(!is_dotenv_name("env.sample"));
    }
}
