<template>
  <div class="action-card" :class="`priority-${item.priority}`">
    <div class="card-header">
      <span class="category-tag">{{ item.category }}</span>
      <span class="priority-tag" :class="`p-${item.priority}`">{{ item.priority }}</span>
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
</script>

<style scoped>
.action-card {
  border: 1px solid var(--color-border, #e0e0e0);
  border-left: 4px solid #ccc;
  border-radius: 8px;
  padding: 14px;
  background: var(--color-bg-card, #fff);
}
.priority-high { border-left-color: #e74c3c; }
.priority-medium { border-left-color: #f39c12; }
.priority-low { border-left-color: #27ae60; }

.card-header { display: flex; gap: 6px; margin-bottom: 8px; }
.category-tag, .priority-tag {
  padding: 1px 8px; border-radius: 10px; font-size: 11px; font-weight: 500;
}
.category-tag { background: #e8f4fd; color: #2980b9; }
.p-high { background: #fde8e8; color: #e74c3c; }
.p-medium { background: #fef3e2; color: #e67e22; }
.p-low { background: #e8f8ef; color: #27ae60; }

.card-title { margin: 0 0 6px; font-size: 15px; }
.card-desc { margin: 0 0 8px; font-size: 13px; color: var(--color-text-secondary, #666); line-height: 1.5; }
.due-date { font-size: 12px; color: #e67e22; margin-bottom: 8px; }

.card-actions { display: flex; gap: 8px; align-items: center; }
.btn-sm { padding: 4px 12px; border-radius: 5px; font-size: 12px; cursor: pointer; border: none; }
.btn-complete { background: #27ae60; color: white; }
.btn-dismiss { background: transparent; border: 1px solid #ddd; color: #666; }
.done-badge { font-size: 12px; color: #27ae60; }
.dismissed-badge { font-size: 12px; color: #999; }
</style>
