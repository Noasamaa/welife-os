<template>
  <div class="graph-view">
    <div v-if="loading" class="center">加载中...</div>
    <div v-else-if="error" class="center error">{{ error }}</div>
    <div v-else-if="!overview || overview.nodes.length === 0" class="center empty">
      暂无图谱数据，请先导入对话并构建图谱。
    </div>
    <template v-else>
      <div class="stats-bar">
        <div class="stats">
          <span class="stat">{{ overview.stats.entity_count }} 个实体</span>
          <span class="stat">{{ overview.stats.relationship_count }} 条关系</span>
          <span v-for="(count, type) in overview.stats.entity_types" :key="type" class="tag">
            {{ entityTypeLabel(type as string) }}: {{ count }}
          </span>
        </div>
        <div class="controls">
          <button class="ctrl-btn" title="放大" @click="controls.zoomIn()">+</button>
          <button class="ctrl-btn" title="缩小" @click="controls.zoomOut()">-</button>
          <button class="ctrl-btn" title="重置视图" @click="controls.resetView()">&#8962;</button>
        </div>
      </div>
      <div class="graph-canvas-wrapper">
        <div ref="containerRef" class="pixi-container" />
        <GraphFilterPanel
          v-if="overview"
          :filters="controls.filters"
          :entity-types="overview.stats.entity_types"
          :on-refresh="() => controls.refresh()"
        />
      </div>
      <div v-if="controls.selectedNode.value" class="node-info">
        <span class="node-info-label">选中:</span>
        <span class="node-info-name">{{ selectedNodeName }}</span>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, toRef, computed } from "vue";
import type { GraphOverview } from "../types/import";
import { usePixiGraph } from "../composables/usePixiGraph";
import GraphFilterPanel from "./GraphFilterPanel.vue";

const props = defineProps<{
  overview: GraphOverview | null;
  loading: boolean;
  error: string | null;
}>();

const emit = defineEmits<{
  (e: "node-click", payload: { id: string; type: string; name: string }): void;
}>();

const containerRef = ref<HTMLElement | null>(null);

const controls = usePixiGraph(
  containerRef,
  toRef(props, "overview"),
  (id, type, name) => {
    emit("node-click", { id, type, name });
  },
);

const selectedNodeName = computed(() => {
  const nodeId = controls.selectedNode.value;
  if (!nodeId || !props.overview) return "";
  const node = props.overview.nodes.find((n) => n.id === nodeId);
  return node?.name ?? nodeId;
});

const ENTITY_TYPE_LABELS: Record<string, string> = {
  person: "人物",
  event: "事件",
  topic: "话题",
  promise: "承诺",
  place: "地点",
};

function entityTypeLabel(type: string): string {
  return ENTITY_TYPE_LABELS[type] ?? type;
}

watch(
  () => props.overview,
  async (newOverview) => {
    if (newOverview && newOverview.nodes.length > 0 && containerRef.value) {
      await nextTick();
      controls.reinit();
    }
  },
);

watch(containerRef, async (el) => {
  if (el && props.overview && props.overview.nodes.length > 0) {
    await nextTick();
    controls.reinit();
  }
});
</script>

<style scoped>
.graph-view {
  min-height: 300px;
  display: flex;
  flex-direction: column;
}

.center {
  text-align: center;
  padding: 40px;
  color: var(--color-text-muted);
}

.error {
  color: var(--color-danger);
}

.stats-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.stats {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}

.stat {
  font-weight: 600;
  font-size: 13px;
  color: var(--color-text);
}

.tag {
  font-size: 12px;
  font-weight: 500;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  color: var(--color-text-secondary);
  background: var(--color-bg-tertiary);
}

.controls {
  display: flex;
  gap: 4px;
}

.ctrl-btn {
  width: 28px;
  height: 28px;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-text-secondary);
  font-size: 16px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
}

.ctrl-btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-text);
}

.graph-canvas-wrapper {
  position: relative;
}

.pixi-container {
  width: 100%;
  height: 500px;
  border-radius: var(--radius-lg);
  background: #111827;
  overflow: hidden;
  position: relative;
}

.node-info {
  margin-top: 8px;
  padding: 6px 12px;
  font-size: 13px;
  border-radius: var(--radius-md);
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
}

.node-info-label {
  color: var(--color-text-muted);
  margin-right: 6px;
}

.node-info-name {
  font-weight: 600;
  color: var(--color-text);
}
</style>
