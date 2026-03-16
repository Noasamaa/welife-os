<template>
  <div class="debate-timeline">
    <div v-for="round in rounds" :key="round" class="round-group">
      <div class="round-header">
        <span class="round-badge">第 {{ round }} 轮</span>
        <span class="round-label">{{ round === 1 ? '独立分析' : '交叉辩论' }}</span>
      </div>

      <div class="messages">
        <div
          v-for="msg in messagesForRound(round)"
          :key="msg.id"
          class="message-card"
          :class="agentClass(msg.agent_name)"
        >
          <div class="message-header">
            <span class="agent-name">{{ agentDisplayName(msg.agent_name) }}</span>
            <span class="stance-badge" :class="stanceClass(msg.stance)">
              {{ stanceLabel(msg.stance) }}
            </span>
            <span class="confidence">置信度: {{ (msg.confidence * 100).toFixed(0) }}%</span>
          </div>
          <div class="message-content">{{ msg.content }}</div>
          <div v-if="normalizedEvidence(msg.evidence).length > 0" class="evidence">
            <span class="evidence-label">证据:</span>
            <span
              v-for="(ev, i) in normalizedEvidence(msg.evidence)"
              :key="i"
              class="evidence-tag"
            >{{ ev }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { ForumMessage } from "../types/forum";

const props = defineProps<{
  messages: ForumMessage[];
}>();

const rounds = computed(() => {
  const set = new Set(props.messages.map((m) => m.round));
  return [...set].sort((a, b) => a - b);
});

function messagesForRound(round: number): ForumMessage[] {
  return props.messages.filter((m) => m.round === round);
}

function agentDisplayName(name: string): string {
  const names: Record<string, string> = {
    emotion_analyst: "情感分析师",
    opportunity_agent: "机会探索者",
    risk_agent: "风险评估师",
    execution_coach: "执行教练",
    simulator_agent: "模拟推演师",
  };
  return names[name] ?? name;
}

function agentClass(name: string): string {
  return `agent-${name.replace(/_/g, "-")}`;
}

function stanceClass(stance: string): string {
  const m: Record<string, string> = {
    analysis: "stance-analysis",
    support: "stance-support",
    oppose: "stance-oppose",
    neutral: "stance-neutral",
    synthesize: "stance-synthesize",
  };
  return m[stance] ?? "stance-debate";
}

function stanceLabel(stance: string): string {
  const m: Record<string, string> = {
    analysis: "分析",
    support: "支持",
    oppose: "反对",
    neutral: "中立",
    synthesize: "综合",
  };
  return m[stance] ?? stance;
}

function stringifyEvidenceItem(item: unknown): string | null {
  if (typeof item === "string") {
    return item;
  }
  if (!item || typeof item !== "object") {
    return null;
  }

  const record = item as Record<string, unknown>;
  const title = typeof record.title === "string" ? record.title : "";
  const content = typeof record.content === "string" ? record.content : "";
  const note = typeof record.note === "string" ? record.note : "";

  if (title && content) {
    return `${title}: ${content}`;
  }
  if (title) {
    return title;
  }
  if (content) {
    return content;
  }
  if (note) {
    return note;
  }
  return null;
}

function normalizedEvidence(evidence?: string): string[] {
  if (!evidence) return [];
  try {
    const parsed = JSON.parse(evidence);
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed
      .map((item) => stringifyEvidenceItem(item))
      .filter((item): item is string => Boolean(item));
  } catch {
    return [];
  }
}
</script>

<style scoped>
.debate-timeline {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.round-group {
  border-left: 3px solid var(--color-border);
  padding-left: 16px;
}

.round-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.round-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 500;
  color: var(--color-primary);
  background: var(--color-primary-bg);
}

.round-label {
  color: var(--color-text-muted);
  font-size: 13px;
}

.messages {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.message-card {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 14px;
  border-left: 3px solid var(--color-border-strong);
}

.message-card.agent-emotion-analyst { border-left-color: #f5827a; }
.message-card.agent-opportunity-agent { border-left-color: #7ee8a8; }
.message-card.agent-risk-agent { border-left-color: #f0b866; }
.message-card.agent-execution-coach { border-left-color: #7aadff; }
.message-card.agent-simulator-agent { border-left-color: #c4a0f5; }

.message-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  flex-wrap: wrap;
}

.agent-name {
  font-weight: 600;
  font-size: 14px;
  color: var(--color-text);
}

.stance-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 500;
}

.stance-analysis { color: var(--color-info); background: var(--color-info-bg); }
.stance-support { color: var(--color-success); background: var(--color-success-bg); }
.stance-oppose { color: var(--color-danger); background: var(--color-danger-bg); }
.stance-neutral { color: var(--color-text-muted); background: var(--color-bg-tertiary); }
.stance-synthesize { color: #c4a0f5; background: rgba(196,160,245,0.12); }
.stance-debate { color: var(--color-warning); background: var(--color-warning-bg); }

.confidence {
  margin-left: auto;
  color: var(--color-text-muted);
  font-size: 12px;
}

.message-content {
  font-size: 14px;
  line-height: 1.6;
  color: var(--color-text);
  white-space: pre-wrap;
}

.evidence {
  margin-top: 8px;
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}

.evidence-label {
  font-size: 12px;
  color: var(--color-text-muted);
}

.evidence-tag {
  background: var(--color-bg-tertiary);
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-size: 11px;
  color: var(--color-text-secondary);
}
</style>
