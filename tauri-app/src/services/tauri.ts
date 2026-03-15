import { invoke } from "@tauri-apps/api/core";

export function isTauriRuntime(): boolean {
  return "__TAURI_INTERNALS__" in window;
}

export async function setTrayBadge(count: number): Promise<void> {
  if (!isTauriRuntime()) return;
  await invoke("set_tray_badge", { count });
}

