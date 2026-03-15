import { ref } from "vue";
import type { PersonProfile, SimulationSession, SimulationDetail } from "../types/simulation";
import {
  buildProfiles,
  fetchProfiles,
  runSimulation,
  fetchSimulations,
  fetchSimulation,
} from "../services/api";

export function useSimulation() {
  const profiles = ref<PersonProfile[]>([]);
  const sessions = ref<SimulationSession[]>([]);
  const currentSession = ref<SimulationDetail | null>(null);
  const loading = ref(false);
  const running = ref(false);
  const building = ref(false);
  const error = ref<string | null>(null);

  async function loadProfiles(conversationID: string) {
    if (!conversationID) {
      profiles.value = [];
      return;
    }
    loading.value = true;
    error.value = null;
    try {
      profiles.value = await fetchProfiles(conversationID);
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : "加载人物画像失败";
    } finally {
      loading.value = false;
    }
  }

  async function buildAllProfiles(conversationID: string): Promise<{ task_id: string } | null> {
    if (!conversationID) {
      error.value = "请先选择一个对话";
      return null;
    }
    building.value = true;
    error.value = null;
    try {
      return await buildProfiles(conversationID);
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : "构建画像失败";
      return null;
    } finally {
      building.value = false;
    }
  }

  async function loadSessions(conversationID: string) {
    if (!conversationID) {
      sessions.value = [];
      currentSession.value = null;
      return;
    }
    loading.value = true;
    error.value = null;
    try {
      sessions.value = await fetchSimulations(conversationID);
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : "加载模拟列表失败";
    } finally {
      loading.value = false;
    }
  }

  async function loadSession(id: string, conversationID: string) {
    if (!conversationID) {
      currentSession.value = null;
      return;
    }
    loading.value = true;
    error.value = null;
    try {
      currentSession.value = await fetchSimulation(id, conversationID);
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : "加载模拟详情失败";
    } finally {
      loading.value = false;
    }
  }

  async function startSimulation(
    conversationID: string,
    forkDescription: string,
    affectedNodes: string[],
    changes: Record<string, string>,
    steps?: number,
  ) {
    if (!conversationID) {
      error.value = "请先选择一个对话";
      return null;
    }
    running.value = true;
    error.value = null;
    try {
      const result = await runSimulation(conversationID, forkDescription, affectedNodes, changes, steps);
      await Promise.all([
        loadSessions(conversationID),
        loadSession(result.session_id, conversationID),
      ]);
      return result;
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : "启动模拟失败";
      return null;
    } finally {
      running.value = false;
    }
  }

  return {
    profiles, sessions, currentSession,
    loading, running, building, error,
    loadProfiles, buildAllProfiles,
    loadSessions, loadSession, startSimulation,
  };
}
