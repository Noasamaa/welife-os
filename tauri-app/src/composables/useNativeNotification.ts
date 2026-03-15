import { isTauriRuntime } from "../services/tauri";
import type { Reminder } from "../types/reminder";

let notifiedIds = new Set<string>();

export function useNativeNotification() {
  async function notifyNewReminders(reminders: ReadonlyArray<Reminder>): Promise<void> {
    if (!isTauriRuntime()) return;

    let sendNotification: (options: { title: string; body: string }) => void;
    try {
      const mod = await import("@tauri-apps/plugin-notification");
      const { isPermissionGranted, requestPermission } = mod;
      let permitted = await isPermissionGranted();
      if (!permitted) {
        const result = await requestPermission();
        permitted = result === "granted";
      }
      if (!permitted) return;
      sendNotification = mod.sendNotification;
    } catch {
      return;
    }

    for (const r of reminders) {
      if (notifiedIds.has(r.id)) continue;
      notifiedIds.add(r.id);
      sendNotification({ title: "WeLife OS 提醒", body: r.message });
    }
  }

  function resetNotified(): void {
    notifiedIds = new Set();
  }

  return { notifyNewReminders, resetNotified };
}
