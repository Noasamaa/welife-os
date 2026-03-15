import { onMounted, ref } from "vue";

import { API_BASE_URL, fetchSystemStatus } from "../services/api";
import type { SystemStatusResponse } from "../types/api";

type BackendStatus = "checking" | "healthy" | "unreachable";

const status = ref<BackendStatus>("checking");
const systemStatus = ref<SystemStatusResponse | null>(null);
const errorMessage = ref<string | null>(null);
let initialized = false;
let pollHandle: number | null = null;

export function useBackendHealth() {
  async function checkHealth(): Promise<void> {
    status.value = "checking";
    try {
      systemStatus.value = await fetchSystemStatus();
      status.value = systemStatus.value.backend.status === "ok" ? "healthy" : "unreachable";
      errorMessage.value = null;
    } catch (error) {
      status.value = "unreachable";
      errorMessage.value = error instanceof Error ? error.message : "unknown error";
    }
  }

  onMounted(() => {
    if (!initialized) {
      initialized = true;
      void checkHealth();
      pollHandle = window.setInterval(() => {
        void checkHealth();
      }, 5000);
    }
  });

  return {
    apiBaseUrl: API_BASE_URL,
    checkHealth,
    errorMessage,
    status,
    systemStatus,
  };
}
