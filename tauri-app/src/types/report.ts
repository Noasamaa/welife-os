export type ReportType = "weekly" | "monthly" | "annual";
export type ReportStatus = "running" | "completed" | "failed";
export type ReportSectionType = "chart" | "list" | "text";
export type ReportChartType = "line" | "network" | "heatmap";
export type ReportListItem = string | number | boolean | Record<string, unknown>;
export type ReportChartData = Record<string, unknown>;

const allowedReportTypes: ReadonlySet<ReportType> = new Set(["weekly", "monthly", "annual"]);
const allowedSectionTypes: ReadonlySet<ReportSectionType> = new Set(["chart", "list", "text"]);
const allowedChartTypes: ReadonlySet<ReportChartType> = new Set(["line", "network", "heatmap"]);

export interface Report {
  id: string;
  type: ReportType;
  conversation_id: string;
  task_id: string;
  status: ReportStatus;
  title: string;
  content: string;
  period_start: string;
  period_end: string;
  created_at: string;
  completed_at?: string;
}

export interface ReportContent {
  title: string;
  type: ReportType;
  period: { start: string; end: string };
  sections: ReportSection[];
  summary: string;
}

export interface ReportSection {
  title: string;
  type: ReportSectionType;
  chart_type?: ReportChartType;
  data?: ReportChartData;
  items?: ReportListItem[];
  narrative: string;
}

export function sanitizeReportContent(raw: unknown): ReportContent | null {
  if (!isRecord(raw)) {
    return null;
  }

  const period = isRecord(raw.period) ? raw.period : {};
  const sections = Array.isArray(raw.sections)
    ? raw.sections.map((section) => sanitizeReportSection(section)).filter((section): section is ReportSection => section !== null)
    : [];

  return {
    title: sanitizeText(raw.title, "未命名报告", 160),
    type: sanitizeReportType(raw.type),
    period: {
      start: sanitizeText(period.start, "", 40),
      end: sanitizeText(period.end, "", 40),
    },
    sections,
    summary: sanitizeText(raw.summary, "", 4000),
  };
}

export function sanitizeReportSection(raw: unknown): ReportSection | null {
  if (!isRecord(raw)) {
    return null;
  }

  const type = sanitizeSectionType(raw.type);
  const section: ReportSection = {
    title: sanitizeText(raw.title, "未命名章节", 120),
    type,
    narrative: sanitizeText(raw.narrative, "", 6000),
  };

  if (type === "chart") {
    const chartType = sanitizeChartType(raw.chart_type);
    if (!chartType) {
      return null;
    }
    section.chart_type = chartType;
    section.data = sanitizeChartData(chartType, raw.data);
  } else if (type === "list") {
    section.items = sanitizeListItems(raw.items);
  }

  return section;
}

export function sanitizeChartData(
  chartType: ReportChartType,
  raw: unknown,
): ReportChartData | undefined {
  if (!isRecord(raw)) {
    return undefined;
  }

  const allowedKeysByType: Record<ReportChartType, string[]> = {
    line: ["title", "tooltip", "legend", "grid", "xAxis", "yAxis", "series"],
    heatmap: ["title", "tooltip", "legend", "grid", "xAxis", "yAxis", "calendar", "visualMap", "series"],
    network: ["title", "tooltip", "legend", "series"],
  };

  const sanitized: ReportChartData = {};
  for (const key of allowedKeysByType[chartType]) {
    if (!(key in raw)) {
      continue;
    }
    const value = sanitizeJSONValue(raw[key], 0);
    if (value !== undefined) {
      sanitized[key] = value;
    }
  }
  return Object.keys(sanitized).length > 0 ? sanitized : undefined;
}

function sanitizeListItems(raw: unknown): ReportListItem[] {
  if (!Array.isArray(raw)) {
    return [];
  }
  return raw
    .slice(0, 20)
    .map((item) => sanitizeJSONValue(item, 0))
    .filter((item): item is ReportListItem => item !== undefined);
}

function sanitizeJSONValue(value: unknown, depth: number): unknown {
  if (depth > 4) {
    return undefined;
  }
  if (typeof value === "string") {
    return sanitizeText(value, "", 2000);
  }
  if (typeof value === "number") {
    return Number.isFinite(value) ? value : undefined;
  }
  if (typeof value === "boolean") {
    return value;
  }
  if (Array.isArray(value)) {
    return value
      .slice(0, 50)
      .map((item) => sanitizeJSONValue(item, depth + 1))
      .filter((item) => item !== undefined);
  }
  if (!isRecord(value)) {
    return undefined;
  }

  const output: Record<string, unknown> = {};
  for (const [key, nested] of Object.entries(value).slice(0, 50)) {
    const sanitized = sanitizeJSONValue(nested, depth + 1);
    if (sanitized !== undefined) {
      output[key] = sanitized;
    }
  }
  return output;
}

function sanitizeReportType(value: unknown): ReportType {
  return typeof value === "string" && allowedReportTypes.has(value as ReportType)
    ? (value as ReportType)
    : "weekly";
}

function sanitizeSectionType(value: unknown): ReportSectionType {
  return typeof value === "string" && allowedSectionTypes.has(value as ReportSectionType)
    ? (value as ReportSectionType)
    : "text";
}

function sanitizeChartType(value: unknown): ReportChartType | undefined {
  return typeof value === "string" && allowedChartTypes.has(value as ReportChartType)
    ? (value as ReportChartType)
    : undefined;
}

function sanitizeText(value: unknown, fallback: string, maxLength: number): string {
  if (typeof value !== "string") {
    return fallback;
  }
  return value.trim().slice(0, maxLength);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
