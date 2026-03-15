<template>
  <div class="graph-view">
    <div v-if="loading" class="center">加载中...</div>
    <div v-else-if="error" class="center error">{{ error }}</div>
    <div v-else-if="!overview || overview.nodes.length === 0" class="center empty">
      暂无图谱数据，请先导入对话并构建图谱。
    </div>
    <template v-else>
      <div class="stats">
        <span class="stat">{{ overview.stats.entity_count }} 个实体</span>
        <span class="stat">{{ overview.stats.relationship_count }} 条关系</span>
        <span v-for="(count, type) in overview.stats.entity_types" :key="type" class="tag">
          {{ type }}: {{ count }}
        </span>
      </div>
      <svg ref="svgEl" class="canvas" :viewBox="viewBox">
        <!-- edges -->
        <line
          v-for="edge in overview.edges"
          :key="edge.id"
          :x1="pos(edge.source).x"
          :y1="pos(edge.source).y"
          :x2="pos(edge.target).x"
          :y2="pos(edge.target).y"
          class="edge"
          :stroke-width="Math.max(1, edge.weight)"
        />
        <!-- nodes -->
        <g v-for="node in overview.nodes" :key="node.id">
          <circle
            :cx="pos(node.id).x"
            :cy="pos(node.id).y"
            :r="14"
            :class="['node', node.type]"
          />
          <text
            :x="pos(node.id).x"
            :y="pos(node.id).y + 28"
            text-anchor="middle"
            class="label"
          >{{ node.name }}</text>
        </g>
      </svg>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { GraphOverview } from "../types/import";

const props = defineProps<{
  overview: GraphOverview | null;
  loading: boolean;
  error: string | null;
}>();

// Simple circular layout
const positions = computed(() => {
  const map: Record<string, { x: number; y: number }> = {};
  const nodes = props.overview?.nodes ?? [];
  const cx = 300, cy = 300, r = Math.min(250, nodes.length * 30);
  nodes.forEach((n, i) => {
    const angle = (2 * Math.PI * i) / nodes.length - Math.PI / 2;
    map[n.id] = {
      x: cx + r * Math.cos(angle),
      y: cy + r * Math.sin(angle),
    };
  });
  return map;
});

const viewBox = computed(() => "0 0 600 600");

function pos(id: string) {
  return positions.value[id] ?? { x: 300, y: 300 };
}
</script>

<style scoped>
.graph-view {
  min-height: 300px;
}

.center {
  text-align: center;
  padding: 40px;
  color: #7a9a8e;
}

.error {
  color: #c0392b;
}

.stats {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 12px;
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

.canvas {
  width: 100%;
  max-height: 500px;
  background: rgba(0, 0, 0, 0.02);
  border-radius: 8px;
}

.edge {
  stroke: #b7c9c1;
  stroke-opacity: 0.6;
}

.node {
  fill: #2d6a4f;
  stroke: #fff;
  stroke-width: 2;
}

.node.person { fill: #2d6a4f; }
.node.event { fill: #e67e22; }
.node.topic { fill: #3498db; }
.node.promise { fill: #9b59b6; }
.node.place { fill: #e74c3c; }

.label {
  font-size: 11px;
  fill: #333;
}
</style>
