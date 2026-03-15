import { ref } from "vue";
import type { GraphOverview } from "../types/import";
import { fetchGraphOverview, triggerGraphBuild } from "../services/api";

export function useGraph() {
  const overview = ref<GraphOverview | null>(null);
  const loading = ref(false);
  const building = ref(false);
  const error = ref<string | null>(null);

  async function loadOverview() {
    loading.value = true;
    error.value = null;
    try {
      overview.value = await fetchGraphOverview();
    } catch (e: any) {
      error.value = e.message ?? "加载图谱失败";
    } finally {
      loading.value = false;
    }
  }

  async function buildGraph(conversationID: string) {
    building.value = true;
    error.value = null;
    try {
      await triggerGraphBuild(conversationID);
    } catch (e: any) {
      error.value = e.message ?? "构建图谱失败";
    } finally {
      building.value = false;
    }
  }

  return { overview, loading, building, error, loadOverview, buildGraph };
}
