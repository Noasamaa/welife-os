import { describe, it, expect, vi, beforeEach } from "vitest";
import { useNativeNotification } from "../useNativeNotification";
import type { Reminder } from "../../types/reminder";

const mockSendNotification = vi.fn();
const mockIsPermissionGranted = vi.fn();
const mockRequestPermission = vi.fn();

vi.mock("@tauri-apps/plugin-notification", () => ({
  sendNotification: (...args: unknown[]) => mockSendNotification(...args),
  isPermissionGranted: () => mockIsPermissionGranted(),
  requestPermission: () => mockRequestPermission(),
}));

vi.mock("../../services/tauri", () => ({
  isTauriRuntime: () => true,
}));

function fakeReminder(id: string, message: string): Reminder {
  return {
    id,
    rule_id: "rule1",
    message,
    status: "pending",
    triggered_at: "2026-01-01",
    read_at: undefined,
  };
}

describe("useNativeNotification", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockIsPermissionGranted.mockResolvedValue(true);
    const { resetNotified } = useNativeNotification();
    resetNotified();
  });

  it("sends notification for new reminders", async () => {
    const { notifyNewReminders } = useNativeNotification();
    const reminders = [fakeReminder("r1", "Call Alice")];

    await notifyNewReminders(reminders);

    expect(mockSendNotification).toHaveBeenCalledWith({
      title: "WeLife OS 提醒",
      body: "Call Alice",
    });
  });

  it("deduplicates already-notified reminders", async () => {
    const { notifyNewReminders } = useNativeNotification();
    const reminders = [fakeReminder("r1", "Call Alice")];

    await notifyNewReminders(reminders);
    await notifyNewReminders(reminders);

    expect(mockSendNotification).toHaveBeenCalledTimes(1);
  });

  it("sends notification for new items only in mixed batch", async () => {
    const { notifyNewReminders } = useNativeNotification();

    await notifyNewReminders([fakeReminder("r1", "Call Alice")]);
    mockSendNotification.mockClear();

    await notifyNewReminders([
      fakeReminder("r1", "Call Alice"),
      fakeReminder("r2", "Call Bob"),
    ]);

    expect(mockSendNotification).toHaveBeenCalledTimes(1);
    expect(mockSendNotification).toHaveBeenCalledWith({
      title: "WeLife OS 提醒",
      body: "Call Bob",
    });
  });

  it("requests permission if not granted", async () => {
    mockIsPermissionGranted.mockResolvedValue(false);
    mockRequestPermission.mockResolvedValue("granted");

    const { notifyNewReminders } = useNativeNotification();
    await notifyNewReminders([fakeReminder("r1", "Test")]);

    expect(mockRequestPermission).toHaveBeenCalled();
    expect(mockSendNotification).toHaveBeenCalled();
  });

  it("skips notifications when permission denied", async () => {
    mockIsPermissionGranted.mockResolvedValue(false);
    mockRequestPermission.mockResolvedValue("denied");

    const { notifyNewReminders } = useNativeNotification();
    await notifyNewReminders([fakeReminder("r1", "Test")]);

    expect(mockSendNotification).not.toHaveBeenCalled();
  });

  it("resetNotified clears dedup set", async () => {
    const { notifyNewReminders, resetNotified } = useNativeNotification();
    const reminders = [fakeReminder("r1", "Call Alice")];

    await notifyNewReminders(reminders);
    resetNotified();
    await notifyNewReminders(reminders);

    expect(mockSendNotification).toHaveBeenCalledTimes(2);
  });
});
