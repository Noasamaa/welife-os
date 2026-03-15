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
