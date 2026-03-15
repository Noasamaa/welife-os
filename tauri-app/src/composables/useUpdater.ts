import { ref } from "vue";
import { isTauriRuntime } from "../services/tauri";

export function useUpdater() {
  const updateAvailable = ref(false);
  const updateVersion = ref("");
  const checking = ref(false);
  const downloading = ref(false);
  const error = ref<string | null>(null);

  let cachedUpdate: { downloadAndInstall: () => Promise<void> } | null = null;

  async function checkForUpdate(): Promise<void> {
    if (!isTauriRuntime()) {
      error.value = "仅在桌面客户端中可用";
      return;
    }

    checking.value = true;
    error.value = null;
    updateAvailable.value = false;
    updateVersion.value = "";
    cachedUpdate = null;

    try {
      const { check } = await import("@tauri-apps/plugin-updater");
      const update = await check();
      if (update && update.version) {
        updateAvailable.value = true;
        updateVersion.value = update.version;
        cachedUpdate = update;
      }
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : "检查更新失败";
      error.value = message;
    } finally {
      checking.value = false;
    }
  }

  async function downloadAndInstall(): Promise<void> {
    if (!cachedUpdate) {
      error.value = "没有可用的更新";
      return;
    }

    downloading.value = true;
    error.value = null;

    try {
      await cachedUpdate.downloadAndInstall();
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : "下载安装失败";
      error.value = message;
    } finally {
      downloading.value = false;
    }
  }

  return {
    updateAvailable,
    updateVersion,
    checking,
    downloading,
    error,
    checkForUpdate,
    downloadAndInstall,
  };
}
