<template>
  <section class="page">
    <div class="page-header">
      <h2>报告中心</h2>
      <p class="subtitle">AI 驱动的人生报告：每周简报、月报与年度复盘</p>
    </div>

    <!-- Generate Panel -->
    <div class="card generate-panel">
      <h3>生成新报告</h3>
      <div class="generate-form">
        <select v-model="selectedConversation" class="form-select">
          <option value="">选择对话...</option>
          <option
            v-for="conv in conversations"
            :key="conv.id"
            :value="conv.id"
          >
            {{ conv.title || conv.id }} ({{ conv.platform }}, {{ conv.message_count }} 条)
          </option>
        </select>

        <div class="type-selector">
          <label
            v-for="t in reportTypes"
            :key="t.value"
            class="type-option"
            :class="{ active: selectedType === t.value }"
          >
            <input
              type="radio"
              :value="t.value"
              v-model="selectedType"
              class="sr-only"
            />
            {{ t.label }}
          </label>
        </div>

        <button
          class="btn-primary"
          :disabled="!selectedConversation || generating"
          @click="handleGenerate"
        >
          {{ generating ? '生成中...' : '生成报告' }}
        </button>
      </div>
    </div>

    <!-- Error -->
    <div v-if="error" class="card error-banner">{{ error }}</div>

    <!-- Report List -->
    <div class="card list-panel">
      <div class="panel-header">
        <h3>历史报告</h3>
        <button class="btn-secondary" @click="loadReports">刷新</button>
      </div>

      <div v-if="loading && !currentReport" class="loading">加载中...</div>

      <div v-if="reports.length === 0 && !loading" class="empty">
        暂无报告，选择对话并生成你的第一份人生报告。
      </div>

      <div class="report-list">
        <div
          v-for="r in reports"
          :key="r.id"
          class="report-item"
          :class="{ active: currentReport?.id === r.id }"
          @click="handleSelect(r.id)"
        >
          <div class="report-info">
            <span class="report-title">{{ r.title || r.id.slice(0, 20) }}</span>
            <div class="report-badges">
              <span class="type-badge" :class="`type-${r.type}`">{{ typeLabel(r.type) }}</span>
              <span class="status-badge" :class="`status-${r.status}`">{{ statusLabel(r.status) }}</span>
            </div>
          </div>
          <div class="report-meta">
            <span>{{ r.period_start?.slice(0, 10) }} ~ {{ r.period_end?.slice(0, 10) }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Report Detail -->
    <div v-if="currentReport && parsedContent" class="card detail-panel">
      <div class="panel-header">
        <h3>报告内容</h3>
        <div class="panel-actions">
          <button class="btn-secondary" @click="handleExportHTML">导出 HTML</button>
          <button class="btn-secondary" @click="handleExportPDF">导出 PDF</button>
          <button class="btn-danger" @click="handleDelete">删除</button>
        </div>
      </div>
      <ReportViewer :content="parsedContent" />
    </div>

    <div v-else-if="currentReport && currentReport.status === 'running'" class="card detail-panel">
      <div class="loading">报告生成中，请稍后刷新...</div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useReport } from "../composables/useReport";
import { fetchConversations, reportHTMLUrl, reportPDFUrl } from "../services/api";
import type { Conversation } from "../types/import";
import type { ReportType } from "../types/report";
import ReportViewer from "../components/ReportViewer.vue";

const {
  reports,
  currentReport,
  parsedContent,
  loading,
  generating,
  error,
  loadReports,
  loadReport,
  generate,
  remove,
} = useReport();

const conversations = ref<Conversation[]>([]);
const selectedConversation = ref("");
const selectedType = ref<ReportType>("weekly");

const reportTypes = [
  { value: "weekly" as ReportType, label: "每周简报" },
  { value: "monthly" as ReportType, label: "每月报告" },
  { value: "annual" as ReportType, label: "年度复盘" },
];

onMounted(async () => {
  await Promise.all([loadReports(), loadConversations()]);
});

async function loadConversations() {
  try {
    conversations.value = await fetchConversations();
  } catch (e: any) {
    error.value = e.message ?? "加载对话失败";
  }
}

async function handleGenerate() {
  if (!selectedConversation.value) return;
  await generate(selectedType.value, selectedConversation.value);
}

async function handleSelect(id: string) {
  await loadReport(id);
}

async function handleDelete() {
  if (!currentReport.value) return;
  await remove(currentReport.value.id);
}

function handleExportHTML() {
  if (!currentReport.value) return;
  window.open(reportHTMLUrl(currentReport.value.id), "_blank");
}

function handleExportPDF() {
  if (!currentReport.value) return;
  window.open(reportPDFUrl(currentReport.value.id), "_blank");
}

function typeLabel(type: string): string {
  const m: Record<string, string> = { weekly: "周报", monthly: "月报", annual: "年报" };
  return m[type] ?? type;
}

function statusLabel(status: string): string {
  const m: Record<string, string> = { running: "生成中", completed: "已完成", failed: "失败" };
  return m[status] ?? status;
}
</script>

<style scoped>
.page {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.page-header h2 { margin: 0; }
.subtitle { color: var(--color-text-secondary, #666); margin: 4px 0 0; font-size: 14px; }

.card {
  background: var(--color-bg-card, #fff);
  border: 1px solid var(--color-border, #e0e0e0);
  border-radius: 10px;
  padding: 20px;
}

.generate-panel h3 { margin: 0 0 12px; }

.generate-form {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.form-select {
  flex: 1;
  min-width: 200px;
  padding: 8px 12px;
  border: 1px solid var(--color-border, #ddd);
  border-radius: 6px;
  font-size: 14px;
  background: var(--color-bg, #fff);
}

.type-selector { display: flex; gap: 4px; }

.type-option {
  padding: 6px 14px;
  border: 1px solid var(--color-border, #ddd);
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s;
}

.type-option.active {
  background: var(--color-primary, #4a90d9);
  color: white;
  border-color: var(--color-primary, #4a90d9);
}

.sr-only { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0,0,0,0); }

.btn-primary {
  padding: 8px 20px;
  background: var(--color-primary, #4a90d9);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  white-space: nowrap;
}

.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

.btn-secondary {
  padding: 6px 14px;
  background: transparent;
  border: 1px solid var(--color-border, #ddd);
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
}

.btn-danger {
  padding: 6px 14px;
  background: #e74c3c;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
}

.error-banner { background: #fde8e8; border-color: #e74c3c; color: #c0392b; }

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.panel-header h3 { margin: 0; }

.panel-actions { display: flex; gap: 8px; align-items: center; }

.loading, .empty {
  text-align: center;
  padding: 24px;
  color: var(--color-text-secondary, #888);
  font-size: 14px;
}

.report-list { display: flex; flex-direction: column; gap: 8px; }

.report-item {
  padding: 12px;
  border: 1px solid var(--color-border, #eee);
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s;
}

.report-item:hover { background: var(--color-bg-secondary, #f8f8f8); }
.report-item.active { border-color: var(--color-primary, #4a90d9); background: #f0f7ff; }

.report-info { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 8px; }

.report-title { font-weight: 500; font-size: 14px; }

.report-badges { display: flex; gap: 6px; }

.type-badge, .status-badge {
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 500;
}

.type-weekly { background: #e8f4fd; color: #2980b9; }
.type-monthly { background: #e8f8ef; color: #27ae60; }
.type-annual { background: #fef3e2; color: #e67e22; }

.status-running { background: #fef3e2; color: #e67e22; }
.status-completed { background: #e8f8ef; color: #27ae60; }
.status-failed { background: #fde8e8; color: #e74c3c; }

.report-meta { margin-top: 4px; font-size: 12px; color: var(--color-text-secondary, #888); }
</style>
