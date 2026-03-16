<template>
  <div class="report-viewer">
    <div class="report-header">
      <h2>{{ content.title }}</h2>
      <div class="report-meta">
        <span class="type-badge" :class="`type-${content.type}`">
          {{ typeLabel(content.type) }}
        </span>
        <span class="period">{{ content.period.start.slice(0, 10) }} ~ {{ content.period.end.slice(0, 10) }}</span>
      </div>
    </div>

    <div class="sections">
      <div v-for="(section, idx) in content.sections" :key="idx" class="section-card">
        <h3 class="section-title">{{ section.title }}</h3>

        <!-- Chart section -->
        <ReportChart v-if="section.type === 'chart'" :section="section" />

        <!-- List section -->
        <ul v-else-if="section.type === 'list' && section.items && section.items.length > 0" class="section-list">
          <li v-for="(item, i) in section.items" :key="i" class="list-item">
            <template v-if="typeof item === 'string'">{{ item }}</template>
            <template v-else-if="typeof item === 'object' && item !== null">
              <strong v-if="(item as Record<string, unknown>).title">{{ (item as Record<string, unknown>).title }}</strong>
              <span v-if="(item as Record<string, unknown>).description"> — {{ (item as Record<string, unknown>).description }}</span>
              <span v-if="(item as Record<string, unknown>).content"> — {{ (item as Record<string, unknown>).content }}</span>
            </template>
          </li>
        </ul>

        <!-- Narrative -->
        <p v-if="section.narrative" class="narrative">{{ section.narrative }}</p>
      </div>
    </div>

    <!-- Summary -->
    <div v-if="content.summary" class="summary-card">
      <h3>报告总结</h3>
      <p>{{ content.summary }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ReportContent } from "../types/report";
import ReportChart from "./ReportChart.vue";

defineProps<{
  content: ReportContent;
}>();

function typeLabel(type: string): string {
  const labels: Record<string, string> = {
    weekly: "每周简报",
    monthly: "每月报告",
    annual: "年度复盘",
  };
  return labels[type] ?? type;
}
</script>

<style scoped>
.report-viewer {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.report-header h2 {
  margin: 0 0 8px;
  color: var(--color-text);
}

.report-meta {
  display: flex;
  align-items: center;
  gap: 12px;
}

.type-badge {
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 500;
}

.type-weekly {
  color: var(--color-info);
  background: var(--color-info-bg);
}

.type-monthly {
  color: var(--color-success);
  background: var(--color-success-bg);
}

.type-annual {
  color: var(--color-warning);
  background: var(--color-warning-bg);
}

.period {
  font-size: 13px;
  color: var(--color-text-secondary);
}

.sections {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section-card {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 16px;
}

.section-title {
  margin: 0 0 12px;
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text);
}

.section-list {
  list-style: none;
  padding: 0;
  margin: 0 0 12px;
}

.list-item {
  padding: 8px 12px;
  border-left: 3px solid var(--color-primary);
  margin-bottom: 8px;
  font-size: 14px;
  line-height: 1.5;
  color: var(--color-text);
}

.narrative {
  font-size: 14px;
  line-height: 1.7;
  color: var(--color-text);
  white-space: pre-wrap;
  margin: 0;
}

.summary-card {
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 16px;
}

.summary-card h3 {
  margin: 0 0 8px;
  font-size: 15px;
  color: var(--color-text);
}

.summary-card p {
  margin: 0;
  font-size: 14px;
  line-height: 1.7;
  color: var(--color-text-secondary);
  white-space: pre-wrap;
}
</style>
