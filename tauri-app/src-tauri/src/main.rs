#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::error::Error;
use std::path::PathBuf;
use std::process::{Child, Command};
use std::sync::Mutex;

use tauri::{AppHandle, Manager, RunEvent};

#[derive(Default)]
struct BackendState(Mutex<Option<Child>>);

fn main() {
    tauri::Builder::default()
        .manage(BackendState::default())
        .setup(|app| {
            let window = app.get_webview_window("main");
            if let Some(window) = window {
                window.set_title("WeLife OS")?;
            }

            #[cfg(debug_assertions)]
            spawn_go_backend(app.handle())?;

            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("error while building WeLife desktop shell")
        .run(|app_handle, event| match event {
            RunEvent::ExitRequested { .. } | RunEvent::Exit => {
                let _ = stop_go_backend(app_handle);
            }
            _ => {}
        });
}

#[cfg(debug_assertions)]
fn spawn_go_backend(app_handle: &AppHandle) -> Result<(), Box<dyn Error>> {
    let state = app_handle.state::<BackendState>();
    let mut process = state.0.lock().expect("backend state poisoned");
    if process.is_some() {
        return Ok(());
    }

    let child = Command::new(resolve_go_binary())
        .args(["run", "./cmd/welife"])
        .current_dir(engine_dir())
        .env("WELIFE_HOST", "127.0.0.1")
        .env("WELIFE_PORT", "18080")
        .spawn()?;

    *process = Some(child);
    Ok(())
}

#[cfg(debug_assertions)]
fn stop_go_backend(app_handle: &AppHandle) -> Result<(), Box<dyn Error>> {
    let state = app_handle.state::<BackendState>();
    let mut process = state.0.lock().expect("backend state poisoned");

    if let Some(mut child) = process.take() {
        let _ = child.kill();
        let _ = child.wait();
    }

    Ok(())
}

#[cfg(debug_assertions)]
fn engine_dir() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("../../engine")
}

#[cfg(debug_assertions)]
fn resolve_go_binary() -> String {
    if let Ok(value) = std::env::var("WELIFE_GO_BINARY") {
        if !value.trim().is_empty() {
            return value;
        }
    }

    if let Ok(home) = std::env::var("HOME") {
        let local_go = PathBuf::from(home).join(".local/bin/go");
        if local_go.exists() {
            return local_go.to_string_lossy().into_owned();
        }
    }

    "go".to_string()
}

#[cfg(not(debug_assertions))]
fn stop_go_backend(_: &AppHandle) -> Result<(), Box<dyn Error>> {
    Ok(())
}
