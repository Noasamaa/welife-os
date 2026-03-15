#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::error::Error;
use std::fs;
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

fn spawn_go_backend(app_handle: &AppHandle) -> Result<(), Box<dyn Error>> {
    let state = app_handle.state::<BackendState>();
    let mut process = state.0.lock().expect("backend state poisoned");
    if process.is_some() {
        return Ok(());
    }

    let backend_data_dir = backend_data_dir(app_handle)?;
    fs::create_dir_all(&backend_data_dir)?;

    let mut command = backend_command(app_handle)?;
    command
        .current_dir(backend_workdir(app_handle)?)
        .env("WELIFE_HOST", "127.0.0.1")
        .env("WELIFE_PORT", "18080")
        .env("WELIFE_DB_PATH", backend_data_dir.join("welife.db"));
    let child = command.spawn()?;

    *process = Some(child);
    Ok(())
}

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

fn backend_command(app_handle: &AppHandle) -> Result<Command, Box<dyn Error>> {
    #[cfg(debug_assertions)]
    {
        let mut command = Command::new(resolve_go_binary());
        command.args(["run", "./cmd/welife"]);
        return Ok(command);
    }

    #[cfg(not(debug_assertions))]
    {
        let exe_suffix = std::env::consts::EXE_SUFFIX;
        let binary = app_handle
            .path()
            .resource_dir()?
            .join("bin")
            .join(format!("welife-engine{}", exe_suffix));
        Ok(Command::new(binary))
    }
}

fn backend_workdir(app_handle: &AppHandle) -> Result<PathBuf, Box<dyn Error>> {
    #[cfg(debug_assertions)]
    {
        return Ok(engine_dir());
    }

    #[cfg(not(debug_assertions))]
    {
        Ok(backend_data_dir(app_handle)?)
    }
}

fn backend_data_dir(app_handle: &AppHandle) -> Result<PathBuf, Box<dyn Error>> {
    Ok(app_handle.path().app_data_dir()?.join("engine"))
}
