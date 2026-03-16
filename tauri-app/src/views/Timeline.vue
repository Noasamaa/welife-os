<template>
  <section class="page">
    <div class="page-header">
      <h2>人生时间线</h2>
      <p class="subtitle">关键事件、关系变化和情绪拐点的全景视图</p>
    </div>

    <div v-if="error" class="card error-banner">{{ error }}</div>

    <div class="card">
      <div class="panel-header">
        <h3>最近动态</h3>
        <button class="btn-secondary" @click="reload">刷新</button>
      </div>

      <div v-if="loading" class="loading">加载中...</div>

      <div v-if="events.length === 0 && !loading" class="empty">
        暂无时间线数据。导入对话并运行辩论后，这里会展示关键事件。
      </div>

      <div class="timeline">
        <div v-for="(event, idx) in events" :key="idx" class="timeline-item">
          <div class="timeline-dot" :class="`dot-${event.type}`"></div>
          <div class="timeline-content">
            <div class="event-header">
              <span class="event-type" :class="`type-${event.type}`">{{ typeLabel(event.type) }}</span>
              <span class="event-time">{{ event.time?.slice(0, 16) }}</span>
            </div>
            <p class="event-text">{{ event.text }}</p>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { fetchForumSessions, fetchActionItems, fetchPendingReminders } from "../services/api";

interface TimelineEvent {
  type: "debate" | "debate-running" | "debate-failed" | "action" | "reminder";
  text: string;
  time: string;
}

const events = ref<TimelineEvent[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

onMounted(() => reload());

async function reload() {
  loading.value = true;
  error.value = null;
  try {
    const [sessions, items, reminders] = await Promise.all([
      fetchForumSessions(),
      fetchActionItems(),
      fetchPendingReminders(),
    ]);

    const all: TimelineEvent[] = [];

    const sessionList = Array.isArray(sessions) ? sessions : [];
    for (const s of sessionList) {
      const debateType: TimelineEvent["type"] =
        s.status === "failed" ? "debate-failed" :
        s.status === "running" ? "debate-running" :
        "debate";
      let text: string;
      if (s.status === "failed") {
        text = `辩论失败 — 会话 ${s.id.slice(0, 16)}`;
      } else if (s.status === "running") {
        text = `辩论进行中 — 会话 ${s.id.slice(0, 16)}`;
      } else {
        text = s.summary || `辩论会话 ${s.id.slice(0, 16)}`;
      }
      all.push({
        type: debateType,
        text,
        time: s.completed_at || s.created_at,
      });
    }

    const itemList = Array.isArray(items) ? items : [];
    for (const item of itemList) {
      all.push({
        type: "action",
        text: `[${item.priority}] ${item.title}`,
        time: item.created_at,
      });
    }

    const reminderList = Array.isArray(reminders) ? reminders : [];
    for (const r of reminderList) {
      all.push({
        type: "reminder",
        text: r.message,
        time: r.triggered_at,
      });
    }

    all.sort((a, b) => (b.time || "").localeCompare(a.time || ""));
    events.value = all;
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : "加载时间线失败";
  } finally {
    loading.value = false;
  }
}

function typeLabel(type: string): string {
  const m: Record<string, string> = {
    debate: "辩论",
    "debate-running": "辩论 (进行中)",
    "debate-failed": "辩论 (失败)",
    action: "行动",
    reminder: "提醒",
  };
  return m[type] ?? type;
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

.loading,
.empty {
  text-align: center;
  padding: 24px;
  color: var(--color-text-muted);
  font-size: 14px;
}

.timeline {
  position: relative;
  padding-left: 20px;
}

.timeline::before {
  content: "";
  position: absolute;
  left: 3px;
  top: 0;
  bottom: 0;
  width: 1px;
  background: var(--color-border);
}

.timeline-item {
  position: relative;
  padding-bottom: 16px;
  display: flex;
  gap: 12px;
}

.timeline-dot {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-full);
  margin-top: 6px;
  flex-shrink: 0;
  position: relative;
  z-index: 1;
}

.dot-debate {
  background: var(--color-info);
}

.dot-debate-running {
  background: var(--color-warning);
}

.dot-debate-failed {
  background: var(--color-danger);
}

.dot-action {
  background: var(--color-success);
}

.dot-reminder {
  background: var(--color-warning);
}

.timeline-content {
  flex: 1;
}

.event-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.event-type {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 500;
}

.type-debate {
  background: var(--color-info-bg);
  color: var(--color-info);
}

.type-debate-running {
  background: var(--color-warning-bg);
  color: var(--color-warning);
}

.type-debate-failed {
  background: var(--color-danger-bg);
  color: var(--color-danger);
}

.type-action {
  background: var(--color-success-bg);
  color: var(--color-success);
}

.type-reminder {
  background: var(--color-warning-bg);
  color: var(--color-warning);
}

.event-time {
  font-size: 12px;
  color: var(--color-text-muted);
}

.event-text {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
  color: var(--color-text);
}
</style>
