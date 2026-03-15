<template>
  <section class="page">
    <div class="page-header">
      <h2>辩论论坛</h2>
      <p class="subtitle">多 Agent 交叉辩论，发现深层洞见</p>
    </div>

    <!-- Start Debate Panel -->
    <div class="card start-panel">
      <h3>发起新辩论</h3>
      <div class="start-form">
        <select v-model="selectedConversation" class="select-conversation">
          <option value="">选择对话...</option>
          <option
            v-for="conv in conversations"
            :key="conv.id"
            :value="conv.id"
          >
            {{ conv.title || conv.id }} ({{ conv.platform }}, {{ conv.message_count }} 条消息)
          </option>
        </select>
        <button
          class="btn-primary"
          :disabled="!selectedConversation || debating"
          @click="handleStartDebate"
        >
          {{ debating ? '辩论进行中...' : '开始辩论' }}
        </button>
      </div>
    </div>

    <!-- Error Display -->
    <div v-if="error" class="card error-banner">
      {{ error }}
    </div>

    <!-- Session List -->
    <div class="card sessions-panel">
      <div class="panel-header">
        <h3>辩论会话</h3>
        <button class="btn-secondary" @click="loadSessions">刷新</button>
      </div>

      <div v-if="loading && !currentSession" class="loading">加载中...</div>

      <div v-if="sessions.length === 0 && !loading" class="empty">
        暂无辩论记录，请先选择对话并发起辩论。
      </div>

      <div class="session-list">
        <div
          v-for="session in sessions"
          :key="session.id"
          class="session-item"
          :class="{ active: currentSession?.session.id === session.id }"
          @click="handleSelectSession(session.id)"
        >
          <div class="session-info">
            <span class="session-id">{{ session.id.slice(0, 16) }}...</span>
            <span class="status-badge" :class="`status-${session.status}`">
              {{ statusLabel(session.status) }}
            </span>
          </div>
          <div class="session-meta">
            <span>{{ session.created_at }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Session Detail -->
    <div v-if="currentSession" class="card detail-panel">
      <div class="panel-header">
        <h3>辩论详情</h3>
        <span class="status-badge" :class="`status-${currentSession.session.status}`">
          {{ statusLabel(currentSession.session.status) }}
        </span>
      </div>

      <!-- Summary -->
      <div v-if="currentSession.session.summary" class="summary-box">
        <h4>共识摘要</h4>
        <p>{{ currentSession.session.summary }}</p>
      </div>

      <!-- Debate Timeline -->
      <DebateTimeline
        v-if="currentSession.messages && currentSession.messages.length > 0"
        :messages="currentSession.messages"
      />

      <div v-else class="empty">辩论进行中，尚无消息记录...</div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useForum } from "../composables/useForum";
import { fetchConversations } from "../services/api";
import type { Conversation } from "../types/import";
import DebateTimeline from "../components/DebateTimeline.vue";

const {
  sessions,
  currentSession,
  loading,
  debating,
  error,
  loadSessions,
  loadSession,
  startDebate,
} = useForum();

const conversations = ref<Conversation[]>([]);
const selectedConversation = ref("");

onMounted(async () => {
  await Promise.all([loadSessions(), loadConversations()]);
});

async function loadConversations() {
  try {
    conversations.value = await fetchConversations();
  } catch (e: any) {
    error.value = e.message ?? "加载对话列表失败";
  }
}

async function handleStartDebate() {
  if (!selectedConversation.value) return;
  await startDebate(selectedConversation.value);
}

async function handleSelectSession(id: string) {
  await loadSession(id);
}

function statusLabel(status: string): string {
  const labels: Record<string, string> = {
    running: "进行中",
    completed: "已完成",
    failed: "失败",
  };
  return labels[status] ?? status;
}
</script>

<style scoped>
.page {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.page-header h2 {
  margin: 0;
}

.subtitle {
  color: var(--color-text-secondary, #666);
  margin: 4px 0 0;
  font-size: 14px;
}

.card {
  background: var(--color-bg-card, #fff);
  border: 1px solid var(--color-border, #e0e0e0);
  border-radius: 10px;
  padding: 20px;
}

.start-panel h3 {
  margin: 0 0 12px;
}

.start-form {
  display: flex;
  gap: 12px;
  align-items: center;
}

.select-conversation {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid var(--color-border, #ddd);
  border-radius: 6px;
  font-size: 14px;
  background: var(--color-bg, #fff);
}

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

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  padding: 6px 14px;
  background: transparent;
  border: 1px solid var(--color-border, #ddd);
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
}

.error-banner {
  background: #fde8e8;
  border-color: #e74c3c;
  color: #c0392b;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.panel-header h3 {
  margin: 0;
}

.loading,
.empty {
  text-align: center;
  padding: 24px;
  color: var(--color-text-secondary, #888);
  font-size: 14px;
}

.session-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.session-item {
  padding: 12px;
  border: 1px solid var(--color-border, #eee);
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s;
}

.session-item:hover {
  background: var(--color-bg-secondary, #f8f8f8);
}

.session-item.active {
  border-color: var(--color-primary, #4a90d9);
  background: #f0f7ff;
}

.session-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.session-id {
  font-family: monospace;
  font-size: 13px;
}

.status-badge {
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 500;
}

.status-running {
  background: #fef3e2;
  color: #e67e22;
}

.status-completed {
  background: #e8f8ef;
  color: #27ae60;
}

.status-failed {
  background: #fde8e8;
  color: #e74c3c;
}

.session-meta {
  margin-top: 4px;
  font-size: 12px;
  color: var(--color-text-secondary, #888);
}

.summary-box {
  background: var(--color-bg-secondary, #f8f9fa);
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 20px;
}

.summary-box h4 {
  margin: 0 0 8px;
  font-size: 14px;
}

.summary-box p {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
}
</style>
