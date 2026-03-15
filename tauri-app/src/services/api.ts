import type { SystemStatusResponse } from "../types/api";
import type {
  ImportResult,
  ImportJob,
  Conversation,
  GraphOverview,
} from "../types/import";
import type {
  ForumSession,
  ForumSessionDetail,
} from "../types/forum";
import type { Report, ReportType } from "../types/report";
import type { ActionItem } from "../types/coach";
import type { ReminderRule, Reminder } from "../types/reminder";
import type { PersonProfile, SimulationSession, SimulationDetail } from "../types/simulation";

export const API_BASE_URL = "http://127.0.0.1:18080";

export async function fetchSystemStatus(): Promise<SystemStatusResponse> {
  const response = await fetch(`${API_BASE_URL}/api/v1/system/status`);
  if (!response.ok) {
    throw new Error(`failed to fetch system status: ${response.status}`);
  }

  return (await response.json()) as SystemStatusResponse;
}

// Import
export async function uploadFile(
  file: File,
  format?: string,
  selfName?: string,
): Promise<ImportResult> {
  const form = new FormData();
  form.append("file", file);
  if (format) form.append("format", format);
  if (selfName) form.append("self_name", selfName);

  const res = await fetch(`${API_BASE_URL}/api/v1/import`, {
    method: "POST",
    body: form,
  });
  if (!res.ok) throw new Error(await res.text());
  return (await res.json()) as ImportResult;
}

export async function fetchImportJobs(): Promise<ImportJob[]> {
  const res = await fetch(`${API_BASE_URL}/api/v1/import/jobs`);
  if (!res.ok) throw new Error(`fetch jobs: ${res.status}`);
  return (await res.json()) as ImportJob[];
}

// Conversations
export async function fetchConversations(): Promise<Conversation[]> {
  const res = await fetch(`${API_BASE_URL}/api/v1/conversations`);
  if (!res.ok) throw new Error(`fetch conversations: ${res.status}`);
  return (await res.json()) as Conversation[];
}

// Graph
export async function triggerGraphBuild(
  conversationID: string,
): Promise<{ task_id: string }> {
  const res = await fetch(`${API_BASE_URL}/api/v1/graph/build`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ conversation_id: conversationID }),
  });
  if (!res.ok) throw new Error(await res.text());
  return (await res.json()) as { task_id: string };
}

export async function fetchGraphOverview(): Promise<GraphOverview> {
  const res = await fetch(`${API_BASE_URL}/api/v1/graph/overview`);
  if (!res.ok) throw new Error(`fetch graph: ${res.status}`);
  return (await res.json()) as GraphOverview;
}

// Forum
export async function triggerDebate(
  conversationID: string,
): Promise<{ session_id: string; task_id: string }> {
  const res = await fetch(`${API_BASE_URL}/api/v1/forum/debate`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ conversation_id: conversationID }),
  });
  if (!res.ok) throw new Error(await res.text());
  return (await res.json()) as { session_id: string; task_id: string };
}

export async function fetchForumSessions(): Promise<ForumSession[]> {
  const res = await fetch(`${API_BASE_URL}/api/v1/forum/sessions`);
  if (!res.ok) throw new Error(`fetch sessions: ${res.status}`);
  return (await res.json()) as ForumSession[];
}

export async function fetchForumSession(
  id: string,
): Promise<ForumSessionDetail> {
  const res = await fetch(`${API_BASE_URL}/api/v1/forum/sessions/${id}`);
  if (!res.ok) throw new Error(`fetch session: ${res.status}`);
  return (await res.json()) as ForumSessionDetail;
}

// Reports
export async function generateReport(
  type: ReportType,
  conversationID: string,
  periodStart?: string,
  periodEnd?: string,
): Promise<{ report_id: string; task_id: string }> {
  const body: Record<string, string> = { type, conversation_id: conversationID };
  if (periodStart) body.period_start = periodStart;
  if (periodEnd) body.period_end = periodEnd;

  const res = await fetch(`${API_BASE_URL}/api/v1/reports/generate`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(await res.text());
  return (await res.json()) as { report_id: string; task_id: string };
}

export async function fetchReports(): Promise<Report[]> {
  const res = await fetch(`${API_BASE_URL}/api/v1/reports`);
  if (!res.ok) throw new Error(`fetch reports: ${res.status}`);
  return (await res.json()) as Report[];
}

export async function fetchReport(id: string): Promise<Report> {
  const res = await fetch(`${API_BASE_URL}/api/v1/reports/${id}`);
  if (!res.ok) throw new Error(`fetch report: ${res.status}`);
  return (await res.json()) as Report;
}

export async function deleteReport(id: string): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/api/v1/reports/${id}`, {
    method: "DELETE",
  });
  if (!res.ok) throw new Error(`delete report: ${res.status}`);
}

// Coach / Action Items
export async function generateActionPlan(
  sessionID: string,
): Promise<ActionItem[]> {
  const res = await fetch(`${API_BASE_URL}/api/v1/coach/generate-plan`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ session_id: sessionID }),
  });
  if (!res.ok) throw new Error(await res.text());
  return (await res.json()) as ActionItem[];
}

export async function fetchActionItems(
  status?: string,
  category?: string,
): Promise<ActionItem[]> {
  const params = new URLSearchParams();
  if (status) params.set("status", status);
  if (category) params.set("category", category);
  const q = params.toString();
  const res = await fetch(`${API_BASE_URL}/api/v1/action-items${q ? "?" + q : ""}`);
  if (!res.ok) throw new Error(`fetch action items: ${res.status}`);
  return (await res.json()) as ActionItem[];
}

export async function updateActionItemStatus(
  id: string,
  status: string,
): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/api/v1/action-items/${id}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ status }),
  });
  if (!res.ok) throw new Error(await res.text());
}

export async function deleteActionItem(id: string): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/api/v1/action-items/${id}`, {
    method: "DELETE",
  });
  if (!res.ok) throw new Error(`delete action item: ${res.status}`);
}

// Reminders
export async function fetchPendingReminders(): Promise<Reminder[]> {
  const res = await fetch(`${API_BASE_URL}/api/v1/reminders/pending`);
  if (!res.ok) throw new Error(`fetch reminders: ${res.status}`);
  return (await res.json()) as Reminder[];
}

export async function markReminderRead(id: string): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/api/v1/reminders/${id}/read`, {
    method: "PATCH",
  });
  if (!res.ok) throw new Error(`mark read: ${res.status}`);
}

export async function dismissReminder(id: string): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/api/v1/reminders/${id}/dismiss`, {
    method: "PATCH",
  });
  if (!res.ok) throw new Error(`dismiss: ${res.status}`);
}

export async function fetchReminderRules(): Promise<ReminderRule[]> {
  const res = await fetch(`${API_BASE_URL}/api/v1/reminder-rules`);
  if (!res.ok) throw new Error(`fetch rules: ${res.status}`);
  return (await res.json()) as ReminderRule[];
}

export async function createReminderRule(
  rule: Omit<ReminderRule, "id" | "created_at" | "last_triggered_at">,
): Promise<ReminderRule> {
  const res = await fetch(`${API_BASE_URL}/api/v1/reminder-rules`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(rule),
  });
  if (!res.ok) throw new Error(await res.text());
  return (await res.json()) as ReminderRule;
}

export async function updateReminderRule(
  id: string,
  enabled: boolean,
): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/api/v1/reminder-rules/${id}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ enabled }),
  });
  if (!res.ok) throw new Error(`update rule: ${res.status}`);
}

export async function deleteReminderRule(id: string): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/api/v1/reminder-rules/${id}`, {
    method: "DELETE",
  });
  if (!res.ok) throw new Error(`delete rule: ${res.status}`);
}

// Simulation
export async function buildProfiles(): Promise<{ task_id: string }> {
  const res = await fetch(`${API_BASE_URL}/api/v1/simulation/profiles/build`, {
    method: "POST",
  });
  if (!res.ok) throw new Error(await res.text());
  return (await res.json()) as { task_id: string };
}

export async function fetchProfiles(): Promise<PersonProfile[]> {
  const res = await fetch(`${API_BASE_URL}/api/v1/simulation/profiles`);
  if (!res.ok) throw new Error(`fetch profiles: ${res.status}`);
  return (await res.json()) as PersonProfile[];
}

export async function runSimulation(
  forkDescription: string,
  affectedNodes: string[],
  changes: Record<string, string>,
  steps?: number,
): Promise<{ session_id: string; task_id: string }> {
  const res = await fetch(`${API_BASE_URL}/api/v1/simulation/run`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      steps: steps ?? 5,
      fork_point: {
        description: forkDescription,
        affected_nodes: affectedNodes,
        changes,
      },
    }),
  });
  if (!res.ok) throw new Error(await res.text());
  return (await res.json()) as { session_id: string; task_id: string };
}

export async function fetchSimulations(): Promise<SimulationSession[]> {
  const res = await fetch(`${API_BASE_URL}/api/v1/simulation/sessions`);
  if (!res.ok) throw new Error(`fetch simulations: ${res.status}`);
  return (await res.json()) as SimulationSession[];
}

export async function fetchSimulation(id: string): Promise<SimulationDetail> {
  const res = await fetch(`${API_BASE_URL}/api/v1/simulation/sessions/${id}`);
  if (!res.ok) throw new Error(`fetch simulation: ${res.status}`);
  return (await res.json()) as SimulationDetail;
}
