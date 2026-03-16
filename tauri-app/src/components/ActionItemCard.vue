<template>
  <div class="action-card" :class="`priority-${item.priority}`">
    <div class="card-header">
      <span class="category-tag">{{ categoryLabel(item.category) }}</span>
      <span class="priority-tag" :class="`p-${item.priority}`">{{ priorityLabel(item.priority) }}</span>
    </div>
    <h4 class="card-title">{{ item.title }}</h4>
    <p class="card-desc">{{ item.description }}</p>
    <div v-if="item.due_date" class="due-date">截止: {{ item.due_date.slice(0, 10) }}</div>
    <div class="card-actions">
      <button
        v-if="item.status === 'pending'"
        class="btn-sm btn-complete"
        @click="$emit('complete', item.id)"
      >完成</button>
      <button
        v-if="item.status === 'pending'"
        class="btn-sm btn-dismiss"
        @click="$emit('dismiss', item.id)"
      >取消</button>
      <span v-if="item.status === 'completed'" class="done-badge">已完成</span>
      <span v-if="item.status === 'dismissed'" class="dismissed-badge">已取消</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ActionItem } from "../types/coach";

defineProps<{ item: ActionItem }>();
defineEmits<{
  complete: [id: string];
  dismiss: [id: string];
}>();

const CATEGORY_LABELS: Record<string, string> = {
  project: "项目",
  contact: "联系人",
  decision: "决策",
  followup: "跟进",
  general: "通用",
};

const PRIORITY_LABELS: Record<string, string> = {
  high: "高",
  medium: "中",
  low: "低",
};

function categoryLabel(category: string): string {
  return CATEGORY_LABELS[category] ?? category;
}

function priorityLabel(priority: string): string {
  return PRIORITY_LABELS[priority] ?? priority;
}
</script>

<style scoped>
.action-card {
  border: 1px solid var(--color-border);
  border-left: 4px solid var(--color-border-strong);
  border-radius: var(--radius-lg);
  padding: 16px;
  background: var(--color-bg-card);
}

.priority-high {
  border-left-color: var(--color-danger);
}

.priority-medium {
  border-left-color: var(--color-warning);
}

.priority-low {
  border-left-color: var(--color-success);
}

.card-header {
  display: flex;
  gap: 6px;
  margin-bottom: 8px;
}

.category-tag,
.priority-tag {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 500;
}

.category-tag {
  color: var(--color-info);
  background: var(--color-info-bg);
}

.p-high {
  color: var(--color-danger);
  background: var(--color-danger-bg);
}

.p-medium {
  color: var(--color-warning);
  background: var(--color-warning-bg);
}

.p-low {
  color: var(--color-success);
  background: var(--color-success-bg);
}

.card-title {
  margin: 0 0 6px;
  font-size: 15px;
  color: var(--color-text);
}

.card-desc {
  margin: 0 0 8px;
  font-size: 13px;
  color: var(--color-text-secondary);
  line-height: 1.5;
}

.due-date {
  font-size: 12px;
  color: var(--color-warning);
  margin-bottom: 8px;
}

.card-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.btn-sm {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 4px 10px;
  border-radius: var(--radius-md);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  border: 1px solid transparent;
  transition: all var(--transition-fast);
}

.btn-complete {
  background: var(--color-success);
  color: var(--color-text-inverse);
}

.btn-complete:hover {
  opacity: 0.9;
}

.btn-dismiss {
  background: transparent;
  border-color: var(--color-border);
  color: var(--color-text-secondary);
}

.btn-dismiss:hover {
  background: var(--color-bg-hover);
  color: var(--color-text);
}

.done-badge {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-success);
}

.dismissed-badge {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text-muted);
}
</style>
