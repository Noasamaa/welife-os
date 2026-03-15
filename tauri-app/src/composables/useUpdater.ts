import { ref } from "vue";

const updaterDisabledReason = "当前构建未启用签名更新，已关闭在线更新入口。";

export function useUpdater() {
  const enabled = ref(false);
  const updateAvailable = ref(false);
  const updateVersion = ref("");
  const checking = ref(false);
  const downloading = ref(false);
  const disabledReason = ref(updaterDisabledReason);
  const error = ref<string | null>(updaterDisabledReason);

  async function checkForUpdate(): Promise<void> {
    error.value = updaterDisabledReason;
  }

  async function downloadAndInstall(): Promise<void> {
    error.value = updaterDisabledReason;
  }

  return {
    enabled,
    updateAvailable,
    updateVersion,
    checking,
    downloading,
    disabledReason,
    error,
    checkForUpdate,
    downloadAndInstall,
  };
}
