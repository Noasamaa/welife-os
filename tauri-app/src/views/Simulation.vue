<template>
  <section class="page">
    <div class="page-header">
      <h2>平行人生模拟</h2>
      <p class="subtitle">探索「如果当时…」的平行世界</p>
    </div>

    <div v-if="error" class="card error-banner">{{ error }}</div>

    <!-- Profiles -->
    <div class="card">
      <div class="panel-header">
        <h3>人物画像</h3>
        <button class="btn-secondary" :disabled="building" @click="handleBuildProfiles">
          {{ building ? '构建中...' : '构建画像' }}
        </button>
      </div>
      <div v-if="profileStatus" class="status-note">{{ profileStatus }}</div>
      <div v-if="profiles.length === 0" class="empty">暂无画像，点击「构建画像」从知识图谱生成。</div>
      <div class="profile-grid">
        <div v-for="p in profiles" :key="p.id" class="profile-card">
          <div class="profile-name">{{ p.name }}</div>
          <div class="profile-detail">{{ parseJson(p.personality).traits || p.personality }}</div>
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
          :disabled="!forkDescription || running"
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
        <button class="btn-secondary" @click="loadSessions">刷新</button>
      </div>
      <div v-if="sessions.length === 0 && !loading" class="empty">暂无模拟记录。</div>
      <div class="session-list">
        <div
          v-for="s in sessions"
          :key="s.id"
          class="session-item"
          :class="{ active: currentSession?.session.id === s.id }"
          @click="loadSession(s.id)"
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

const {
  profiles, sessions, currentSession,
  loading, running, building, error,
  loadProfiles, buildAllProfiles, loadSessions, loadSession, startSimulation,
} = useSimulation();

const forkDescription = ref("");
const profileStatus = ref("");
let profilePollHandle: ReturnType<typeof setInterval> | null = null;
let sessionPollHandle: ReturnType<typeof setInterval> | null = null;

onMounted(async () => {
  await Promise.all([loadProfiles(), loadSessions()]);
});

onUnmounted(() => {
  stopProfilePolling();
  stopSessionPolling();
});

watch(
  () => currentSession.value?.session.status,
  (status) => {
    stopSessionPolling();
    if (status !== "running" || !currentSession.value) {
      return;
    }
    sessionPollHandle = setInterval(() => {
      if (!currentSession.value) {
        stopSessionPolling();
        return;
      }
      void Promise.all([loadSessions(), loadSession(currentSession.value.session.id)]);
    }, 2000);
  },
);

async function handleRun() {
  if (!forkDescription.value) return;
  await startSimulation(forkDescription.value, [], {});
}

async function handleBuildProfiles() {
  const result = await buildAllProfiles();
  if (!result) return;

  profileStatus.value = "画像构建任务已提交，正在后台刷新...";
  startProfilePolling();
}

function statusLabel(s: string): string {
  const m: Record<string, string> = { running: "运行中", completed: "已完成", failed: "失败" };
  return m[s] ?? s;
}

function parseJson(s: string): any {
  try { return JSON.parse(s); } catch { return {}; }
}

function startProfilePolling() {
  stopProfilePolling();
  let attempts = 0;
  profilePollHandle = setInterval(async () => {
    attempts += 1;
    await loadProfiles();
    if (profiles.value.length > 0 || attempts >= 10) {
      profileStatus.value = profiles.value.length > 0
        ? "人物画像已刷新完成。"
        : "画像仍在后台构建中，可稍后手动刷新。";
      stopProfilePolling();
    }
  }, 2000);
}

function stopProfilePolling() {
  if (profilePollHandle !== null) {
    clearInterval(profilePollHandle);
    profilePollHandle = null;
  }
}

function stopSessionPolling() {
  if (sessionPollHandle !== null) {
    clearInterval(sessionPollHandle);
    sessionPollHandle = null;
  }
}
</script>

<style scoped>
.page { padding: 24px; display: flex; flex-direction: column; gap: 20px; }
.page-header h2 { margin: 0; }
.subtitle { color: var(--color-text-secondary, #666); margin: 4px 0 0; font-size: 14px; }
.card { background: var(--color-bg-card, #fff); border: 1px solid var(--color-border, #e0e0e0); border-radius: 10px; padding: 20px; }
.error-banner { background: #fde8e8; border-color: #e74c3c; color: #c0392b; }
.panel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.panel-header h3 { margin: 0; }
.empty { text-align: center; padding: 16px; color: var(--color-text-secondary, #888); font-size: 14px; }
.status-note { margin-bottom: 12px; font-size: 13px; color: var(--color-text-secondary, #666); }
.btn-primary { padding: 8px 20px; background: var(--color-primary, #4a90d9); color: white; border: none; border-radius: 6px; cursor: pointer; font-size: 14px; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-secondary { padding: 6px 14px; background: transparent; border: 1px solid var(--color-border, #ddd); border-radius: 6px; cursor: pointer; font-size: 13px; }

.profile-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 10px; }
.profile-card { border: 1px solid #eee; border-radius: 8px; padding: 12px; }
.profile-name { font-weight: 600; font-size: 14px; margin-bottom: 4px; }
.profile-detail { font-size: 12px; color: #888; }

.fork-form { display: flex; gap: 12px; }
.form-input { flex: 1; padding: 8px 12px; border: 1px solid var(--color-border, #ddd); border-radius: 6px; font-size: 14px; }

.session-list { display: flex; flex-direction: column; gap: 8px; }
.session-item { padding: 12px; border: 1px solid #eee; border-radius: 6px; cursor: pointer; }
.session-item:hover { background: #f8f8f8; }
.session-item.active { border-color: var(--color-primary, #4a90d9); background: #f0f7ff; }
.session-info { display: flex; justify-content: space-between; align-items: center; }
.fork-text { font-size: 14px; }
.session-meta { font-size: 12px; color: #888; margin-top: 4px; }
.status-badge { padding: 2px 8px; border-radius: 10px; font-size: 11px; font-weight: 500; }
.status-running { background: #e8f4fd; color: #2980b9; }
.status-completed { background: #e8f8ef; color: #27ae60; }
.status-failed { background: #fde8e8; color: #e74c3c; }

.narrative-box { margin-top: 16px; background: #f8f9fa; border-radius: 8px; padding: 16px; }
.narrative-box h4 { margin: 0 0 8px; font-size: 14px; }
.narrative-box p { margin: 0; font-size: 14px; line-height: 1.7; white-space: pre-wrap; }

.steps { margin-top: 16px; }
.steps h4 { margin: 0 0 12px; }
.step-card { padding: 10px 12px; border-left: 3px solid var(--color-primary, #4a90d9); margin-bottom: 8px; }
.step-num { font-weight: 600; font-size: 13px; color: var(--color-primary, #4a90d9); }
.step-card p { margin: 4px 0 0; font-size: 13px; line-height: 1.5; }
</style>
