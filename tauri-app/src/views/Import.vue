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
        <select
          v-if="conversations.length"
          v-model="selectedGraphConversation"
        >
          <option value="" disabled>选择对话</option>
          <option
            v-for="c in conversations"
            :key="c.id"
            :value="c.id"
          >
            {{ c.title || c.id }}
          </option>
        </select>
        <button
          v-if="conversations.length"
          class="btn"
          :disabled="graphBuilding || !selectedGraphConversation"
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
import { fetchConversations, pollTaskUntilDone } from "../services/api";
import type { Conversation } from "../types/import";

const { jobs, uploading, error: importError, upload, refreshJobs } = useImport();
const { overview, loading: graphLoading, error: graphError, loadOverview, buildGraph } = useGraph();
const conversations = ref<Conversation[]>([]);
const selectedGraphConversation = ref("");
const graphStatus = ref("");
const graphBuilding = ref(false);
let cancelPoll: (() => void) | null = null;

async function onFile(file: File) {
  await upload(file);
  await loadConversations();
}

async function onBuildGraph() {
  if (graphBuilding.value) return;
  if (!selectedGraphConversation.value) return;
  graphBuilding.value = true;
  const result = await buildGraph(selectedGraphConversation.value);
  if (!result) {
    graphBuilding.value = false;
    return;
  }
  graphStatus.value = "图谱构建任务已提交，正在后台处理...";

  const taskId = result.task_id;
  const { promise, cancel } = pollTaskUntilDone(taskId, (info) => {
    if (info.status === "running") {
      graphStatus.value = "图谱构建中...";
    }
  });
  cancelPoll = cancel;

  try {
    const finalInfo = await promise;
    if (finalInfo.status === "succeeded") {
      graphStatus.value = "图谱构建完成！";
      await loadOverview();
    } else if (finalInfo.status === "failed") {
      graphStatus.value = `图谱构建失败: ${finalInfo.error || "未知错误"}`;
    } else {
      graphStatus.value = "图谱仍在后台构建中，可稍后手动刷新。";
    }
  } catch {
    graphStatus.value = "图谱构建状态查询失败，可稍后手动刷新。";
  } finally {
    graphBuilding.value = false;
    cancelPoll = null;
  }
}

async function loadConversations() {
  try {
    conversations.value = await fetchConversations();
    if (conversations.value.length > 0 && !selectedGraphConversation.value) {
      selectedGraphConversation.value = conversations.value[0].id;
    }
  } catch (e: unknown) {
    importError.value = e instanceof Error ? e.message : "加载对话列表失败";
  }
}

onMounted(async () => {
  await refreshJobs();
  await loadConversations();
  await loadOverview();
});

onUnmounted(() => {
  cancelPoll?.();
});
</script>

<style scoped>
.import-page {
  display: flex;
  flex-direction: column;
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
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 7px 14px;
  font-size: 13px;
  font-weight: 500;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: var(--color-primary);
  color: var(--color-text-inverse);
  cursor: pointer;
  transition: all var(--transition-fast);
  white-space: nowrap;
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
  font-size: 13px;
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
  padding: 10px 14px;
  background: var(--color-info-bg);
  border-radius: var(--radius-md);
  font-size: 13px;
  font-weight: 500;
  color: var(--color-info);
}

.spinner {
  display: inline-block;
  width: 14px;
  height: 14px;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: var(--radius-full);
  animation: spin 0.8s linear infinite;
  flex-shrink: 0;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
