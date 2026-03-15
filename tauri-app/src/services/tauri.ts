import { invoke } from "@tauri-apps/api/core";

export interface BackendRuntimeInfo {
  baseUrl: string;
  apiToken: string;
}

const browserRuntime: BackendRuntimeInfo = {
  baseUrl: "",
  apiToken: "",
};

let backendRuntimePromise: Promise<BackendRuntimeInfo> | null = null;

export function isTauriRuntime(): boolean {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
}

export async function getBackendRuntime(): Promise<BackendRuntimeInfo> {
  if (!isTauriRuntime()) {
    return browserRuntime;
  }

  if (!backendRuntimePromise) {
    backendRuntimePromise = invoke<BackendRuntimeInfo>("get_backend_runtime").catch((error) => {
      backendRuntimePromise = null;
      throw error;
    });
  }

  return backendRuntimePromise;
}

export async function setTrayBadge(count: number): Promise<void> {
  if (!isTauriRuntime()) return;
  await invoke("set_tray_badge", { count });
}

export async function getBackendBaseUrl(): Promise<string> {
  const runtime = await getBackendRuntime();
  return runtime.baseUrl;
}

export function resetBackendRuntimeCacheForTests(): void {
  backendRuntimePromise = null;
}
