#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::error::Error;
use std::fmt::Write as _;
use std::fs;
use std::net::TcpListener;
use std::path::PathBuf;
use std::process::{Child, Command};
use std::sync::Mutex;

use serde::Serialize;
use tauri::image::Image;
use tauri::menu::{MenuBuilder, MenuItemBuilder};
use tauri::tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent};
use tauri::{AppHandle, Manager, RunEvent, WindowEvent};

#[derive(Clone, Serialize)]
#[serde(rename_all = "camelCase")]
struct BackendRuntimeInfo {
    base_url: String,
    api_token: String,
}

#[derive(Default)]
struct BackendProcessState {
    child: Option<Child>,
    runtime: Option<BackendRuntimeInfo>,
}

#[derive(Default)]
struct BackendState(Mutex<BackendProcessState>);

#[tauri::command]
fn set_tray_badge(app: AppHandle, count: u32) -> Result<(), String> {
    let tray = app
        .tray_by_id("main-tray")
        .ok_or_else(|| "tray icon not found".to_string())?;
    let tooltip = if count > 0 {
        format!("WeLife OS - {} 条提醒", count)
    } else {
        "WeLife OS".to_string()
    };
    tray.set_tooltip(Some(&tooltip)).map_err(|e| e.to_string())
}

#[tauri::command]
fn get_backend_runtime(app: AppHandle) -> Result<BackendRuntimeInfo, String> {
    let state = app.state::<BackendState>();
    let process = state.0.lock().expect("backend state poisoned");
    process
        .runtime
        .clone()
        .ok_or_else(|| "backend runtime not initialized".to_string())
}

fn main() {
    tauri::Builder::default()
        .manage(BackendState::default())
        .plugin(tauri_plugin_notification::init())
        .invoke_handler(tauri::generate_handler![
            get_backend_runtime,
            set_tray_badge
        ])
        .setup(|app| {
            let window = app.get_webview_window("main");
            if let Some(ref win) = window {
                win.set_title("WeLife OS")?;
            }

            spawn_go_backend(app.handle())?;
            setup_tray(app)?;

            // Close window → hide to tray instead of quitting
            if let Some(win) = window {
                let win_clone = win.clone();
                win.on_window_event(move |event| {
                    if let WindowEvent::CloseRequested { api, .. } = event {
                        api.prevent_close();
                        let _ = win_clone.hide();
                    }
                });
            }

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

fn setup_tray(app: &tauri::App) -> Result<(), Box<dyn Error>> {
    let show_hide = MenuItemBuilder::with_id("show_hide", "显示/隐藏").build(app)?;
    let quit = MenuItemBuilder::with_id("quit", "退出").build(app)?;
    let menu = MenuBuilder::new(app)
        .item(&show_hide)
        .separator()
        .item(&quit)
        .build()?;
    let icon = Image::from_bytes(include_bytes!("../icons/32x32.png"))?;

    TrayIconBuilder::with_id("main-tray")
        .icon(icon)
        .tooltip("WeLife OS")
        .menu(&menu)
        .on_tray_icon_event(|tray, event| {
            if let TrayIconEvent::Click {
                button: MouseButton::Left,
                button_state: MouseButtonState::Up,
                ..
            } = event
            {
                if let Some(win) = tray.app_handle().get_webview_window("main") {
                    let _ = win.show();
                    let _ = win.set_focus();
                }
            }
        })
        .on_menu_event(|app, event| match event.id().as_ref() {
            "show_hide" => {
                if let Some(win) = app.get_webview_window("main") {
                    if win.is_visible().unwrap_or(false) {
                        let _ = win.hide();
                    } else {
                        let _ = win.show();
                        let _ = win.set_focus();
                    }
                }
            }
            "quit" => {
                let _ = stop_go_backend(app);
                app.exit(0);
            }
            _ => {}
        })
        .build(app)?;

    Ok(())
}

fn spawn_go_backend(app_handle: &AppHandle) -> Result<(), Box<dyn Error>> {
    let state = app_handle.state::<BackendState>();
    let mut process = state.0.lock().expect("backend state poisoned");
    if process.child.is_some() {
        return Ok(());
    }

    let backend_data_dir = backend_data_dir(app_handle)?;
    fs::create_dir_all(&backend_data_dir)?;
    let backend_runtime = build_backend_runtime()?;
    let backend_port = backend_port(&backend_runtime)?;

    let mut command = backend_command(app_handle)?;
    command
        .current_dir(backend_workdir(app_handle)?)
        .env("WELIFE_HOST", "127.0.0.1")
        .env("WELIFE_PORT", backend_port.to_string())
        .env("WELIFE_API_TOKEN", backend_runtime.api_token.as_str())
        .env("WELIFE_DB_PATH", backend_data_dir.join("welife.db"));
    let child = command.spawn()?;

    process.child = Some(child);
    process.runtime = Some(backend_runtime);
    Ok(())
}

fn stop_go_backend(app_handle: &AppHandle) -> Result<(), Box<dyn Error>> {
    let state = app_handle.state::<BackendState>();
    let mut process = state.0.lock().expect("backend state poisoned");

    if let Some(mut child) = process.child.take() {
        let _ = child.kill();
        let _ = child.wait();
    }
    process.runtime = None;

    Ok(())
}

fn build_backend_runtime() -> Result<BackendRuntimeInfo, Box<dyn Error>> {
    let port = select_loopback_port()?;
    let token = generate_api_token()?;
    Ok(BackendRuntimeInfo {
        base_url: format!("http://127.0.0.1:{port}"),
        api_token: token,
    })
}

fn select_loopback_port() -> Result<u16, Box<dyn Error>> {
    let listener = TcpListener::bind(("127.0.0.1", 0))?;
    Ok(listener.local_addr()?.port())
}

fn generate_api_token() -> Result<String, Box<dyn Error>> {
    let mut bytes = [0_u8; 32];
    getrandom::fill(&mut bytes)?;

    let mut token = String::with_capacity(bytes.len() * 2);
    for byte in bytes {
        write!(&mut token, "{byte:02x}")?;
    }
    Ok(token)
}

fn backend_port(runtime: &BackendRuntimeInfo) -> Result<u16, Box<dyn Error>> {
    let base_url = runtime
        .base_url
        .strip_prefix("http://127.0.0.1:")
        .ok_or("invalid backend runtime base URL")?;
    Ok(base_url.parse()?)
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

fn backend_command(_app_handle: &AppHandle) -> Result<Command, Box<dyn Error>> {
    #[cfg(debug_assertions)]
    {
        let mut command = Command::new(resolve_go_binary());
        command.args(["run", "./cmd/welife"]);
        return Ok(command);
    }

    #[cfg(not(debug_assertions))]
    {
        let exe_suffix = std::env::consts::EXE_SUFFIX;
        let binary = _app_handle
            .path()
            .resource_dir()?
            .join("bin")
            .join(format!("welife-engine{}", exe_suffix));
        Ok(Command::new(binary))
    }
}

fn backend_workdir(_app_handle: &AppHandle) -> Result<PathBuf, Box<dyn Error>> {
    #[cfg(debug_assertions)]
    {
        return Ok(engine_dir());
    }

    #[cfg(not(debug_assertions))]
    {
        Ok(backend_data_dir(_app_handle)?)
    }
}

fn backend_data_dir(app_handle: &AppHandle) -> Result<PathBuf, Box<dyn Error>> {
    Ok(app_handle.path().app_data_dir()?.join("engine"))
}

#[cfg(test)]
mod tests {
    use super::{generate_api_token, select_loopback_port};

    #[test]
    fn generate_api_token_returns_hex_string() {
        let token = generate_api_token().expect("token");
        assert_eq!(token.len(), 64);
        assert!(token.chars().all(|ch| ch.is_ascii_hexdigit()));
    }

    #[test]
    fn select_loopback_port_returns_ephemeral_port() {
        let port = select_loopback_port().expect("port");
        assert!(port > 0);
    }
}
