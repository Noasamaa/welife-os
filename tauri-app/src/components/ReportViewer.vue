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
              <strong v-if="(item as any).title">{{ (item as any).title }}</strong>
              <span v-if="(item as any).description"> — {{ (item as any).description }}</span>
              <span v-if="(item as any).content"> — {{ (item as any).content }}</span>
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
}

.report-meta {
  display: flex;
  align-items: center;
  gap: 12px;
}

.type-badge {
  padding: 2px 10px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 600;
}

.type-weekly { background: #e8f4fd; color: #2980b9; }
.type-monthly { background: #e8f8ef; color: #27ae60; }
.type-annual { background: #fef3e2; color: #e67e22; }

.period {
  font-size: 13px;
  color: var(--color-text-secondary, #888);
}

.sections {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section-card {
  background: var(--color-bg-card, #fff);
  border: 1px solid var(--color-border, #e0e0e0);
  border-radius: 8px;
  padding: 16px;
}

.section-title {
  margin: 0 0 12px;
  font-size: 16px;
}

.section-list {
  list-style: none;
  padding: 0;
  margin: 0 0 12px;
}

.list-item {
  padding: 8px 12px;
  border-left: 3px solid var(--color-primary, #4a90d9);
  margin-bottom: 8px;
  font-size: 14px;
  line-height: 1.5;
}

.narrative {
  font-size: 14px;
  line-height: 1.7;
  color: var(--color-text, #333);
  white-space: pre-wrap;
  margin: 0;
}

.summary-card {
  background: var(--color-bg-secondary, #f8f9fa);
  border-radius: 8px;
  padding: 16px;
}

.summary-card h3 {
  margin: 0 0 8px;
  font-size: 15px;
}

.summary-card p {
  margin: 0;
  font-size: 14px;
  line-height: 1.7;
  white-space: pre-wrap;
}
</style>
