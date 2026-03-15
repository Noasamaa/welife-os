import { ref } from "vue";
import type { ForumSession, ForumSessionDetail } from "../types/forum";
import {
  triggerDebate,
  fetchForumSessions,
  fetchForumSession,
} from "../services/api";

export function useForum() {
  const sessions = ref<ForumSession[]>([]);
  const currentSession = ref<ForumSessionDetail | null>(null);
  const loading = ref(false);
  const debating = ref(false);
  const error = ref<string | null>(null);

  async function loadSessions() {
    loading.value = true;
    error.value = null;
    try {
      sessions.value = await fetchForumSessions();
    } catch (e: any) {
      error.value = e.message ?? "加载辩论会话失败";
    } finally {
      loading.value = false;
    }
  }

  async function loadSession(id: string) {
    loading.value = true;
    error.value = null;
    try {
      currentSession.value = await fetchForumSession(id);
    } catch (e: any) {
      error.value = e.message ?? "加载辩论详情失败";
    } finally {
      loading.value = false;
    }
  }

  async function startDebate(
    conversationID: string,
  ): Promise<{ session_id: string; task_id: string } | null> {
    debating.value = true;
    error.value = null;
    try {
      const result = await triggerDebate(conversationID);
      await Promise.all([loadSessions(), loadSession(result.session_id)]);
      return result;
    } catch (e: any) {
      error.value = e.message ?? "启动辩论失败";
      return null;
    } finally {
      debating.value = false;
    }
  }

  return {
    sessions,
    currentSession,
    loading,
    debating,
    error,
    loadSessions,
    loadSession,
    startDebate,
  };
}
