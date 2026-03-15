import { describe, it, expect, vi, beforeEach } from "vitest";
import { useCoach } from "../useCoach";
import { mockAllApi } from "../../test-utils/mock-api";
import type { ActionItem, ActionPlanResponse } from "../../types/coach";

describe("useCoach", () => {
  let mocks: ReturnType<typeof mockAllApi>;

  beforeEach(() => {
    vi.restoreAllMocks();
    mocks = mockAllApi();
  });

  it("has correct initial state", () => {
    const { items, loading, generating, error } = useCoach();
    expect(items.value).toEqual([]);
    expect(loading.value).toBe(false);
    expect(generating.value).toBe(false);
    expect(error.value).toBeNull();
  });

  it("loadItems fetches action items", async () => {
    const fakeItems: ActionItem[] = [
      { id: "a1", source_agent: "coach", source_session_id: "s1", title: "Task 1", description: "desc", priority: "high", status: "pending", category: "project", due_date: "", created_at: "" },
    ];
    mocks.fetchActionItems.mockResolvedValue(fakeItems);

    const { loadItems, items, loading } = useCoach();
    const promise = loadItems("pending", "project");
    expect(loading.value).toBe(true);
    await promise;

    expect(items.value).toEqual(fakeItems);
    expect(loading.value).toBe(false);
    expect(mocks.fetchActionItems).toHaveBeenCalledWith("pending", "project");
  });

  it("loadItems sets error on failure", async () => {
    mocks.fetchActionItems.mockRejectedValue(new Error("load fail"));

    const { loadItems, error } = useCoach();
    await loadItems();

    expect(error.value).toBe("load fail");
  });

  it("generate creates action plan and reloads items", async () => {
    const fakeResponse: ActionPlanResponse = {
      items: [{ id: "a2", source_agent: "coach", source_session_id: "s1", title: "New", description: "d", priority: "medium", status: "pending", category: "contact", due_date: "", created_at: "" }],
      count: 1,
    };
    mocks.generateActionPlan.mockResolvedValue(fakeResponse);
    mocks.fetchActionItems.mockResolvedValue(fakeResponse.items);

    const { generate, generating } = useCoach();
    const promise = generate("session1");
    expect(generating.value).toBe(true);
    const result = await promise;

    expect(result).toEqual(fakeResponse);
    expect(generating.value).toBe(false);
    expect(mocks.generateActionPlan).toHaveBeenCalledWith("session1");
  });

  it("generate sets error on failure", async () => {
    mocks.generateActionPlan.mockRejectedValue(new Error("gen fail"));

    const { generate, error } = useCoach();
    const result = await generate("s1");

    expect(result).toBeNull();
    expect(error.value).toBe("gen fail");
  });

  it("updateStatus calls API and reloads items", async () => {
    mocks.updateActionItemStatus.mockResolvedValue(undefined);
    mocks.fetchActionItems.mockResolvedValue([]);

    const { updateStatus } = useCoach();
    await updateStatus("a1", "completed");

    expect(mocks.updateActionItemStatus).toHaveBeenCalledWith("a1", "completed");
    expect(mocks.fetchActionItems).toHaveBeenCalled();
  });

  it("remove calls API and reloads items", async () => {
    mocks.deleteActionItem.mockResolvedValue(undefined);
    mocks.fetchActionItems.mockResolvedValue([]);

    const { remove } = useCoach();
    await remove("a1");

    expect(mocks.deleteActionItem).toHaveBeenCalledWith("a1");
    expect(mocks.fetchActionItems).toHaveBeenCalled();
  });

  it("updateStatus sets error on failure", async () => {
    mocks.updateActionItemStatus.mockRejectedValue(new Error("update fail"));

    const { updateStatus, error } = useCoach();
    await updateStatus("a1", "completed");

    expect(error.value).toBe("update fail");
  });
});
