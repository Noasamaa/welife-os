import { ref } from "vue";

import { fetchSystemStatus, getAPIBaseURL } from "../services/api";
import type { SystemStatusResponse } from "../types/api";

type BackendStatus = "checking" | "healthy" | "unreachable";

const apiBaseUrl = ref("/api");
const status = ref<BackendStatus>("checking");
const systemStatus = ref<SystemStatusResponse | null>(null);
const errorMessage = ref<string | null>(null);
let pollHandle: number | null = null;

async function checkHealth(): Promise<void> {
  // Don't flash "checking" on subsequent polls — only on first check
  if (status.value !== "healthy" && status.value !== "unreachable") {
    status.value = "checking";
  }
  try {
    apiBaseUrl.value = await getAPIBaseURL();
    systemStatus.value = await fetchSystemStatus();
    status.value = systemStatus.value.backend.status === "ok" ? "healthy" : "unreachable";
    errorMessage.value = null;
  } catch (error) {
    status.value = "unreachable";
    errorMessage.value = error instanceof Error ? error.message : "unknown error";
  }
}

// Start polling once on first import — never stops during app lifetime.
if (pollHandle === null) {
  void checkHealth();
  pollHandle = window.setInterval(() => {
    void checkHealth();
  }, 5000);
}

export function useBackendHealth() {
  return {
    apiBaseUrl,
    checkHealth,
    errorMessage,
    status,
    systemStatus,
  };
}
