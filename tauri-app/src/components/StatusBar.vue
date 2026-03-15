<template>
  <div class="bar card">
    <div>
      <p class="label">后端状态</p>
      <span class="status-pill" :class="backendClass">{{ backendLabel }}</span>
    </div>
    <div>
      <p class="label">存储状态</p>
      <span class="status-pill" :class="storageClass">{{ storageLabel }}</span>
    </div>
    <div>
      <p class="label">Ollama 状态</p>
      <span class="status-pill" :class="llmClass">{{ llmLabel }}</span>
    </div>
    <div>
      <p class="label">API 地址</p>
      <strong>{{ apiBaseUrl }}</strong>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

import { useBackendHealth } from "../composables/useBackendHealth";

const { apiBaseUrl, status, systemStatus } = useBackendHealth();

const backendLabel = computed(() => {
  if (status.value === "healthy") {
    return `在线 · v${systemStatus.value?.backend.version ?? "0.1.0"}`;
  }
  if (status.value === "unreachable") {
    return "未连接";
  }
  return "检查中";
});

const backendClass = computed(() => {
  if (status.value === "healthy") {
    return "ok";
  }
  if (status.value === "unreachable") {
    return "warn";
  }
  return "pending";
});

const storageLabel = computed(() => {
  if (!systemStatus.value) {
    return "检查中";
  }
  return systemStatus.value.storage.ready ? "SQLCipher 已就绪" : "存储未就绪";
});

const storageClass = computed(() => {
  if (!systemStatus.value) {
    return "pending";
  }
  return systemStatus.value.storage.ready ? "ok" : "warn";
});

const llmLabel = computed(() => {
  if (!systemStatus.value) {
    return "检查中";
  }
  return systemStatus.value.llm.reachable
    ? `已连接 · ${systemStatus.value.llm.model}`
    : "未连接";
});

const llmClass = computed(() => {
  if (!systemStatus.value) {
    return "pending";
  }
  return systemStatus.value.llm.reachable ? "ok" : "warn";
});
</script>

<style scoped>
.bar {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
  padding: 18px 20px;
}

.label {
  margin: 0 0 6px;
  font-size: 12px;
  color: var(--color-text-muted);
}

strong {
  word-break: break-all;
  color: var(--color-text);
}

@media (max-width: 1100px) {
  .bar {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .bar {
    grid-template-columns: 1fr;
  }
}
</style>
