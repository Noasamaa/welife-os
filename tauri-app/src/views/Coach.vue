<template>
  <section class="page">
    <div class="page-header">
      <div>
        <h2>执行教练</h2>
        <p class="subtitle">将洞察转化为行动，追踪执行进度</p>
      </div>
    </div>

    <div v-if="error" class="error-banner">{{ error }}</div>

    <!-- Generator -->
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
            {{ session.summary?.slice(0, 40) || session.id.slice(0, 16) }}... · {{ session.created_at?.slice(0, 10) }}
          </option>
        </select>
        <button class="btn-primary" :disabled="!selectedSession || generating" @click="handleGenerate">
          {{ generating ? "生成中..." : "生成行动计划" }}
        </button>
      </div>
      <p v-if="generationMessage" class="status-message">{{ generationMessage }}</p>
    </div>

    <!-- Progress bar -->
    <div v-if="items.length > 0" class="progress-section">
      <div class="progress-stats">
        <span class="stat-item">
          <span class="stat-dot dot-pending"></span>
          待处理 <strong>{{ stats.pending }}</strong>
        </span>
        <span class="stat-item">
          <span class="stat-dot dot-completed"></span>
          已完成 <strong>{{ stats.completed }}</strong>
        </span>
        <span class="stat-item">
          <span class="stat-dot dot-dismissed"></span>
          已取消 <strong>{{ stats.dismissed }}</strong>
        </span>
      </div>
      <div class="progress-bar">
        <div class="bar-completed" :style="{ width: progressPercent + '%' }"></div>
      </div>
      <div class="progress-label">执行率 {{ progressPercent }}%</div>
    </div>

    <!-- Filter chips -->
    <div class="filter-row">
      <button
        v-for="f in statusFilters"
        :key="f.key"
        class="filter-chip"
        :class="{ active: filterStatus === f.key }"
        @click="filterStatus = f.key; reload()"
      >
        {{ f.label }}
        <span v-if="f.count > 0" class="chip-count">{{ f.count }}</span>
      </button>
      <span class="filter-divider"></span>
      <button
        v-for="f in categoryFilters"
        :key="f.key"
        class="filter-chip small"
        :class="{ active: filterCategory === f.key }"
        @click="filterCategory = filterCategory === f.key ? '' : f.key; reload()"
      >
        {{ f.label }}
      </button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading">加载中...</div>

    <!-- Empty -->
    <div v-if="items.length === 0 && !loading" class="empty">
      暂无行动项。通过辩论论坛生成行动计划。
    </div>

    <!-- Action items list -->
    <div class="items-list">
      <div
        v-for="item in items"
        :key="item.id"
        class="action-row"
        :class="[`priority-${item.priority}`, `status-${item.status}`]"
      >
        <!-- Checkbox / status -->
        <div class="row-check">
          <button
            v-if="item.status === 'pending'"
            class="check-btn"
            :class="`check-${item.priority}`"
            @click="handleComplete(item.id)"
            title="标记完成"
          >
            <span class="check-ring"></span>
          </button>
          <span v-else-if="item.status === 'completed'" class="check-done">✓</span>
          <span v-else class="check-dismissed">—</span>
        </div>

        <!-- Content -->
        <div class="row-body">
          <div class="row-header">
            <span class="row-title" :class="{ 'title-done': item.status !== 'pending' }">{{ item.title }}</span>
            <div class="row-tags">
              <span class="tag-category">{{ categoryLabel(item.category) }}</span>
              <span class="tag-priority" :class="`tp-${item.priority}`">{{ priorityLabel(item.priority) }}</span>
            </div>
          </div>
          <p v-if="item.description" class="row-desc">{{ item.description }}</p>
          <div class="row-meta">
            <span v-if="item.due_date" class="meta-due">截止 {{ item.due_date.slice(0, 10) }}</span>
            <span class="meta-time">{{ relativeTime(item.created_at) }}</span>
          </div>
        </div>

        <!-- Actions -->
        <div class="row-actions" v-if="item.status === 'pending'">
          <button class="act-btn act-dismiss" @click="handleDismiss(item.id)" title="取消">✕</button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from "vue";
import { useCoach } from "../composables/useCoach";
import { fetchForumSessions } from "../services/api";
import type { ForumSession } from "../types/forum";

const { items, loading, generating, error, loadItems, generate, updateStatus } = useCoach();

const filterStatus = ref("");
const filterCategory = ref("");
const selectedSession = ref("");
const forumSessions = ref<ForumSession[]>([]);
const generationMessage = ref("");
const completedSessions = computed(() => forumSessions.value.filter((s) => s.status === "completed"));

const stats = computed(() => {
  const all = items.value;
  return {
    pending: all.filter((i) => i.status === "pending").length,
    completed: all.filter((i) => i.status === "completed").length,
    dismissed: all.filter((i) => i.status === "dismissed").length,
    total: all.length,
  };
});

const progressPercent = computed(() => {
  const total = stats.value.pending + stats.value.completed;
  if (total === 0) return 0;
  return Math.round((stats.value.completed / total) * 100);
});

const statusFilters = computed(() => [
  { key: "", label: "全部", count: stats.value.total },
  { key: "pending", label: "待处理", count: stats.value.pending },
  { key: "completed", label: "已完成", count: stats.value.completed },
  { key: "dismissed", label: "已取消", count: stats.value.dismissed },
]);

const categoryFilters = [
  { key: "project", label: "项目" },
  { key: "contact", label: "联系人" },
  { key: "decision", label: "决策" },
  { key: "followup", label: "跟进" },
  { key: "general", label: "通用" },
];

const CATEGORY_LABELS: Record<string, string> = {
  project: "项目", contact: "联系人", decision: "决策",
  followup: "跟进", general: "通用",
};
const PRIORITY_LABELS: Record<string, string> = { high: "紧急", medium: "一般", low: "低" };

function categoryLabel(c: string): string { return CATEGORY_LABELS[c] ?? c; }
function priorityLabel(p: string): string { return PRIORITY_LABELS[p] ?? p; }

function relativeTime(time: string): string {
  if (!time) return "";
  const d = new Date(time);
  if (isNaN(d.getTime())) return "";
  const diff = Math.floor((Date.now() - d.getTime()) / 60000);
  if (diff < 1) return "刚刚";
  if (diff < 60) return `${diff}分钟前`;
  const h = Math.floor(diff / 60);
  if (h < 24) return `${h}小时前`;
  const days = Math.floor(h / 24);
  if (days < 7) return `${days}天前`;
  return `${d.getMonth() + 1}/${d.getDate()}`;
}

onMounted(async () => {
  await Promise.all([reload(), loadForumSessions()]);
});

function reload() {
  return loadItems(filterStatus.value || undefined, filterCategory.value || undefined);
}

function handleComplete(id: string) { void updateStatus(id, "completed").then(reload); }
function handleDismiss(id: string) { void updateStatus(id, "dismissed").then(reload); }

async function loadForumSessions() {
  try { forumSessions.value = await fetchForumSessions(); } catch { forumSessions.value = []; }
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

.page-header { display: flex; justify-content: space-between; align-items: flex-start; }
.page-header h2 { margin: 0; font-size: 20px; font-weight: 600; }
.subtitle { color: var(--color-text-secondary); margin: 4px 0 0; font-size: 13px; }

.error-banner {
  background: var(--color-danger-bg); border: 1px solid var(--color-danger);
  border-radius: var(--radius-md); padding: 12px 16px; color: var(--color-danger); font-size: 13px;
}

.card {
  background: var(--color-bg-card); border: 1px solid var(--color-border);
  border-radius: var(--radius-lg); padding: 20px; box-shadow: var(--shadow-sm);
}

.generator-panel h3 { margin: 0 0 12px; font-size: 14px; font-weight: 600; }
.generator-form { display: flex; gap: 12px; align-items: center; flex-wrap: wrap; }

.form-select {
  flex: 1; min-width: 200px; padding: 7px 12px;
  border: 1px solid var(--color-border); border-radius: var(--radius-md);
  font-size: 13px; color: var(--color-text); background: var(--color-bg-card); outline: none;
}
.form-select:focus { border-color: var(--color-primary); }

.btn-primary {
  padding: 7px 16px; background: var(--color-primary); color: var(--color-text-inverse);
  border: 1px solid var(--color-primary); border-radius: var(--radius-md);
  cursor: pointer; font-size: 13px; font-weight: 500; white-space: nowrap;
}
.btn-primary:hover:not(:disabled) { background: var(--color-primary-hover); }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

.status-message { margin: 10px 0 0; font-size: 13px; color: var(--color-text-secondary); }

/* Progress */
.progress-section {
  background: var(--color-bg-card); border: 1px solid var(--color-border);
  border-radius: var(--radius-lg); padding: 16px 20px; box-shadow: var(--shadow-sm);
}

.progress-stats { display: flex; gap: 20px; margin-bottom: 10px; font-size: 13px; color: var(--color-text-secondary); }
.stat-item { display: flex; align-items: center; gap: 6px; }
.stat-item strong { color: var(--color-text); font-weight: 600; }
.stat-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.dot-pending { background: var(--color-warning); }
.dot-completed { background: var(--color-success); }
.dot-dismissed { background: var(--color-text-muted); }

.progress-bar {
  height: 6px; background: var(--color-bg-tertiary); border-radius: 3px; overflow: hidden;
}
.bar-completed {
  height: 100%; background: var(--color-success); border-radius: 3px;
  transition: width 0.4s ease;
}
.progress-label { margin-top: 6px; font-size: 12px; color: var(--color-text-muted); }

/* Filters */
.filter-row { display: flex; gap: 6px; align-items: center; flex-wrap: wrap; }

.filter-chip {
  padding: 5px 12px; border: 1px solid var(--color-border); border-radius: 16px;
  background: var(--color-bg-card); color: var(--color-text-secondary);
  font-size: 13px; cursor: pointer; display: flex; align-items: center; gap: 4px;
  transition: all var(--transition-fast);
}
.filter-chip:hover { border-color: var(--color-border-strong); color: var(--color-text); }
.filter-chip.active { background: var(--color-primary); color: var(--color-text-inverse); border-color: var(--color-primary); }
.filter-chip.small { font-size: 12px; padding: 4px 10px; }
.chip-count {
  font-size: 11px; background: rgba(0,0,0,0.08); padding: 0 5px; border-radius: 8px;
  font-weight: 600; min-width: 16px; text-align: center;
}
.filter-chip.active .chip-count { background: rgba(255,255,255,0.25); }
.filter-divider { width: 1px; height: 20px; background: var(--color-border); margin: 0 4px; }

.loading, .empty {
  text-align: center; padding: 40px; color: var(--color-text-muted); font-size: 14px;
}

/* Action items list */
.items-list { display: flex; flex-direction: column; gap: 4px; }

.action-row {
  display: flex; align-items: flex-start; gap: 12px;
  padding: 12px 16px; border-radius: var(--radius-md);
  background: var(--color-bg-card); border: 1px solid var(--color-border);
  transition: all var(--transition-fast);
}
.action-row:hover { border-color: var(--color-border-strong); box-shadow: var(--shadow-sm); }
.action-row.status-completed { opacity: 0.6; }
.action-row.status-dismissed { opacity: 0.4; }

/* Checkbox */
.row-check { padding-top: 2px; flex-shrink: 0; width: 24px; }

.check-btn {
  width: 20px; height: 20px; border: none; background: none; cursor: pointer; padding: 0;
  display: flex; align-items: center; justify-content: center;
}
.check-ring {
  width: 18px; height: 18px; border-radius: 50%;
  border: 2px solid var(--color-border-strong); transition: all var(--transition-fast);
}
.check-btn:hover .check-ring { border-color: var(--color-success); background: rgba(126,232,168,0.15); }
.check-high .check-ring { border-color: var(--color-danger); }
.check-medium .check-ring { border-color: var(--color-warning); }
.check-low .check-ring { border-color: var(--color-text-muted); }

.check-done { color: var(--color-success); font-size: 16px; font-weight: 700; }
.check-dismissed { color: var(--color-text-muted); font-size: 16px; }

/* Body */
.row-body { flex: 1; min-width: 0; }
.row-header { display: flex; justify-content: space-between; align-items: center; gap: 8px; flex-wrap: wrap; }
.row-title { font-size: 14px; font-weight: 500; color: var(--color-text); }
.title-done { text-decoration: line-through; color: var(--color-text-muted); }

.row-tags { display: flex; gap: 4px; flex-shrink: 0; }
.tag-category, .tag-priority {
  padding: 1px 7px; border-radius: 8px; font-size: 11px; font-weight: 500;
}
.tag-category { background: var(--color-info-bg); color: var(--color-info); }
.tp-high { background: rgba(245,130,122,0.15); color: #e85d5d; }
.tp-medium { background: rgba(240,184,102,0.15); color: #d4902e; }
.tp-low { background: var(--color-bg-tertiary); color: var(--color-text-muted); }

.row-desc {
  margin: 4px 0 0; font-size: 13px; line-height: 1.5;
  color: var(--color-text-secondary);
}

.row-meta { display: flex; gap: 12px; margin-top: 4px; font-size: 12px; color: var(--color-text-muted); }
.meta-due { color: var(--color-warning); font-weight: 500; }

/* Dismiss button */
.row-actions { flex-shrink: 0; padding-top: 2px; }
.act-btn {
  width: 24px; height: 24px; border: none; border-radius: var(--radius-sm);
  background: transparent; color: var(--color-text-muted); cursor: pointer;
  font-size: 14px; display: flex; align-items: center; justify-content: center;
  transition: all var(--transition-fast);
}
.act-dismiss:hover { background: var(--color-danger-bg); color: var(--color-danger); }

@media (max-width: 768px) {
  .generator-form { flex-direction: column; align-items: stretch; }
  .filter-row { gap: 4px; }
  .progress-stats { flex-wrap: wrap; gap: 10px; }
}
</style>
