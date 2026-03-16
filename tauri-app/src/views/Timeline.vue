<template>
  <section class="timeline-page">
    <div class="page-header">
      <div>
        <h2>人生时间线</h2>
        <p class="subtitle">关键事件、关系变化和情绪拐点的全景视图</p>
      </div>
      <button class="btn-refresh" @click="reload">刷新</button>
    </div>

    <div v-if="error" class="error-banner">{{ error }}</div>

    <div v-if="loading" class="loading">加载中...</div>

    <div v-if="events.length === 0 && !loading" class="empty">
      暂无时间线数据。导入对话并运行辩论后，这里会展示关键事件。
    </div>

    <!-- Timeline -->
    <div v-if="events.length > 0" class="timeline-container">
      <!-- 左侧月份导航 -->
      <div class="month-nav">
        <button
          v-for="m in months"
          :key="m.key"
          class="month-btn"
          :class="{ active: activeMonth === m.key }"
          @click="scrollToMonth(m.key)"
        >
          <span class="month-label">{{ m.label }}</span>
          <span class="month-count">{{ m.count }}</span>
        </button>
      </div>

      <!-- 右侧时间线 -->
      <div class="timeline-body" ref="timelineBody">
        <template v-for="(group, gi) in groupedEvents" :key="gi">
          <div class="date-anchor" :data-month="group.monthKey"></div>
          <div class="date-header">{{ group.dateLabel }}</div>
          <div class="day-events">
            <div
              v-for="(event, ei) in group.events"
              :key="ei"
              class="event-row"
            >
              <div class="event-time-col">
                <span class="event-time">{{ formatHM(event.time) }}</span>
              </div>
              <div class="event-line-col">
                <span class="event-dot" :class="`dot-${event.type}`"></span>
                <span v-if="ei < group.events.length - 1" class="event-stem"></span>
              </div>
              <div class="event-card" :class="`card-${event.type}`">
                <div class="event-card-header">
                  <span class="event-tag" :class="`tag-${event.type}`">{{ typeLabel(event.type) }}</span>
                  <span v-if="event.priority" class="event-priority" :class="`p-${event.priority}`">{{ priorityLabel(event.priority) }}</span>
                </div>
                <p class="event-text">{{ event.text }}</p>
                <p v-if="event.detail" class="event-detail">{{ event.detail }}</p>
              </div>
            </div>
          </div>
        </template>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from "vue";
import { fetchForumSessions, fetchActionItems, fetchPendingReminders } from "../services/api";

interface TimelineEvent {
  type: "debate" | "debate-running" | "debate-failed" | "action" | "reminder";
  text: string;
  detail?: string;
  time: string;
  priority?: string;
}

const events = ref<TimelineEvent[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);
const activeMonth = ref("");
const timelineBody = ref<HTMLElement | null>(null);

// Group events by date
const groupedEvents = computed(() => {
  const groups: { dateLabel: string; monthKey: string; events: TimelineEvent[] }[] = [];
  let currentDate = "";
  for (const event of events.value) {
    const d = parseDate(event.time);
    const dateStr = d ? `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}` : "未知";
    const monthKey = d ? `${d.getFullYear()}-${pad(d.getMonth() + 1)}` : "未知";
    const label = d ? formatDateLabel(d) : "未知日期";
    if (dateStr !== currentDate) {
      currentDate = dateStr;
      groups.push({ dateLabel: label, monthKey, events: [] });
    }
    groups[groups.length - 1].events.push(event);
  }
  return groups;
});

// Month navigation
const months = computed(() => {
  const map = new Map<string, { label: string; count: number }>();
  for (const g of groupedEvents.value) {
    const existing = map.get(g.monthKey);
    if (existing) {
      existing.count += g.events.length;
    } else {
      const d = parseDate(g.events[0]?.time);
      const label = d ? `${d.getFullYear()}年${d.getMonth() + 1}月` : g.monthKey;
      map.set(g.monthKey, { label, count: g.events.length });
    }
  }
  return Array.from(map.entries()).map(([key, v]) => ({ key, ...v }));
});

function scrollToMonth(key: string) {
  activeMonth.value = key;
  const el = timelineBody.value?.querySelector(`[data-month="${key}"]`);
  el?.scrollIntoView({ behavior: "smooth", block: "start" });
}

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

    for (const s of (Array.isArray(sessions) ? sessions : [])) {
      const debateType: TimelineEvent["type"] =
        s.status === "failed" ? "debate-failed" :
        s.status === "running" ? "debate-running" : "debate";
      all.push({
        type: debateType,
        text: s.status === "failed" ? `辩论失败` :
              s.status === "running" ? `辩论进行中` :
              s.summary || `辩论会话完成`,
        detail: s.summary && s.summary.length > 40 ? s.summary : undefined,
        time: s.completed_at || s.created_at,
      });
    }

    for (const item of (Array.isArray(items) ? items : [])) {
      all.push({
        type: "action",
        text: item.title,
        detail: item.description,
        time: item.created_at,
        priority: item.priority,
      });
    }

    for (const r of (Array.isArray(reminders) ? reminders : [])) {
      all.push({
        type: "reminder",
        text: r.message,
        time: r.triggered_at,
      });
    }

    all.sort((a, b) => (b.time || "").localeCompare(a.time || ""));
    events.value = all;

    await nextTick();
    if (months.value.length > 0) {
      activeMonth.value = months.value[0].key;
    }
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : "加载时间线失败";
  } finally {
    loading.value = false;
  }
}

function parseDate(time: string): Date | null {
  if (!time) return null;
  const d = new Date(time);
  return isNaN(d.getTime()) ? null : d;
}

function pad(n: number): string {
  return n < 10 ? `0${n}` : `${n}`;
}

function formatDateLabel(d: Date): string {
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const target = new Date(d.getFullYear(), d.getMonth(), d.getDate());
  const diff = Math.floor((today.getTime() - target.getTime()) / 86400000);
  const weekdays = ["日", "一", "二", "三", "四", "五", "六"];
  const weekday = `周${weekdays[d.getDay()]}`;
  if (diff === 0) return `今天 · ${weekday}`;
  if (diff === 1) return `昨天 · ${weekday}`;
  return `${d.getMonth() + 1}月${d.getDate()}日 · ${weekday}`;
}

function formatHM(time: string): string {
  const d = parseDate(time);
  if (!d) return "";
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function typeLabel(type: string): string {
  const m: Record<string, string> = {
    debate: "辩论洞察",
    "debate-running": "进行中",
    "debate-failed": "失败",
    action: "行动项",
    reminder: "提醒",
  };
  return m[type] ?? type;
}

function priorityLabel(p: string): string {
  const m: Record<string, string> = { high: "紧急", medium: "一般", low: "低" };
  return m[p] ?? p;
}
</script>

<style scoped>
.timeline-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.page-header h2 { margin: 0; font-size: 20px; font-weight: 600; }
.subtitle { color: var(--color-text-secondary); margin: 4px 0 0; font-size: 13px; }

.btn-refresh {
  padding: 7px 14px;
  background: var(--color-bg-card);
  color: var(--color-text);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
}

.btn-refresh:hover {
  background: var(--color-bg-hover);
}

.error-banner {
  background: var(--color-danger-bg);
  border: 1px solid var(--color-danger);
  border-radius: var(--radius-md);
  padding: 12px 16px;
  color: var(--color-danger);
  font-size: 13px;
}

.loading, .empty {
  text-align: center;
  padding: 40px;
  color: var(--color-text-muted);
  font-size: 14px;
}

/* Layout: left month nav + right timeline */
.timeline-container {
  display: flex;
  gap: 20px;
  align-items: flex-start;
}

/* Month nav */
.month-nav {
  position: sticky;
  top: 16px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 120px;
  flex-shrink: 0;
}

.month-btn {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  border: none;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-text-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.month-btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-text);
}

.month-btn.active {
  background: var(--color-primary-bg);
  color: var(--color-primary);
  font-weight: 600;
}

.month-count {
  font-size: 11px;
  background: var(--color-bg-tertiary);
  padding: 1px 6px;
  border-radius: 10px;
  color: var(--color-text-muted);
}

.month-btn.active .month-count {
  background: var(--color-primary);
  color: var(--color-text-inverse);
}

/* Timeline body */
.timeline-body {
  flex: 1;
  min-width: 0;
}

.date-header {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary);
  padding: 4px 0 8px;
  margin-top: 16px;
  border-bottom: 1px solid var(--color-border);
  margin-bottom: 12px;
}

.date-header:first-of-type {
  margin-top: 0;
}

.day-events {
  display: flex;
  flex-direction: column;
  margin-bottom: 8px;
}

/* Event row: time | dot+stem | card */
.event-row {
  display: flex;
  gap: 0;
  min-height: 60px;
}

.event-time-col {
  width: 50px;
  flex-shrink: 0;
  padding-top: 14px;
  text-align: right;
  padding-right: 12px;
}

.event-time {
  font-size: 12px;
  color: var(--color-text-muted);
  font-variant-numeric: tabular-nums;
}

.event-line-col {
  width: 20px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
}

.event-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  margin-top: 16px;
  flex-shrink: 0;
  z-index: 1;
  box-shadow: 0 0 0 3px var(--color-bg);
}

.dot-debate { background: #7aadff; }
.dot-debate-running { background: #f0b866; }
.dot-debate-failed { background: #f5827a; }
.dot-action { background: #7ee8a8; }
.dot-reminder { background: #c4a0f5; }

.event-stem {
  flex: 1;
  width: 1.5px;
  background: var(--color-border);
  margin-top: 4px;
}

.event-card {
  flex: 1;
  padding: 12px 14px;
  margin-bottom: 6px;
  border-radius: var(--radius-md);
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  transition: all var(--transition-fast);
}

.event-card:hover {
  border-color: var(--color-border-strong);
  box-shadow: var(--shadow-sm);
}

/* Subtle left accent per type */
.card-debate { border-left: 3px solid #7aadff; }
.card-debate-running { border-left: 3px solid #f0b866; }
.card-debate-failed { border-left: 3px solid #f5827a; }
.card-action { border-left: 3px solid #7ee8a8; }
.card-reminder { border-left: 3px solid #c4a0f5; }

.event-card-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}

.event-tag {
  padding: 1px 7px;
  border-radius: 8px;
  font-size: 11px;
  font-weight: 600;
}

.tag-debate { background: rgba(122,173,255,0.15); color: #7aadff; }
.tag-debate-running { background: rgba(240,184,102,0.15); color: #f0b866; }
.tag-debate-failed { background: rgba(245,130,122,0.15); color: #f5827a; }
.tag-action { background: rgba(126,232,168,0.15); color: #52c47e; }
.tag-reminder { background: rgba(196,160,245,0.15); color: #c4a0f5; }

.event-priority {
  font-size: 11px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 8px;
}

.p-high { background: rgba(245,130,122,0.15); color: #f5827a; }
.p-medium { background: rgba(240,184,102,0.15); color: #f0b866; }
.p-low { background: var(--color-bg-tertiary); color: var(--color-text-muted); }

.event-text {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
  color: var(--color-text);
}

.event-detail {
  margin: 6px 0 0;
  font-size: 13px;
  line-height: 1.5;
  color: var(--color-text-secondary);
}

/* Responsive: hide month nav on narrow screens */
@media (max-width: 640px) {
  .month-nav { display: none; }
}
</style>
