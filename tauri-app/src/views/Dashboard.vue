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
      <div class="card-header">
        <h2>待办事项</h2>
        <span v-if="pendingActionCount > 0" class="badge">{{ pendingActionCount }}</span>
      </div>
      <p class="card-description">来自执行教练的行动项</p>
      <div class="action-summary">
        <div class="summary-row">
          <span class="summary-label">待处理</span>
          <span class="summary-value pending-text">{{ pendingActionCount }}</span>
        </div>
        <div class="summary-row">
          <span class="summary-label">进行中</span>
          <span class="summary-value active-text">{{ inProgressActionCount }}</span>
        </div>
        <div class="summary-row">
          <span class="summary-label">已完成</span>
          <span class="summary-value success-text">{{ completedActionCount }}</span>
        </div>
      </div>
    </section>

    <section class="card block">
      <div class="card-header">
        <h2>提醒</h2>
        <span v-if="pendingReminderCount > 0" class="badge badge-warning">{{ pendingReminderCount }}</span>
      </div>
      <p class="card-description">待处理的提醒通知</p>
      <div class="reminder-summary">
        <div v-if="pendingReminderCount > 0" class="reminder-count">
          <span class="count-number">{{ pendingReminderCount }}</span>
          <span class="count-label">条待处理提醒</span>
        </div>
        <div v-else class="reminder-empty">
          <span>暂无待处理提醒</span>
        </div>
      </div>
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
import { computed, onMounted, ref } from "vue";

import { useBackendHealth } from "../composables/useBackendHealth";
import { fetchActionItems, fetchPendingReminders } from "../services/api";
import type { ActionItem } from "../types/coach";
import type { Reminder } from "../types/reminder";

const { status, systemStatus } = useBackendHealth();

const actionItems = ref<ActionItem[]>([]);
const reminders = ref<Reminder[]>([]);

const pendingActionCount = computed(
  () => actionItems.value.filter((i) => i.status === "pending").length,
);
const inProgressActionCount = computed(
  () => actionItems.value.filter((i) => i.status === "in_progress").length,
);
const completedActionCount = computed(
  () => actionItems.value.filter((i) => i.status === "completed").length,
);
const pendingReminderCount = computed(() => reminders.value.length);

onMounted(async () => {
  try {
    actionItems.value = await fetchActionItems();
  } catch {
    actionItems.value = [];
  }
  try {
    reminders.value = await fetchPendingReminders();
  } catch {
    reminders.value = [];
  }
});

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
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.block {
  padding: 24px;
}

h2 {
  margin: 0 0 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
}

p {
  margin: 0;
  font-size: 13px;
  color: var(--color-text-secondary);
}

ul {
  color: var(--color-text-secondary);
  padding-left: 20px;
  margin: 0;
  font-size: 13px;
}

li {
  margin-bottom: 6px;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.card-description {
  margin: 0 0 16px;
  font-size: 13px;
  color: var(--color-text-muted);
}

.badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-inverse);
  background: var(--color-primary);
}

.badge-warning {
  background: var(--color-warning);
}

.status-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 16px;
}

.status-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-size: 13px;
  color: var(--color-text-secondary);
}

.action-summary {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.summary-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-radius: var(--radius-md);
  background: var(--color-bg-secondary);
}

.summary-label {
  font-size: 13px;
  color: var(--color-text-secondary);
}

.summary-value {
  font-size: 20px;
  font-weight: 700;
  color: var(--color-text);
}

.pending-text {
  color: var(--color-warning);
}

.active-text {
  color: var(--color-primary);
}

.success-text {
  color: var(--color-success);
}

.reminder-summary {
  margin-top: 4px;
}

.reminder-count {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 12px;
  border-radius: var(--radius-lg);
  background: var(--color-warning-bg);
}

.count-number {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-warning);
}

.count-label {
  font-size: 13px;
  color: var(--color-text-secondary);
}

.reminder-empty {
  padding: 12px;
  border-radius: var(--radius-lg);
  background: var(--color-muted-bg);
  font-size: 13px;
  color: var(--color-text-muted);
}

@media (max-width: 768px) {
  .grid {
    grid-template-columns: 1fr;
  }
}
</style>
