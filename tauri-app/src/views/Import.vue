<template>
  <div class="import-page">
    <section class="card block">
      <h2>导入聊天记录</h2>
      <DropZone accept=".csv,.json,.txt,.db,.sqlite,.sqlite3" @file="onFile" />
      <p v-if="importState.uploading.value" class="status-msg">上传中...</p>
      <p v-if="importState.error.value" class="status-msg error">{{ importState.error.value }}</p>
      <p v-if="graphStatus" class="status-msg">{{ graphStatus }}</p>
    </section>

    <section class="card block">
      <h2>导入记录</h2>
      <ImportJobList :jobs="importState.jobs.value" />
    </section>

    <section class="card block">
      <div class="graph-header">
        <h2>知识图谱</h2>
        <button
          v-if="conversations.length"
          class="btn"
          :disabled="graphState.building.value"
          @click="onBuildGraph"
        >
          {{ graphState.building.value ? "构建中..." : "构建图谱" }}
        </button>
      </div>
      <GraphView
        :overview="graphState.overview.value"
        :loading="graphState.loading.value"
        :error="graphState.error.value"
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

const importState = useImport();
const graphState = useGraph();
const conversations = ref<Conversation[]>([]);
const graphStatus = ref("");
let graphPollHandle: ReturnType<typeof setInterval> | null = null;

async function onFile(file: File) {
  await importState.upload(file);
  await loadConversations();
}

async function onBuildGraph() {
  if (conversations.value.length === 0) return;
  const result = await graphState.buildGraph(conversations.value[0].id);
  if (!result) return;
  graphStatus.value = "图谱构建任务已提交，正在后台刷新结果...";
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
  await importState.refreshJobs();
  await loadConversations();
  await graphState.loadOverview();
});

onUnmounted(() => {
  stopGraphPolling();
});

function startGraphPolling() {
  stopGraphPolling();
  let attempts = 0;
  graphPollHandle = setInterval(async () => {
    attempts += 1;
    await graphState.loadOverview();
    if (attempts >= 10) {
      graphStatus.value = "图谱仍在后台构建中，可稍后手动刷新。";
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
  background: #2d6a4f;
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
  background: #1b4332;
}

.status-msg {
  margin-top: 12px;
  font-size: 14px;
  color: #48625c;
}

.status-msg.error {
  color: #c0392b;
}
</style>
