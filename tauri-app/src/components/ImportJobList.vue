<template>
  <div v-if="jobs.length" class="job-list">
    <div v-for="job in jobs" :key="job.id" class="job-row card">
      <div class="job-info">
        <span class="file-name">{{ job.file_name }}</span>
        <span class="format">{{ job.format }}</span>
      </div>
      <div class="job-status">
        <span class="pill" :class="job.status">{{ statusLabel(job.status) }}</span>
        <span v-if="job.message_count" class="count">{{ job.message_count }} 条消息</span>
      </div>
    </div>
  </div>
  <p v-else class="empty">暂无导入记录</p>
</template>

<script setup lang="ts">
import type { ImportJob } from "../types/import";

defineProps<{
  jobs: ImportJob[];
}>();

function statusLabel(status: string): string {
  const map: Record<string, string> = {
    pending: "等待中",
    running: "导入中",
    succeeded: "已完成",
    failed: "失败",
  };
  return map[status] ?? status;
}
</script>

<style scoped>
.job-list {
  display: grid;
  gap: 8px;
}

.job-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
}

.job-info {
  display: flex;
  gap: 8px;
  align-items: center;
}

.file-name {
  font-weight: 600;
  color: var(--color-text);
}

.format {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text-secondary);
  background: var(--color-bg-tertiary);
  padding: 2px 8px;
  border-radius: var(--radius-sm);
}

.job-status {
  display: flex;
  gap: 12px;
  align-items: center;
}

.pill {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  font-weight: 500;
}

.pill.succeeded {
  color: var(--color-success);
  background: var(--color-success-bg);
}

.pill.running {
  color: var(--color-warning);
  background: var(--color-warning-bg);
}

.pill.failed {
  color: var(--color-danger);
  background: var(--color-danger-bg);
}

.pill.pending {
  color: var(--color-text-secondary);
  background: var(--color-muted-bg);
}

.count {
  font-size: 13px;
  color: var(--color-text-secondary);
}

.empty {
  color: var(--color-text-muted);
  text-align: center;
  padding: 20px;
}
</style>
