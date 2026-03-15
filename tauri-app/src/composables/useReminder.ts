import { ref, onUnmounted } from "vue";
import type { Reminder, ReminderRule } from "../types/reminder";
import {
  fetchPendingReminders,
  markReminderRead,
  dismissReminder,
  fetchReminderRules,
  createReminderRule,
  updateReminderRule,
  deleteReminderRule,
} from "../services/api";

export function useReminder() {
  const pending = ref<Reminder[]>([]);
  const rules = ref<ReminderRule[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);
  let pollHandle: ReturnType<typeof setInterval> | null = null;

  async function loadPending() {
    try {
      pending.value = await fetchPendingReminders();
    } catch (e: any) {
      error.value = e.message ?? "加载提醒失败";
    }
  }

  async function loadRules() {
    loading.value = true;
    error.value = null;
    try {
      rules.value = await fetchReminderRules();
    } catch (e: any) {
      error.value = e.message ?? "加载提醒规则失败";
    } finally {
      loading.value = false;
    }
  }

  async function read(id: string) {
    try {
      await markReminderRead(id);
      await loadPending();
    } catch (e: any) {
      error.value = e.message ?? "标记已读失败";
    }
  }

  async function dismiss(id: string) {
    try {
      await dismissReminder(id);
      await loadPending();
    } catch (e: any) {
      error.value = e.message ?? "取消提醒失败";
    }
  }

  async function addRule(rule: Omit<ReminderRule, "id" | "created_at" | "last_triggered_at">) {
    error.value = null;
    try {
      await createReminderRule(rule);
      await loadRules();
    } catch (e: any) {
      error.value = e.message ?? "创建规则失败";
    }
  }

  async function toggleRule(id: string, enabled: boolean) {
    try {
      await updateReminderRule(id, enabled);
      await loadRules();
    } catch (e: any) {
      error.value = e.message ?? "更新规则失败";
    }
  }

  async function removeRule(id: string) {
    try {
      await deleteReminderRule(id);
      await loadRules();
    } catch (e: any) {
      error.value = e.message ?? "删除规则失败";
    }
  }

  function startPolling(intervalMs = 60000) {
    stopPolling();
    void loadPending();
    pollHandle = setInterval(() => void loadPending(), intervalMs);
  }

  function stopPolling() {
    if (pollHandle !== null) {
      clearInterval(pollHandle);
      pollHandle = null;
    }
  }

  onUnmounted(() => stopPolling());

  return {
    pending, rules, loading, error,
    loadPending, loadRules, read, dismiss,
    addRule, toggleRule, removeRule,
    startPolling, stopPolling,
  };
}
