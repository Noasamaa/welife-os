<template>
  <section class="page">
    <div class="page-header">
      <h2>执行教练</h2>
      <p class="subtitle">将洞察转化为行动，追踪执行进度</p>
    </div>

    <div v-if="error" class="card error-banner">{{ error }}</div>

    <div class="card generator-panel">
      <h3>生成行动计划</h3>
      <div class="generator-form">
        <select v-model="selectedSession" class="form-select">
          <option value="">选择已完成的辩论会话...</option>
          <option
            v-for="session in completedSessions"
            :key="session.id"
            :value="session.id"
          >
            {{ session.id.slice(0, 16) }}... · {{ session.created_at?.slice(0, 16) }}
          </option>
        </select>
        <button class="btn-secondary" :disabled="!selectedSession || generating" @click="handleGenerate">
          {{ generating ? "生成中..." : "生成行动计划" }}
        </button>
      </div>
      <p v-if="generationMessage" class="status-message">{{ generationMessage }}</p>
    </div>

    <!-- Filter bar -->
    <div class="card filter-bar">
      <select v-model="filterStatus" class="form-select" @change="reload">
        <option value="">全部状态</option>
        <option value="pending">待处理</option>
        <option value="completed">已完成</option>
        <option value="dismissed">已取消</option>
      </select>
      <select v-model="filterCategory" class="form-select" @change="reload">
        <option value="">全部类别</option>
        <option value="relationship">关系维护</option>
        <option value="opportunity">机会跟进</option>
        <option value="risk">风险防范</option>
        <option value="general">通用</option>
      </select>
      <button class="btn-secondary" @click="reload">刷新</button>
    </div>

    <!-- Action items -->
    <div v-if="loading" class="loading">加载中...</div>

    <div v-if="items.length === 0 && !loading" class="card empty">
      暂无行动项。通过辩论论坛生成行动计划。
    </div>

    <div class="items-grid">
      <ActionItemCard
        v-for="item in items"
        :key="item.id"
        :item="item"
        @complete="handleComplete"
        @dismiss="handleDismiss"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from "vue";
import { useCoach } from "../composables/useCoach";
import ActionItemCard from "../components/ActionItemCard.vue";
import { fetchForumSessions } from "../services/api";
import type { ForumSession } from "../types/forum";

const { items, loading, generating, error, loadItems, generate, updateStatus } = useCoach();

const filterStatus = ref("");
const filterCategory = ref("");
const selectedSession = ref("");
const forumSessions = ref<ForumSession[]>([]);
const generationMessage = ref("");
const completedSessions = computed(() => forumSessions.value.filter((session) => session.status === "completed"));

onMounted(async () => {
  await Promise.all([reload(), loadForumSessions()]);
});

function reload() {
  return loadItems(filterStatus.value || undefined, filterCategory.value || undefined);
}

function handleComplete(id: string) { void updateStatus(id, "completed"); }
function handleDismiss(id: string) { void updateStatus(id, "dismissed"); }

async function loadForumSessions() {
  try {
    forumSessions.value = await fetchForumSessions();
  } catch {
    forumSessions.value = [];
  }
}

async function handleGenerate() {
  if (!selectedSession.value) return;
  const result = await generate(selectedSession.value);
  if (!result) return;
  generationMessage.value = `已生成 ${result.count} 条行动项。`;
  window.setTimeout(() => { generationMessage.value = ""; }, 3000);
  await Promise.all([reload(), loadForumSessions()]);
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

.error-banner {
  background: var(--color-danger-bg);
  border-color: var(--color-danger);
  color: var(--color-danger);
}

.generator-panel h3 {
  margin: 0 0 12px;
  font-size: 14px;
  font-weight: 600;
}

.generator-form {
  display: flex;
  gap: 12px;
  align-items: center;
}

.filter-bar {
  display: flex;
  gap: 12px;
  align-items: center;
}

.form-select {
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

.loading,
.empty {
  text-align: center;
  padding: 24px;
  color: var(--color-text-muted);
  font-size: 14px;
}

.items-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}

.status-message {
  margin: 12px 0 0;
  font-size: 13px;
  color: var(--color-text-secondary);
}

@media (max-width: 768px) {
  .generator-form {
    flex-direction: column;
    align-items: stretch;
  }

  .filter-bar {
    flex-wrap: wrap;
  }

  .items-grid {
    grid-template-columns: 1fr;
  }
}
</style>
