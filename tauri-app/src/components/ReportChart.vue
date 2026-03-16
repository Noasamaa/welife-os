<template>
  <div class="report-chart">
    <VChart
      v-if="hasValidData"
      :option="enhancedOption"
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

const enhancedOption = computed(() => {
  if (!hasValidData.value) return {};
  const raw = props.section.data as ReportChartData;
  return applyStyle(raw, props.section.chart_type);
});

function isChartData(value: unknown): value is ReportChartData {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function applyStyle(raw: ReportChartData, chartType?: string): ReportChartData {
  const option = JSON.parse(JSON.stringify(raw)) as Record<string, unknown>;

  // Enhanced tooltip
  option.tooltip = {
    trigger: "axis",
    backgroundColor: "rgba(15, 23, 42, 0.92)",
    borderColor: "rgba(255,255,255,0.08)",
    textStyle: { color: "#e2e8f0", fontSize: 13 },
    padding: [10, 14],
    ...(isRecord(option.tooltip) ? option.tooltip : {}),
  };

  // Enhanced grid
  option.grid = {
    left: 48,
    right: 24,
    top: 24,
    bottom: 36,
    containLabel: false,
    ...(isRecord(option.grid) ? option.grid : {}),
  };

  // Style axes
  const xAxis = toArray(option.xAxis);
  const yAxis = toArray(option.yAxis);

  for (const ax of xAxis) {
    if (!isRecord(ax)) continue;
    ax.axisLine = { show: false };
    ax.axisTick = { show: false };
    ax.axisLabel = {
      color: "#94a3b8",
      fontSize: 11,
      ...(isRecord(ax.axisLabel) ? ax.axisLabel : {}),
    };
    ax.splitLine = { show: false };
  }

  for (const ax of yAxis) {
    if (!isRecord(ax)) continue;
    ax.axisLine = { show: false };
    ax.axisTick = { show: false };
    ax.axisLabel = {
      color: "#94a3b8",
      fontSize: 11,
      ...(isRecord(ax.axisLabel) ? ax.axisLabel : {}),
    };
    ax.splitLine = {
      show: true,
      lineStyle: { color: "rgba(148,163,184,0.1)", type: "dashed" as const },
    };
  }

  if (xAxis.length > 0) option.xAxis = xAxis.length === 1 ? xAxis[0] : xAxis;
  if (yAxis.length > 0) option.yAxis = yAxis.length === 1 ? yAxis[0] : yAxis;

  // Style series
  const series = toArray(option.series);
  const gradientColors = [
    { line: "#7ee8a8", area: ["rgba(126,232,168,0.35)", "rgba(126,232,168,0.02)"] },
    { line: "#7aadff", area: ["rgba(122,173,255,0.35)", "rgba(122,173,255,0.02)"] },
    { line: "#f0b866", area: ["rgba(240,184,102,0.35)", "rgba(240,184,102,0.02)"] },
    { line: "#c4a0f5", area: ["rgba(196,160,245,0.35)", "rgba(196,160,245,0.02)"] },
  ];

  for (let i = 0; i < series.length; i++) {
    const s = series[i];
    if (!isRecord(s)) continue;

    if (s.type === "line" || chartType === "line") {
      const palette = gradientColors[i % gradientColors.length];
      s.smooth = true;
      s.symbol = "circle";
      s.symbolSize = 6;
      s.lineStyle = {
        width: 2.5,
        color: palette.line,
        ...(isRecord(s.lineStyle) ? s.lineStyle : {}),
      };
      s.itemStyle = {
        color: palette.line,
        borderColor: "#fff",
        borderWidth: 2,
        ...(isRecord(s.itemStyle) ? s.itemStyle : {}),
      };
      s.areaStyle = {
        color: {
          type: "linear",
          x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: palette.area[0] },
            { offset: 1, color: palette.area[1] },
          ],
        },
      };
      // Emphasis
      s.emphasis = {
        focus: "series",
        itemStyle: { borderWidth: 3, shadowBlur: 8, shadowColor: palette.line + "60" },
      };
    }
  }

  if (series.length > 0) option.series = series;

  // Remove title (handled by ReportViewer section title)
  option.title = undefined;

  // Animation
  option.animation = true;
  option.animationDuration = 800;
  option.animationEasing = "cubicOut";

  return option as ReportChartData;
}

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === "object" && v !== null && !Array.isArray(v);
}

function toArray(v: unknown): Record<string, unknown>[] {
  if (Array.isArray(v)) return v.filter(isRecord);
  if (isRecord(v)) return [v];
  return [];
}
</script>

<style scoped>
.report-chart {
  width: 100%;
  min-height: 350px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
  background: var(--color-bg-card);
}

.chart-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: var(--color-text-muted);
  font-size: 14px;
  background: var(--color-bg-secondary);
  border-radius: var(--radius-lg);
}
</style>
