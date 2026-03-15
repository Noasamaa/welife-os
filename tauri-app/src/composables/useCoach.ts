import { ref } from "vue";
import type { ActionItem } from "../types/coach";
import {
  generateActionPlan,
  fetchActionItems,
  updateActionItemStatus,
  deleteActionItem,
} from "../services/api";

export function useCoach() {
  const items = ref<ActionItem[]>([]);
  const loading = ref(false);
  const generating = ref(false);
  const error = ref<string | null>(null);

  async function loadItems(status?: string, category?: string) {
    loading.value = true;
    error.value = null;
    try {
      items.value = await fetchActionItems(status, category);
    } catch (e: any) {
      error.value = e.message ?? "加载行动项失败";
    } finally {
      loading.value = false;
    }
  }

  async function generate(sessionID: string) {
    generating.value = true;
    error.value = null;
    try {
      const result = await generateActionPlan(sessionID);
      await loadItems();
      return result;
    } catch (e: any) {
      error.value = e.message ?? "生成行动计划失败";
      return null;
    } finally {
      generating.value = false;
    }
  }

  async function updateStatus(id: string, status: string) {
    error.value = null;
    try {
      await updateActionItemStatus(id, status);
      await loadItems();
    } catch (e: any) {
      error.value = e.message ?? "更新状态失败";
    }
  }

  async function remove(id: string) {
    error.value = null;
    try {
      await deleteActionItem(id);
      await loadItems();
    } catch (e: any) {
      error.value = e.message ?? "删除失败";
    }
  }

  return { items, loading, generating, error, loadItems, generate, updateStatus, remove };
}
