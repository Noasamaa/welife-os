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
              <span class="status-badge" :class="`status-${r.status}`">
                <span v-if="r.status === 'running'" class="spinner-sm"></span>
                {{ statusLabel(r.status) }}
              </span>
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
import { onMounted, onUnmounted, ref, watch } from "vue";
import { useReport } from "../composables/useReport";
import { fetchConversations, fetchReportExportBlob } from "../services/api";
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
let pollHandle: ReturnType<typeof setInterval> | null = null;

const reportTypes = [
  { value: "weekly" as ReportType, label: "每周简报" },
  { value: "monthly" as ReportType, label: "每月报告" },
  { value: "annual" as ReportType, label: "年度复盘" },
];

onMounted(async () => {
  await Promise.all([loadReports(), loadConversations()]);
});

onUnmounted(() => {
  stopPolling();
});

watch(
  () => currentReport.value?.status,
  (status) => {
    if (status === "running") {
      startPolling();
    }
  },
);

watch(
  () => reports.value.some((r) => r.status === "running"),
  (hasRunning) => {
    if (hasRunning) {
      startPolling();
    }
  },
);

function startPolling() {
  if (pollHandle !== null) return; // already polling
  pollHandle = setInterval(() => {
    void loadReports();
    if (currentReport.value) {
      void loadReport(currentReport.value.id);
    }
    // Stop when nothing is running
    if (
      currentReport.value?.status !== "running" &&
      !reports.value.some((r) => r.status === "running")
    ) {
      stopPolling();
    }
  }, 3000);
}

async function loadConversations() {
  try {
    conversations.value = await fetchConversations();
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : "加载对话失败";
  }
}

async function handleGenerate() {
  if (!selectedConversation.value) return;
  const result = await generate(selectedType.value, selectedConversation.value);
  if (result) {
    await loadReports();
  }
}

async function handleSelect(id: string) {
  await loadReport(id);
}

async function handleDelete() {
  if (!currentReport.value) return;
  await remove(currentReport.value.id);
}

async function handleExportHTML() {
  if (!currentReport.value) return;
  await exportReport("html");
}

async function handleExportPDF() {
  if (!currentReport.value) return;
  await exportReport("pdf");
}

function typeLabel(type: string): string {
  const m: Record<string, string> = { weekly: "周报", monthly: "月报", annual: "年报" };
  return m[type] ?? type;
}

function statusLabel(status: string): string {
  const m: Record<string, string> = { running: "生成中", completed: "已完成", failed: "失败" };
  return m[status] ?? status;
}

function stopPolling() {
  if (pollHandle !== null) {
    clearInterval(pollHandle);
    pollHandle = null;
  }
}

async function exportReport(format: "html" | "pdf") {
  if (!currentReport.value) return;
  try {
    const blob = await fetchReportExportBlob(currentReport.value.id, format);
    const objectURL = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = objectURL;
    link.download = buildExportFilename(format);
    document.body.appendChild(link);
    link.click();
    link.remove();
    window.setTimeout(() => URL.revokeObjectURL(objectURL), 30_000);
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : "导出报告失败";
  }
}

function buildExportFilename(format: "html" | "pdf"): string {
  const rawTitle = currentReport.value?.title || currentReport.value?.id || "welife-report";
  const safeTitle = rawTitle.replace(/[\\/:*?"<>|]+/g, "-").slice(0, 80);
  return `${safeTitle}.${format}`;
}
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
}

.subtitle {
  color: var(--color-text-secondary);
  margin: 4px 0 0;
  font-size: 13px;
}

.card {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 24px;
  box-shadow: var(--shadow-sm);
}

.generate-panel h3 {
  margin: 0 0 12px;
  font-size: 14px;
  font-weight: 600;
}

.generate-form {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.form-select {
  flex: 1;
  min-width: 200px;
  padding: 7px 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: 14px;
  color: var(--color-text);
  background: var(--color-bg-card);
  outline: none;
  transition: border-color var(--transition-fast);
}

.form-select:focus {
  border-color: var(--color-primary);
}

.type-selector {
  display: flex;
  gap: 4px;
}

.type-option {
  padding: 6px 14px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: 13px;
  cursor: pointer;
  color: var(--color-text-secondary);
  background: var(--color-bg-card);
  transition: all var(--transition-fast);
}

.type-option:hover {
  border-color: var(--color-border-strong);
  color: var(--color-text);
}

.type-option.active {
  background: var(--color-primary);
  color: var(--color-text-inverse);
  border-color: var(--color-primary);
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
}

.btn-primary {
  display: inline-flex;
  align-items: center;
  padding: 7px 16px;
  background: var(--color-primary);
  color: var(--color-text-inverse);
  border: 1px solid var(--color-primary);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  white-space: nowrap;
  transition: all var(--transition-fast);
}

.btn-primary:hover:not(:disabled) {
  background: var(--color-primary-hover);
  border-color: var(--color-primary-hover);
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  display: inline-flex;
  align-items: center;
  padding: 7px 14px;
  background: var(--color-bg-card);
  color: var(--color-text);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  transition: all var(--transition-fast);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--color-bg-hover);
  border-color: var(--color-border-strong);
}

.btn-danger {
  display: inline-flex;
  align-items: center;
  padding: 7px 14px;
  background: var(--color-danger);
  color: #ffffff;
  border: 1px solid var(--color-danger);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  transition: all var(--transition-fast);
}

.btn-danger:hover:not(:disabled) {
  opacity: 0.9;
}

.error-banner {
  background: var(--color-danger-bg);
  border-color: var(--color-danger);
  color: var(--color-danger);
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.panel-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}

.panel-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.loading,
.empty {
  text-align: center;
  padding: 24px;
  color: var(--color-text-muted);
  font-size: 14px;
}

.report-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.report-item {
  padding: 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.report-item:hover {
  background: var(--color-bg-hover);
}

.report-item.active {
  border-color: var(--color-primary);
  background: var(--color-primary-bg);
}

.report-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.report-title {
  font-weight: 500;
  font-size: 14px;
  color: var(--color-text);
}

.report-badges {
  display: flex;
  gap: 6px;
}

.type-badge,
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 500;
}

.type-weekly {
  background: var(--color-info-bg);
  color: var(--color-info);
}

.type-monthly {
  background: var(--color-success-bg);
  color: var(--color-success);
}

.type-annual {
  background: var(--color-warning-bg);
  color: var(--color-warning);
}

.status-running {
  background: var(--color-warning-bg);
  color: var(--color-warning);
}

.status-running .spinner-sm {
  display: inline-block;
  width: 10px;
  height: 10px;
  border: 1.5px solid currentColor;
  border-right-color: transparent;
  border-radius: var(--radius-full);
  animation: spin 0.8s linear infinite;
  vertical-align: middle;
  margin-right: 4px;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.status-completed {
  background: var(--color-success-bg);
  color: var(--color-success);
}

.status-failed {
  background: var(--color-danger-bg);
  color: var(--color-danger);
}

.report-meta {
  margin-top: 4px;
  font-size: 12px;
  color: var(--color-text-muted);
}
</style>
