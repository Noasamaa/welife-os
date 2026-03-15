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
}

.format {
  font-size: 12px;
  color: #7a9a8e;
  background: rgba(45, 106, 79, 0.08);
  padding: 2px 8px;
  border-radius: 4px;
}

.job-status {
  display: flex;
  gap: 12px;
  align-items: center;
}

.pill {
  font-size: 12px;
  padding: 2px 10px;
  border-radius: 10px;
  font-weight: 600;
}

.pill.succeeded {
  background: #d4edda;
  color: #155724;
}

.pill.running {
  background: #fff3cd;
  color: #856404;
}

.pill.failed {
  background: #f8d7da;
  color: #721c24;
}

.pill.pending {
  background: #e2e8f0;
  color: #4a5568;
}

.count {
  font-size: 13px;
  color: #48625c;
}

.empty {
  color: #7a9a8e;
  text-align: center;
  padding: 20px;
}
</style>
