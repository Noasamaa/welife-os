import { describe, it, expect, vi, beforeEach } from "vitest";
import { useReminder } from "../useReminder";
import { mockAllApi } from "../../test-utils/mock-api";
import type { Reminder, ReminderRule } from "../../types/reminder";
import { withSetup } from "../../test-utils/with-setup";

const mockSetTrayBadge = vi.fn();
const mockNotifyNewReminders = vi.fn();

vi.mock("../../services/tauri", () => ({
  isTauriRuntime: () => false,
  setTrayBadge: (...args: unknown[]) => mockSetTrayBadge(...args),
}));

vi.mock("../useNativeNotification", () => ({
  useNativeNotification: () => ({
    notifyNewReminders: (...args: unknown[]) => mockNotifyNewReminders(...args),
    resetNotified: vi.fn(),
  }),
}));

describe("useReminder", () => {
  let mocks: ReturnType<typeof mockAllApi>;

  beforeEach(() => {
    vi.restoreAllMocks();
    mockSetTrayBadge.mockClear();
    mockNotifyNewReminders.mockClear();
    mocks = mockAllApi();
  });

  function setup() {
    return withSetup(() => useReminder());
  }

  it("has correct initial state", () => {
    const { result } = setup();
    expect(result.pending.value).toEqual([]);
    expect(result.rules.value).toEqual([]);
    expect(result.loading.value).toBe(false);
    expect(result.error.value).toBeNull();
  });

  it("loadPending fetches pending reminders", async () => {
    const fakeReminders: Reminder[] = [
      { id: "r1", rule_id: "rule1", message: "Contact Alice", status: "pending", triggered_at: "2026-01-01", read_at: undefined },
    ];
    mocks.fetchPendingReminders.mockResolvedValue(fakeReminders);

    const { result } = setup();
    await result.loadPending();

    expect(result.pending.value).toEqual(fakeReminders);
  });

  it("loadPending sets error on failure", async () => {
    mocks.fetchPendingReminders.mockRejectedValue(new Error("fail"));

    const { result } = setup();
    await result.loadPending();

    expect(result.error.value).toBe("fail");
  });

  it("loadRules fetches reminder rules", async () => {
    const fakeRules: ReminderRule[] = [
      { id: "rule1", action_item_id: "", rule_type: "contact_gap", threshold_days: 7, cron_expr: "", message_template: "msg", enabled: true, created_at: "", last_triggered_at: undefined },
    ];
    mocks.fetchReminderRules.mockResolvedValue(fakeRules);

    const { result } = setup();
    await result.loadRules();

    expect(result.rules.value).toEqual(fakeRules);
    expect(result.loading.value).toBe(false);
  });

  it("read marks reminder and reloads pending", async () => {
    mocks.markReminderRead.mockResolvedValue(undefined);
    mocks.fetchPendingReminders.mockResolvedValue([]);

    const { result } = setup();
    await result.read("r1");

    expect(mocks.markReminderRead).toHaveBeenCalledWith("r1");
    expect(mocks.fetchPendingReminders).toHaveBeenCalled();
  });

  it("dismiss dismisses reminder and reloads pending", async () => {
    mocks.dismissReminder.mockResolvedValue(undefined);
    mocks.fetchPendingReminders.mockResolvedValue([]);

    const { result } = setup();
    await result.dismiss("r1");

    expect(mocks.dismissReminder).toHaveBeenCalledWith("r1");
    expect(mocks.fetchPendingReminders).toHaveBeenCalled();
  });

  it("addRule creates rule and reloads rules list", async () => {
    const newRule = { action_item_id: "", rule_type: "deadline" as const, threshold_days: 3, cron_expr: "", message_template: "due", enabled: true };
    mocks.createReminderRule.mockResolvedValue({ ...newRule, id: "rule2", created_at: "", last_triggered_at: undefined });
    mocks.fetchReminderRules.mockResolvedValue([]);

    const { result } = setup();
    await result.addRule(newRule);

    expect(mocks.createReminderRule).toHaveBeenCalledWith(newRule);
    expect(mocks.fetchReminderRules).toHaveBeenCalled();
  });

  it("toggleRule updates rule and reloads", async () => {
    mocks.updateReminderRule.mockResolvedValue(undefined);
    mocks.fetchReminderRules.mockResolvedValue([]);

    const { result } = setup();
    await result.toggleRule("rule1", false);

    expect(mocks.updateReminderRule).toHaveBeenCalledWith("rule1", false);
    expect(mocks.fetchReminderRules).toHaveBeenCalled();
  });

  it("removeRule deletes rule and reloads", async () => {
    mocks.deleteReminderRule.mockResolvedValue(undefined);
    mocks.fetchReminderRules.mockResolvedValue([]);

    const { result } = setup();
    await result.removeRule("rule1");

    expect(mocks.deleteReminderRule).toHaveBeenCalledWith("rule1");
    expect(mocks.fetchReminderRules).toHaveBeenCalled();
  });

  it("startPolling calls loadPending immediately and on interval", async () => {
    vi.useFakeTimers();
    mocks.fetchPendingReminders.mockResolvedValue([]);

    const { result } = setup();
    result.startPolling(1000);

    // Initial call
    expect(mocks.fetchPendingReminders).toHaveBeenCalledTimes(1);

    // Advance timer
    vi.advanceTimersByTime(1000);
    expect(mocks.fetchPendingReminders).toHaveBeenCalledTimes(2);

    result.stopPolling();
    vi.advanceTimersByTime(1000);
    expect(mocks.fetchPendingReminders).toHaveBeenCalledTimes(2);

    vi.useRealTimers();
  });

  it("stopPolling clears interval", () => {
    vi.useFakeTimers();
    mocks.fetchPendingReminders.mockResolvedValue([]);

    const { result } = setup();
    result.startPolling(500);
    result.stopPolling();

    vi.advanceTimersByTime(2000);
    // Only the initial loadPending call, no interval calls
    expect(mocks.fetchPendingReminders).toHaveBeenCalledTimes(1);

    vi.useRealTimers();
  });

  it("loadPending calls setTrayBadge with pending count", async () => {
    const fakeReminders: Reminder[] = [
      { id: "r1", rule_id: "rule1", message: "Test", status: "pending", triggered_at: "2026-01-01", read_at: undefined },
      { id: "r2", rule_id: "rule1", message: "Test 2", status: "pending", triggered_at: "2026-01-01", read_at: undefined },
    ];
    mocks.fetchPendingReminders.mockResolvedValue(fakeReminders);

    const { result } = setup();
    await result.loadPending();

    expect(mockSetTrayBadge).toHaveBeenCalledWith(2);
  });

  it("loadPending calls notifyNewReminders with pending list", async () => {
    const fakeReminders: Reminder[] = [
      { id: "r1", rule_id: "rule1", message: "Test", status: "pending", triggered_at: "2026-01-01", read_at: undefined },
    ];
    mocks.fetchPendingReminders.mockResolvedValue(fakeReminders);

    const { result } = setup();
    await result.loadPending();

    expect(mockNotifyNewReminders).toHaveBeenCalledWith(fakeReminders);
  });
});
