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
            {{ type }}: {{ count }}
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

watch(
  () => props.overview,
  async (newOverview) => {
    if (newOverview && newOverview.nodes.length > 0) {
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
  color: #7a9a8e;
}

.error {
  color: #c0392b;
}

.stats-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.stats {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  align-items: center;
}

.stat {
  font-weight: 600;
  font-size: 14px;
}

.tag {
  font-size: 12px;
  background: rgba(45, 106, 79, 0.08);
  padding: 2px 8px;
  border-radius: 4px;
  color: #2d6a4f;
}

.controls {
  display: flex;
  gap: 4px;
}

.ctrl-btn {
  width: 28px;
  height: 28px;
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.06);
  color: #ccc;
  font-size: 16px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s;
}

.ctrl-btn:hover {
  background: rgba(255, 255, 255, 0.12);
}

.graph-canvas-wrapper {
  position: relative;
}

.pixi-container {
  width: 100%;
  height: 500px;
  border-radius: 8px;
  background: #1a1a2e;
  overflow: hidden;
  position: relative;
}

.node-info {
  margin-top: 8px;
  padding: 6px 12px;
  font-size: 13px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.05);
}

.node-info-label {
  color: #888;
  margin-right: 6px;
}

.node-info-name {
  font-weight: 600;
  color: #e0e0e0;
}
</style>
