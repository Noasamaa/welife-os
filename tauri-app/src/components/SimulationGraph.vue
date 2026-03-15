<template>
  <div class="sim-graph-compare">
    <div class="graph-panel">
      <h4>原始关系图谱</h4>
      <VChart v-if="originalOption" :option="originalOption" :autoresize="true" style="width:100%;height:300px" />
      <div v-else class="placeholder">暂无数据</div>
    </div>
    <div class="graph-panel">
      <h4>模拟后图谱</h4>
      <VChart v-if="finalOption" :option="finalOption" :autoresize="true" style="width:100%;height:300px" />
      <div v-else class="placeholder">暂无数据</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import VChart from "vue-echarts";
import { use } from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import { GraphChart } from "echarts/charts";
import { TitleComponent, TooltipComponent } from "echarts/components";

use([CanvasRenderer, GraphChart, TitleComponent, TooltipComponent]);

const props = defineProps<{
  originalSnapshot?: string;
  finalSnapshot?: string;
}>();

function parseSnapshot(snapshot?: string): any {
  if (!snapshot) return null;
  try {
    const data = JSON.parse(snapshot);
    if (!data.nodes) return null;
    return {
      series: [{
        type: "graph",
        layout: "force",
        force: { repulsion: 120, edgeLength: 80 },
        roam: true,
        label: { show: true, fontSize: 11 },
        data: (data.nodes || []).map((n: any) => ({
          name: n.name || n.Name,
          symbolSize: 30,
          category: n.type || n.Type,
        })),
        links: (data.edges || []).map((e: any) => ({
          source: e.source || e.Source,
          target: e.target || e.Target,
          value: e.weight || e.Weight || 1,
        })),
      }],
      tooltip: {},
    };
  } catch {
    return null;
  }
}

const originalOption = computed(() => parseSnapshot(props.originalSnapshot));
const finalOption = computed(() => parseSnapshot(props.finalSnapshot));
</script>

<style scoped>
.sim-graph-compare { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.graph-panel {
  border: 1px solid var(--color-border, #e0e0e0);
  border-radius: 8px;
  padding: 12px;
  background: var(--color-bg-card, #fff);
}
.graph-panel h4 { margin: 0 0 8px; font-size: 14px; }
.placeholder {
  height: 200px; display: flex; align-items: center; justify-content: center;
  color: var(--color-text-secondary, #888); font-size: 13px;
}
</style>
