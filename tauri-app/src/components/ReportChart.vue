<template>
  <div class="report-chart" ref="chartContainer">
    <VChart
      v-if="hasValidData"
      :option="chartOption"
      :autoresize="true"
      style="width: 100%; height: 350px"
    />
    <div v-else class="chart-placeholder">
      暂无图表数据
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import VChart from "vue-echarts";
import { use } from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import { LineChart, HeatmapChart, GraphChart } from "echarts/charts";
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  VisualMapComponent,
  CalendarComponent,
} from "echarts/components";
import type { ReportChartData, ReportSection } from "../types/report";

use([
  CanvasRenderer,
  LineChart,
  HeatmapChart,
  GraphChart,
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  VisualMapComponent,
  CalendarComponent,
]);

const props = defineProps<{
  section: ReportSection;
}>();

const hasValidData = computed(() => {
  return isChartData(props.section.data);
});

const chartOption = computed(() => {
  if (!hasValidData.value) return {};
  return props.section.data as ReportChartData;
});

function isChartData(value: unknown): value is ReportChartData {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
</script>

<style scoped>
.report-chart {
  width: 100%;
  min-height: 350px;
}

.chart-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: var(--color-text-secondary, #888);
  font-size: 14px;
  background: var(--color-bg-secondary, #f8f9fa);
  border-radius: 8px;
}
</style>
