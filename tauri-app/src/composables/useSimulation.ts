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

  async function loadProfiles() {
    loading.value = true;
    error.value = null;
    try {
      profiles.value = await fetchProfiles();
    } catch (e: any) {
      error.value = e.message ?? "加载人物画像失败";
    } finally {
      loading.value = false;
    }
  }

  async function buildAllProfiles() {
    building.value = true;
    error.value = null;
    try {
      await buildProfiles();
      await loadProfiles();
    } catch (e: any) {
      error.value = e.message ?? "构建画像失败";
    } finally {
      building.value = false;
    }
  }

  async function loadSessions() {
    loading.value = true;
    error.value = null;
    try {
      sessions.value = await fetchSimulations();
    } catch (e: any) {
      error.value = e.message ?? "加载模拟列表失败";
    } finally {
      loading.value = false;
    }
  }

  async function loadSession(id: string) {
    loading.value = true;
    error.value = null;
    try {
      currentSession.value = await fetchSimulation(id);
    } catch (e: any) {
      error.value = e.message ?? "加载模拟详情失败";
    } finally {
      loading.value = false;
    }
  }

  async function startSimulation(
    forkDescription: string,
    affectedNodes: string[],
    changes: Record<string, string>,
    steps?: number,
  ) {
    running.value = true;
    error.value = null;
    try {
      const result = await runSimulation(forkDescription, affectedNodes, changes, steps);
      await loadSessions();
      return result;
    } catch (e: any) {
      error.value = e.message ?? "启动模拟失败";
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
