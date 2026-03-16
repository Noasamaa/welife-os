<template>
  <section class="page">
    <div class="page-header">
      <h2>平行人生模拟</h2>
      <p class="subtitle">探索「如果当时…」的平行世界</p>
    </div>

    <div v-if="error" class="card error-banner">{{ error }}</div>

    <div class="card">
      <div class="panel-header">
        <h3>选择对话</h3>
      </div>
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
      <div v-if="!selectedConversation" class="empty selection-tip">
        先选择一个对话，再构建人物画像或启动模拟。
      </div>
    </div>

    <!-- Profiles -->
    <div class="card" v-if="selectedConversation">
      <div class="panel-header">
        <h3>人物画像</h3>
        <button class="btn-secondary" :disabled="building || !selectedConversation" @click="handleBuildProfiles">
          {{ building ? '构建中...' : '构建画像' }}
        </button>
      </div>
      <div v-if="profileStatus" class="status-note">{{ profileStatus }}</div>
      <div v-if="profiles.length === 0 && !building" class="empty">暂无画像，点击「构建画像」从知识图谱生成。</div>
      <div class="profile-gallery">
        <div
          v-for="p in profiles"
          :key="p.id"
          class="persona-card"
          :class="{ expanded: expandedProfile === p.id }"
          @click="expandedProfile = expandedProfile === p.id ? null : p.id"
        >
          <!-- Avatar -->
          <div class="persona-avatar" :style="{ background: avatarColor(p.name) }">
            {{ p.name.slice(0, 1) }}
          </div>
          <div class="persona-info">
            <div class="persona-name">{{ p.name }}</div>
            <div class="persona-relation">{{ p.relationship_to_self?.slice(0, 50) || '未知关系' }}{{ p.relationship_to_self && p.relationship_to_self.length > 50 ? '...' : '' }}</div>
          </div>
          <!-- Expanded detail -->
          <div v-if="expandedProfile === p.id" class="persona-detail" @click.stop>
            <div class="detail-section">
              <div class="detail-label">性格特质</div>
              <p class="detail-text">{{ p.personality }}</p>
            </div>
            <div class="detail-section" v-if="p.relationship_to_self">
              <div class="detail-label">与你的关系</div>
              <p class="detail-text">{{ p.relationship_to_self }}</p>
            </div>
            <div class="detail-section" v-if="p.behavioral_patterns">
              <div class="detail-label">行为模式</div>
              <p class="detail-text">{{ p.behavioral_patterns }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Fork point -->
    <div class="card">
      <h3>设定分叉点</h3>
      <div class="fork-form">
        <input
          v-model="forkDescription"
          class="form-input"
          placeholder="如果当时接受了深圳的 offer..."
        />
        <button
          class="btn-primary"
          :disabled="!selectedConversation || !forkDescription || running"
          @click="handleRun"
        >
          {{ running ? '模拟中...' : '开始模拟' }}
        </button>
      </div>
    </div>

    <!-- Sessions -->
    <div class="card">
      <div class="panel-header">
        <h3>模拟历史</h3>
        <button class="btn-secondary" @click="handleRefreshSessions">刷新</button>
      </div>
      <div v-if="sessions.length === 0 && !loading" class="empty">暂无模拟记录。</div>
      <div class="session-list">
        <div
          v-for="s in sessions"
          :key="s.id"
          class="session-item"
          :class="{ active: currentSession?.session.id === s.id }"
          @click="handleSelectSession(s.id)"
        >
          <div class="session-info">
            <span class="fork-text">{{ s.fork_description }}</span>
            <span class="status-badge" :class="`status-${s.status}`">{{ statusLabel(s.status) }}</span>
          </div>
          <div class="session-meta">{{ s.step_count }} 步 · {{ s.created_at?.slice(0, 16) }}</div>
        </div>
      </div>
    </div>

    <!-- Detail -->
    <div v-if="currentSession" class="card">
      <h3>模拟结果</h3>

      <SimulationGraph
        :original-snapshot="currentSession.session.original_graph_snapshot"
        :final-snapshot="currentSession.session.final_graph_snapshot"
      />

      <div v-if="currentSession.session.narrative" class="narrative-box">
        <h4>平行人生叙事</h4>
        <p>{{ currentSession.session.narrative }}</p>
      </div>

      <div v-if="currentSession.steps && currentSession.steps.length > 0" class="steps">
        <h4>演化步骤</h4>
        <div v-for="step in currentSession.steps" :key="step.id" class="step-card">
          <span class="step-num">第 {{ step.step_number }} 步</span>
          <p>{{ step.description }}</p>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from "vue";
import { useSimulation } from "../composables/useSimulation";
import SimulationGraph from "../components/SimulationGraph.vue";
import { fetchConversations, pollTaskUntilDone } from "../services/api";
import type { Conversation } from "../types/import";

const {
  profiles, sessions, currentSession,
  loading, running, building, error,
  loadProfiles, buildAllProfiles, loadSessions, loadSession, startSimulation,
} = useSimulation();

const conversations = ref<Conversation[]>([]);
const selectedConversation = ref("");
const forkDescription = ref("");
const profileStatus = ref("");
const expandedProfile = ref<string | null>(null);
let cancelProfilePoll: (() => void) | null = null;
let cancelSessionPoll: (() => void) | null = null;

onMounted(() => {
  void loadConversationOptions();
});

onUnmounted(() => {
  cancelProfilePoll?.();
  cancelSessionPoll?.();
});

watch(selectedConversation, (conversationID, oldConversationID) => {
  cancelProfilePoll?.();
  cancelProfilePoll = null;
  if (oldConversationID && oldConversationID !== conversationID) {
    cancelSessionPoll?.();
    cancelSessionPoll = null;
  }
  profileStatus.value = "";
  void Promise.all([
    loadProfiles(conversationID),
    loadSessions(conversationID),
  ]);
});

async function handleRun() {
  if (!selectedConversation.value || !forkDescription.value) return;
  const result = await startSimulation(selectedConversation.value, forkDescription.value, [], {});
  if (!result) return;

  cancelSessionPoll?.();
  const convId = selectedConversation.value;
  const { promise, cancel } = pollTaskUntilDone(result.task_id, () => {
    void loadSession(result.session_id, convId);
  });
  cancelSessionPoll = cancel;

  try {
    const info = await promise;
    if (info.status === "succeeded") {
      await loadSessions(convId);
      await loadSession(result.session_id, convId);
    } else {
      error.value = info.error || "模拟任务失败";
    }
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : "轮询模拟状态失败";
  } finally {
    cancelSessionPoll = null;
  }
}

async function handleBuildProfiles() {
  if (!selectedConversation.value) return;
  const result = await buildAllProfiles(selectedConversation.value);
  if (!result) return;

  profileStatus.value = "画像构建任务已提交，正在后台刷新...";
  cancelProfilePoll?.();
  const convId = selectedConversation.value;
  const { promise, cancel } = pollTaskUntilDone(result.task_id);
  cancelProfilePoll = cancel;

  try {
    const info = await promise;
    if (info.status === "succeeded") {
      await loadProfiles(convId);
      profileStatus.value = "人物画像已刷新完成。";
    } else {
      profileStatus.value = "画像构建失败，可稍后重试。";
      error.value = info.error || "画像构建任务失败";
    }
  } catch (e: unknown) {
    profileStatus.value = "画像仍在后台构建中，可稍后手动刷新。";
    error.value = e instanceof Error ? e.message : "轮询画像构建状态失败";
  } finally {
    cancelProfilePoll = null;
  }
}

function statusLabel(s: string): string {
  const m: Record<string, string> = { running: "运行中", completed: "已完成", failed: "失败" };
  return m[s] ?? s;
}

function parseJson(s: string): Record<string, unknown> {
  try { return JSON.parse(s) as Record<string, unknown>; } catch { return {}; }
}

const AVATAR_COLORS = [
  "linear-gradient(135deg, #7ee8a8, #4ecca3)",
  "linear-gradient(135deg, #7aadff, #5b8def)",
  "linear-gradient(135deg, #f0b866, #e8963a)",
  "linear-gradient(135deg, #c4a0f5, #a67de8)",
  "linear-gradient(135deg, #f5827a, #e85d5d)",
  "linear-gradient(135deg, #6ecfcf, #4db8b8)",
  "linear-gradient(135deg, #f5a0c0, #e87da0)",
];

function avatarColor(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) hash = (hash * 31 + name.charCodeAt(i)) | 0;
  return AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length];
}

async function loadConversationOptions() {
  try {
    conversations.value = await fetchConversations();
    if (!selectedConversation.value && conversations.value.length > 0) {
      selectedConversation.value = conversations.value[0].id;
    }
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : "加载对话失败";
  }
}

async function handleRefreshSessions() {
  if (!selectedConversation.value) return;
  await loadSessions(selectedConversation.value);
}

async function handleSelectSession(id: string) {
  if (!selectedConversation.value) return;
  await loadSession(id, selectedConversation.value);
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

.card h3 {
  margin: 0 0 12px;
  font-size: 14px;
  font-weight: 600;
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
  margin: 0 0 0;
}

.empty {
  text-align: center;
  padding: 16px;
  color: var(--color-text-muted);
  font-size: 14px;
}

.selection-tip {
  padding-top: 12px;
}

.status-note {
  margin-bottom: 12px;
  font-size: 13px;
  color: var(--color-text-secondary);
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

.btn-secondary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.form-select {
  width: 100%;
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

.profile-gallery {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.persona-card {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 14px;
  padding: 14px 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.persona-card:hover {
  border-color: var(--color-border-strong);
  box-shadow: var(--shadow-sm);
}

.persona-card.expanded {
  border-color: var(--color-primary);
  background: var(--color-primary-bg);
}

.persona-avatar {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  font-weight: 700;
  color: #fff;
  flex-shrink: 0;
  text-shadow: 0 1px 2px rgba(0,0,0,0.2);
}

.persona-info {
  flex: 1;
  min-width: 0;
}

.persona-name {
  font-weight: 600;
  font-size: 15px;
  color: var(--color-text);
  margin-bottom: 2px;
}

.persona-relation {
  font-size: 13px;
  color: var(--color-text-secondary);
  line-height: 1.4;
}

.persona-detail {
  width: 100%;
  padding-top: 12px;
  border-top: 1px solid var(--color-border);
  margin-top: 4px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  cursor: default;
}

.detail-section {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.detail-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-muted);
  letter-spacing: 0.3px;
}

.detail-text {
  margin: 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--color-text-secondary);
  white-space: pre-wrap;
}

.fork-form {
  display: flex;
  gap: 12px;
}

.form-input {
  flex: 1;
  padding: 7px 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: 14px;
  color: var(--color-text);
  background: var(--color-bg-card);
  outline: none;
  transition: border-color var(--transition-fast);
}

.form-input::placeholder {
  color: var(--color-text-muted);
}

.form-input:focus {
  border-color: var(--color-primary);
}

.session-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.session-item {
  padding: 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.session-item:hover {
  background: var(--color-bg-hover);
}

.session-item.active {
  border-color: var(--color-primary);
  background: var(--color-primary-bg);
}

.session-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.fork-text {
  font-size: 14px;
  color: var(--color-text);
}

.session-meta {
  font-size: 12px;
  color: var(--color-text-muted);
  margin-top: 4px;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 500;
}

.status-running {
  background: var(--color-info-bg);
  color: var(--color-info);
}

.status-completed {
  background: var(--color-success-bg);
  color: var(--color-success);
}

.status-failed {
  background: var(--color-danger-bg);
  color: var(--color-danger);
}

.narrative-box {
  margin-top: 16px;
  background: var(--color-bg-secondary);
  border-radius: var(--radius-md);
  padding: 16px;
}

.narrative-box h4 {
  margin: 0 0 8px;
  font-size: 14px;
  font-weight: 600;
}

.narrative-box p {
  margin: 0;
  font-size: 14px;
  line-height: 1.7;
  white-space: pre-wrap;
  color: var(--color-text-secondary);
}

.steps {
  margin-top: 16px;
}

.steps h4 {
  margin: 0 0 12px;
  font-size: 14px;
  font-weight: 600;
}

.step-card {
  padding: 10px 12px;
  border-left: 2px solid var(--color-primary);
  margin-bottom: 8px;
  background: var(--color-bg-secondary);
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
}

.step-num {
  font-weight: 600;
  font-size: 13px;
  color: var(--color-primary);
}

.step-card p {
  margin: 4px 0 0;
  font-size: 13px;
  line-height: 1.5;
  color: var(--color-text-secondary);
}

@media (max-width: 768px) {
  .fork-form {
    flex-direction: column;
  }

  .profile-grid {
    grid-template-columns: 1fr;
  }
}
</style>
