import type { SystemStatusResponse, LLMConfig } from "../types/api";
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
import type { ActionItem, ActionPlanResponse } from "../types/coach";
import type { ReminderRule, Reminder } from "../types/reminder";
import type { PersonProfile, SimulationSession, SimulationDetail } from "../types/simulation";
import { getBackendRuntime } from "./tauri";

function assertObject(data: unknown, endpoint: string): asserts data is Record<string, unknown> {
  if (typeof data !== "object" || data === null || Array.isArray(data)) {
    throw new Error(`${endpoint}: expected JSON object, got ${typeof data}`);
  }
}

export async function getAPIBaseURL(): Promise<string> {
  const runtime = await getBackendRuntime();
  return runtime.baseUrl || "/api";
}

async function apiFetch(path: string, init?: RequestInit): Promise<Response> {
  const runtime = await getBackendRuntime();
  const headers = new Headers(init?.headers);
  if (runtime.apiToken) {
    headers.set("X-WeLife-API-Token", runtime.apiToken);
  }

  return fetch(`${runtime.baseUrl}${path}`, {
    ...init,
    headers,
  });
}

async function readError(response: Response, fallback: string): Promise<string> {
  const text = (await response.text()).trim();
  return text || fallback;
}

function appendConversationQuery(path: string, conversationID: string): string {
  const params = new URLSearchParams({ conversation_id: conversationID });
  return `${path}?${params.toString()}`;
}

export async function fetchSystemStatus(): Promise<SystemStatusResponse> {
  const response = await apiFetch("/api/v1/system/status");
  if (!response.ok) {
    throw new Error(`failed to fetch system status: ${response.status}`);
  }

  const data: unknown = await response.json();
  assertObject(data, "system/status");
  return data as unknown as SystemStatusResponse;
}

export async function fetchLLMConfig(): Promise<LLMConfig> {
  const response = await apiFetch("/api/v1/system/llm-config");
  if (!response.ok) throw new Error(`fetch llm config: ${response.status}`);
  return (await response.json()) as LLMConfig;
}

export async function updateLLMConfig(patch: Partial<LLMConfig>): Promise<void> {
  const response = await apiFetch("/api/v1/system/llm-config", {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
  if (!response.ok) {
    throw new Error(await readError(response, `update llm config: ${response.status}`));
  }
}

export async function uploadFile(
  file: File,
  format?: string,
  selfName?: string,
): Promise<ImportResult> {
  const form = new FormData();
  form.append("file", file);
  if (format) form.append("format", format);
  if (selfName) form.append("self_name", selfName);

  const response = await apiFetch("/api/v1/import", {
    method: "POST",
    body: form,
  });
  if (!response.ok) throw new Error(await readError(response, "upload failed"));
  return (await response.json()) as ImportResult;
}

export async function fetchImportJobs(): Promise<ImportJob[]> {
  const response = await apiFetch("/api/v1/import/jobs");
  if (!response.ok) throw new Error(`fetch jobs: ${response.status}`);
  return (await response.json()) as ImportJob[];
}

export async function fetchConversations(): Promise<Conversation[]> {
  const response = await apiFetch("/api/v1/conversations");
  if (!response.ok) throw new Error(`fetch conversations: ${response.status}`);
  return (await response.json()) as Conversation[];
}

export async function triggerGraphBuild(
  conversationID: string,
): Promise<{ task_id: string }> {
  const response = await apiFetch("/api/v1/graph/build", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ conversation_id: conversationID }),
  });
  if (!response.ok) throw new Error(await readError(response, "trigger graph build failed"));
  return (await response.json()) as { task_id: string };
}

export async function fetchGraphOverview(): Promise<GraphOverview> {
  const response = await apiFetch("/api/v1/graph/overview");
  if (!response.ok) throw new Error(`fetch graph: ${response.status}`);
  return (await response.json()) as GraphOverview;
}

export async function triggerDebate(
  conversationID: string,
): Promise<{ session_id: string; task_id: string }> {
  const response = await apiFetch("/api/v1/forum/debate", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ conversation_id: conversationID }),
  });
  if (!response.ok) throw new Error(await readError(response, "trigger debate failed"));
  return (await response.json()) as { session_id: string; task_id: string };
}

export async function fetchForumSessions(): Promise<ForumSession[]> {
  const response = await apiFetch("/api/v1/forum/sessions");
  if (!response.ok) throw new Error(`fetch sessions: ${response.status}`);
  return (await response.json()) as ForumSession[];
}

export async function fetchForumSession(
  id: string,
): Promise<ForumSessionDetail> {
  const response = await apiFetch(`/api/v1/forum/sessions/${id}`);
  if (!response.ok) throw new Error(`fetch session: ${response.status}`);
  return (await response.json()) as ForumSessionDetail;
}

export async function generateReport(
  type: ReportType,
  conversationID: string,
  periodStart?: string,
  periodEnd?: string,
): Promise<{ report_id: string; task_id: string }> {
  const body: Record<string, string> = { type, conversation_id: conversationID };
  if (periodStart) body.period_start = periodStart;
  if (periodEnd) body.period_end = periodEnd;

  const response = await apiFetch("/api/v1/reports/generate", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!response.ok) throw new Error(await readError(response, "generate report failed"));
  return (await response.json()) as { report_id: string; task_id: string };
}

export async function fetchReports(): Promise<Report[]> {
  const response = await apiFetch("/api/v1/reports");
  if (!response.ok) throw new Error(`fetch reports: ${response.status}`);
  return (await response.json()) as Report[];
}

export async function fetchReport(id: string): Promise<Report> {
  const response = await apiFetch(`/api/v1/reports/${id}`);
  if (!response.ok) throw new Error(`fetch report: ${response.status}`);
  return (await response.json()) as Report;
}

export async function deleteReport(id: string): Promise<void> {
  const response = await apiFetch(`/api/v1/reports/${id}`, {
    method: "DELETE",
  });
  if (!response.ok) throw new Error(`delete report: ${response.status}`);
}

export async function fetchReportExportBlob(id: string, format: "html" | "pdf"): Promise<Blob> {
  const response = await apiFetch(`/api/v1/reports/${encodeURIComponent(id)}/${format}`);
  if (!response.ok) {
    throw new Error(await readError(response, `export report ${format}: ${response.status}`));
  }
  return response.blob();
}

export async function generateActionPlan(
  sessionID: string,
): Promise<ActionPlanResponse> {
  const response = await apiFetch("/api/v1/coach/generate-plan", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ session_id: sessionID }),
  });
  if (!response.ok) throw new Error(await readError(response, "generate action plan failed"));
  return (await response.json()) as ActionPlanResponse;
}

export async function fetchActionItems(
  status?: string,
  category?: string,
): Promise<ActionItem[]> {
  const params = new URLSearchParams();
  if (status) params.set("status", status);
  if (category) params.set("category", category);
  const suffix = params.toString();
  const response = await apiFetch(`/api/v1/action-items${suffix ? `?${suffix}` : ""}`);
  if (!response.ok) throw new Error(`fetch action items: ${response.status}`);
  return (await response.json()) as ActionItem[];
}

export async function updateActionItemStatus(
  id: string,
  status: string,
): Promise<void> {
  const response = await apiFetch(`/api/v1/action-items/${id}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ status }),
  });
  if (!response.ok) throw new Error(await readError(response, "update action item failed"));
}

export async function deleteActionItem(id: string): Promise<void> {
  const response = await apiFetch(`/api/v1/action-items/${id}`, {
    method: "DELETE",
  });
  if (!response.ok) throw new Error(`delete action item: ${response.status}`);
}

export async function fetchPendingReminders(): Promise<Reminder[]> {
  const response = await apiFetch("/api/v1/reminders/pending");
  if (!response.ok) throw new Error(`fetch reminders: ${response.status}`);
  return (await response.json()) as Reminder[];
}

export async function markReminderRead(id: string): Promise<void> {
  const response = await apiFetch(`/api/v1/reminders/${id}/read`, {
    method: "PATCH",
  });
  if (!response.ok) throw new Error(`mark read: ${response.status}`);
}

export async function dismissReminder(id: string): Promise<void> {
  const response = await apiFetch(`/api/v1/reminders/${id}/dismiss`, {
    method: "PATCH",
  });
  if (!response.ok) throw new Error(`dismiss: ${response.status}`);
}

export async function fetchReminderRules(): Promise<ReminderRule[]> {
  const response = await apiFetch("/api/v1/reminder-rules");
  if (!response.ok) throw new Error(`fetch rules: ${response.status}`);
  return (await response.json()) as ReminderRule[];
}

export async function createReminderRule(
  rule: Omit<ReminderRule, "id" | "created_at" | "last_triggered_at">,
): Promise<ReminderRule> {
  const response = await apiFetch("/api/v1/reminder-rules", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(rule),
  });
  if (!response.ok) throw new Error(await readError(response, "create reminder rule failed"));
  return (await response.json()) as ReminderRule;
}

export async function updateReminderRule(
  id: string,
  enabled: boolean,
): Promise<void> {
  const response = await apiFetch(`/api/v1/reminder-rules/${id}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ enabled }),
  });
  if (!response.ok) throw new Error(`update rule: ${response.status}`);
}

export async function deleteReminderRule(id: string): Promise<void> {
  const response = await apiFetch(`/api/v1/reminder-rules/${id}`, {
    method: "DELETE",
  });
  if (!response.ok) throw new Error(`delete rule: ${response.status}`);
}

export async function buildProfiles(conversationID: string): Promise<{ task_id: string }> {
  const response = await apiFetch("/api/v1/simulation/profiles/build", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ conversation_id: conversationID }),
  });
  if (!response.ok) throw new Error(await readError(response, "build profiles failed"));
  return (await response.json()) as { task_id: string };
}

export async function fetchProfiles(conversationID: string): Promise<PersonProfile[]> {
  const response = await apiFetch(appendConversationQuery("/api/v1/simulation/profiles", conversationID));
  if (!response.ok) throw new Error(`fetch profiles: ${response.status}`);
  return (await response.json()) as PersonProfile[];
}

export async function runSimulation(
  conversationID: string,
  forkDescription: string,
  affectedNodes: string[],
  changes: Record<string, string>,
  steps?: number,
): Promise<{ session_id: string; task_id: string }> {
  const response = await apiFetch("/api/v1/simulation/run", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      conversation_id: conversationID,
      steps: steps ?? 5,
      fork_point: {
        description: forkDescription,
        affected_nodes: affectedNodes,
        changes,
      },
    }),
  });
  if (!response.ok) throw new Error(await readError(response, "run simulation failed"));
  return (await response.json()) as { session_id: string; task_id: string };
}

export async function fetchSimulations(conversationID: string): Promise<SimulationSession[]> {
  const response = await apiFetch(appendConversationQuery("/api/v1/simulation/sessions", conversationID));
  if (!response.ok) throw new Error(`fetch simulations: ${response.status}`);
  return (await response.json()) as SimulationSession[];
}

export async function fetchSimulation(id: string, conversationID: string): Promise<SimulationDetail> {
  const path = appendConversationQuery(`/api/v1/simulation/sessions/${id}`, conversationID);
  const response = await apiFetch(path);
  if (!response.ok) throw new Error(`fetch simulation: ${response.status}`);
  return (await response.json()) as SimulationDetail;
}

// ── Task status polling ──

export interface TaskInfo {
  id: string;
  name: string;
  status: "queued" | "running" | "succeeded" | "failed";
  error?: string;
  created_at: string;
  updated_at: string;
}

export async function fetchTaskStatus(taskId: string): Promise<TaskInfo> {
  const response = await apiFetch(`/api/v1/tasks/${encodeURIComponent(taskId)}`);
  if (!response.ok) throw new Error(`fetch task status: ${response.status}`);
  return (await response.json()) as TaskInfo;
}

/**
 * Poll a task until it reaches a terminal state (succeeded/failed).
 * Returns the final TaskInfo.
 *
 * @param taskId - The task ID to poll
 * @param onProgress - Optional callback for each poll (receives current TaskInfo)
 * @param intervalMs - Polling interval in ms (default: 1500)
 * @param timeoutMs - Max total wait time in ms (default: 300000 = 5min)
 */
export function pollTaskUntilDone(
  taskId: string,
  onProgress?: (info: TaskInfo) => void,
  intervalMs = 1500,
  timeoutMs = 300_000,
): { promise: Promise<TaskInfo>; cancel: () => void } {
  let timer: ReturnType<typeof setInterval> | null = null;
  let cancelled = false;

  const promise = new Promise<TaskInfo>((resolve, reject) => {
    const start = Date.now();

    const poll = async () => {
      if (cancelled) return;
      try {
        const info = await fetchTaskStatus(taskId);
        onProgress?.(info);

        if (info.status === "succeeded" || info.status === "failed") {
          if (timer) clearInterval(timer);
          resolve(info);
          return;
        }

        if (Date.now() - start > timeoutMs) {
          if (timer) clearInterval(timer);
          resolve(info); // Return last known state on timeout
        }
      } catch (err) {
        if (Date.now() - start > timeoutMs) {
          if (timer) clearInterval(timer);
          reject(err);
        }
        // Swallow transient errors, keep polling
      }
    };

    // First poll immediately
    poll();
    timer = setInterval(poll, intervalMs);
  });

  const cancel = () => {
    cancelled = true;
    if (timer) clearInterval(timer);
  };

  return { promise, cancel };
}
