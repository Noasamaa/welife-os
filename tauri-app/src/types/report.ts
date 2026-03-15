export type ReportType = "weekly" | "monthly" | "annual";
export type ReportStatus = "running" | "completed" | "failed";

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
  type: "chart" | "list" | "text";
  chart_type?: "line" | "network" | "heatmap";
  data?: any;
  items?: any[];
  narrative: string;
}
