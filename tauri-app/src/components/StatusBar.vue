<template>
  <div class="bar card">
    <div>
      <p class="label">后端状态</p>
      <span class="status-pill" :class="backendClass">{{ backendLabel }}</span>
      <p class="hint">{{ apiBaseUrl }}</p>
    </div>
    <div>
      <p class="label">存储状态</p>
      <span class="status-pill" :class="storageClass">{{ storageLabel }}</span>
    </div>
    <div>
      <p class="label">{{ llmSectionLabel }}</p>
      <span class="status-pill" :class="llmClass">{{ llmLabel }}</span>
      <p v-if="llmBaseUrl" class="hint">{{ llmBaseUrl }}</p>
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

const llmSectionLabel = computed(() => {
  if (!systemStatus.value) return "LLM 状态";
  const provider = systemStatus.value.llm.provider;
  if (provider === "ollama") return "Ollama 状态";
  if (provider === "openai-compatible") return "LLM 状态 (云端)";
  return `LLM 状态 (${provider})`;
});

const llmBaseUrl = computed(() => {
  return systemStatus.value?.llm.base_url ?? null;
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
  display: flex;
  flex-wrap: wrap;
  gap: 12px 24px;
  padding: 14px 16px;
}

.bar > div {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.label {
  margin: 0;
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text-muted);
  white-space: nowrap;
}

.hint {
  margin: 0;
  font-size: 11px;
  color: var(--color-text-muted);
  word-break: break-all;
  line-height: 1.3;
}

.bar :deep(.status-pill) {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  padding: 3px 10px;
}
</style>
