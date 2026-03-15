<template>
  <div class="grid">
    <section class="card block">
      <h2>系统总览</h2>
      <p>桌面壳现在默认连接到 `127.0.0.1:18080`，并读取 Go sidecar 的运行状态。</p>
      <div class="status-list">
        <div class="status-row">
          <span>后端</span>
          <span class="status-pill" :class="backendClass">{{ backendLabel }}</span>
        </div>
        <div class="status-row">
          <span>存储</span>
          <span class="status-pill" :class="storageClass">{{ storageLabel }}</span>
        </div>
        <div class="status-row">
          <span>Ollama</span>
          <span class="status-pill" :class="llmClass">{{ llmLabel }}</span>
        </div>
      </div>
    </section>
    <section class="card block">
      <h2>Phase 0 闭环</h2>
      <ul>
        <li>Go sidecar 默认监听 `127.0.0.1:18080`</li>
        <li>桌面端开发态拉起 `go run ./cmd/welife`</li>
        <li>系统状态页展示 backend / storage / llm 三类健康度</li>
      </ul>
    </section>
    <section class="card block">
      <h2>当前配置</h2>
      <ul>
        <li>存储驱动：{{ systemStatus?.storage.driver ?? "检查中" }}</li>
        <li>数据库路径：{{ systemStatus?.storage.path ?? "检查中" }}</li>
        <li>LLM 模型：{{ systemStatus?.llm.model ?? "检查中" }}</li>
      </ul>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

import { useBackendHealth } from "../composables/useBackendHealth";

const { status, systemStatus } = useBackendHealth();

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
  return systemStatus.value.llm.reachable ? "已连接" : "未连接";
});

const llmClass = computed(() => {
  if (!systemStatus.value) {
    return "pending";
  }
  return systemStatus.value.llm.reachable ? "ok" : "warn";
});
</script>

<style scoped>
.grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 20px;
}

.block {
  padding: 24px;
}

.status-list {
  display: grid;
  gap: 12px;
  margin-top: 20px;
}

.status-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

@media (max-width: 900px) {
  .grid {
    grid-template-columns: 1fr;
  }
}
</style>
