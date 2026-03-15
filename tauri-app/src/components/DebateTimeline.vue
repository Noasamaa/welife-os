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
              {{ msg.stance }}
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
    opportunity_miner: "机会挖掘师",
    risk_debate_team: "风险辩论团",
  };
  return names[name] ?? name;
}

function agentClass(name: string): string {
  return `agent-${name.replace(/_/g, "-")}`;
}

function stanceClass(stance: string): string {
  if (stance === "analysis") return "stance-analysis";
  return "stance-debate";
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
  border-left: 3px solid var(--color-border, #ddd);
  padding-left: 16px;
}

.round-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.round-badge {
  background: var(--color-primary, #4a90d9);
  color: white;
  padding: 2px 10px;
  border-radius: 12px;
  font-size: 13px;
  font-weight: 600;
}

.round-label {
  color: var(--color-text-secondary, #666);
  font-size: 13px;
}

.messages {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.message-card {
  background: var(--color-bg-card, #fff);
  border: 1px solid var(--color-border, #e0e0e0);
  border-radius: 8px;
  padding: 14px;
  border-left: 4px solid #ccc;
}

.message-card.agent-emotion-analyst {
  border-left-color: #e74c3c;
}

.message-card.agent-opportunity-miner {
  border-left-color: #27ae60;
}

.message-card.agent-risk-debate-team {
  border-left-color: #f39c12;
}

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
}

.stance-badge {
  padding: 1px 8px;
  border-radius: 10px;
  font-size: 12px;
}

.stance-analysis {
  background: #e8f4fd;
  color: #2980b9;
}

.stance-debate {
  background: #fef3e2;
  color: #e67e22;
}

.confidence {
  margin-left: auto;
  color: var(--color-text-secondary, #888);
  font-size: 12px;
}

.message-content {
  font-size: 14px;
  line-height: 1.6;
  color: var(--color-text, #333);
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
  color: var(--color-text-secondary, #888);
}

.evidence-tag {
  background: var(--color-bg-secondary, #f5f5f5);
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 11px;
  color: var(--color-text-secondary, #666);
}
</style>
