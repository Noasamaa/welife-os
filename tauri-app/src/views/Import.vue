<template>
  <div class="import-page">
    <section class="card block">
      <h2>导入聊天记录</h2>
      <DropZone accept=".csv,.json,.txt,.db,.sqlite,.sqlite3" @file="onFile" />
      <p v-if="uploading" class="status-msg">上传中...</p>
      <p v-if="importError" class="status-msg error">{{ importError }}</p>
      <div v-if="graphBuilding" class="graph-building-banner">
        <span class="spinner"></span>
        <span>{{ graphStatus || '图谱构建中...' }}</span>
      </div>
      <p v-else-if="graphStatus" class="status-msg">{{ graphStatus }}</p>
    </section>

    <section class="card block">
      <h2>导入记录</h2>
      <ImportJobList :jobs="jobs" />
    </section>

    <section class="card block">
      <div class="graph-header">
        <h2>知识图谱</h2>
        <button
          v-if="conversations.length"
          class="btn"
          :disabled="graphBuilding"
          @click="onBuildGraph"
        >
          {{ graphBuilding ? "构建中..." : "构建图谱" }}
        </button>
      </div>
      <GraphView
        :overview="overview"
        :loading="graphLoading"
        :error="graphError"
      />
    </section>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from "vue";
import DropZone from "../components/DropZone.vue";
import ImportJobList from "../components/ImportJobList.vue";
import GraphView from "../components/GraphView.vue";
import { useImport } from "../composables/useImport";
import { useGraph } from "../composables/useGraph";
import { fetchConversations } from "../services/api";
import type { Conversation } from "../types/import";

const { jobs, uploading, error: importError, upload, refreshJobs } = useImport();
const { overview, loading: graphLoading, error: graphError, loadOverview, buildGraph } = useGraph();
const conversations = ref<Conversation[]>([]);
const graphStatus = ref("");
const graphBuilding = ref(false);
let graphPollHandle: ReturnType<typeof setInterval> | null = null;

async function onFile(file: File) {
  await upload(file);
  await loadConversations();
}

async function onBuildGraph() {
  if (graphBuilding.value) return;
  if (conversations.value.length === 0) return;
  graphBuilding.value = true;
  const result = await buildGraph(conversations.value[0].id);
  if (!result) {
    graphBuilding.value = false;
    return;
  }
  graphStatus.value = "图谱构建任务已提交，正在后台处理...";
  startGraphPolling();
}

async function loadConversations() {
  try {
    conversations.value = await fetchConversations();
  } catch {
    // ignore
  }
}

onMounted(async () => {
  await refreshJobs();
  await loadConversations();
  await loadOverview();
});

onUnmounted(() => {
  stopGraphPolling();
});

function startGraphPolling() {
  stopGraphPolling();
  let attempts = 0;
  graphPollHandle = setInterval(async () => {
    attempts += 1;
    const prevCount = overview.value?.stats?.entity_count ?? 0;
    await loadOverview();
    const newCount = overview.value?.stats?.entity_count ?? 0;

    if (newCount > prevCount) {
      graphStatus.value = `图谱构建中...已生成 ${newCount} 个实体`;
    }

    if (attempts >= 30) {
      graphStatus.value = "图谱仍在后台构建中，可稍后手动刷新。";
      graphBuilding.value = false;
      stopGraphPolling();
    } else if (newCount > 0 && newCount === prevCount && attempts >= 3) {
      graphStatus.value = "图谱构建完成！";
      graphBuilding.value = false;
      stopGraphPolling();
    }
  }, 2000);
}

function stopGraphPolling() {
  if (graphPollHandle !== null) {
    clearInterval(graphPollHandle);
    graphPollHandle = null;
  }
}
</script>

<style scoped>
.import-page {
  display: grid;
  gap: 20px;
}

.block {
  padding: 24px;
}

.graph-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.graph-header h2 {
  margin: 0;
}

.btn {
  padding: 6px 16px;
  border: none;
  border-radius: 6px;
  background: var(--color-primary);
  color: #fff;
  font-weight: 600;
  cursor: pointer;
  font-size: 13px;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn:hover:not(:disabled) {
  background: var(--color-primary-hover);
}

.status-msg {
  margin-top: 12px;
  font-size: 14px;
  color: var(--color-text-secondary);
}

.status-msg.error {
  color: var(--color-danger);
}

.graph-building-banner {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 12px;
  padding: 12px 16px;
  background: var(--color-info-bg, #e8f4fd);
  border: 1px solid var(--color-info, #4a90d9);
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  color: var(--color-info, #2980b9);
}

.spinner {
  display: inline-block;
  width: 16px;
  height: 16px;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  flex-shrink: 0;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
