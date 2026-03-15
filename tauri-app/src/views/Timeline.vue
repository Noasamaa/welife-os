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
.page { padding: 24px; display: flex; flex-direction: column; gap: 20px; }
.page-header h2 { margin: 0; }
.subtitle { color: var(--color-text-secondary, #666); margin: 4px 0 0; font-size: 14px; }
.card { background: var(--color-bg-card, #fff); border: 1px solid var(--color-border, #e0e0e0); border-radius: 10px; padding: 20px; }
.error-banner { background: #fde8e8; border-color: #e74c3c; color: #c0392b; }
.panel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.panel-header h3 { margin: 0; }
.btn-secondary { padding: 6px 14px; background: transparent; border: 1px solid var(--color-border, #ddd); border-radius: 6px; cursor: pointer; font-size: 13px; }
.loading, .empty { text-align: center; padding: 24px; color: var(--color-text-secondary, #888); font-size: 14px; }

.timeline { position: relative; padding-left: 24px; }
.timeline::before {
  content: '';
  position: absolute;
  left: 8px;
  top: 0;
  bottom: 0;
  width: 2px;
  background: var(--color-border, #e0e0e0);
}

.timeline-item { position: relative; padding-bottom: 16px; display: flex; gap: 12px; }

.timeline-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  margin-top: 4px;
  flex-shrink: 0;
  position: relative;
  z-index: 1;
}
.dot-debate { background: #4a90d9; }
.dot-debate-running { background: #f39c12; }
.dot-debate-failed { background: #e74c3c; }
.dot-action { background: #27ae60; }
.dot-reminder { background: #f39c12; }

.timeline-content { flex: 1; }

.event-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px; }
.event-type { padding: 1px 8px; border-radius: 10px; font-size: 11px; font-weight: 500; }
.type-debate { background: #e8f4fd; color: #2980b9; }
.type-debate-running { background: #fef3e2; color: #e67e22; }
.type-debate-failed { background: #fde8e8; color: #c0392b; }
.type-action { background: #e8f8ef; color: #27ae60; }
.type-reminder { background: #fef3e2; color: #e67e22; }

.event-time { font-size: 11px; color: var(--color-text-secondary, #888); }
.event-text { margin: 0; font-size: 13px; line-height: 1.5; color: var(--color-text, #333); }
</style>
