import { describe, it, expect, vi, beforeEach } from "vitest";
import { useGraph } from "../useGraph";
import { mockAllApi } from "../../test-utils/mock-api";
import type { GraphOverview } from "../../types/import";

describe("useGraph", () => {
  let mocks: ReturnType<typeof mockAllApi>;

  beforeEach(() => {
    vi.restoreAllMocks();
    mocks = mockAllApi();
  });

  it("has correct initial state", () => {
    const { overview, loading, building, error } = useGraph();
    expect(overview.value).toBeNull();
    expect(loading.value).toBe(false);
    expect(building.value).toBe(false);
    expect(error.value).toBeNull();
  });

  it("loadOverview fetches overview data", async () => {
    const fakeOverview: GraphOverview = {
      nodes: [{ id: "n1", type: "person", name: "Alice" }],
      edges: [],
      stats: { entity_count: 1, relationship_count: 0, entity_types: {} },
    };
    mocks.fetchGraphOverview.mockResolvedValue(fakeOverview);

    const { loadOverview, overview, loading } = useGraph();
    const promise = loadOverview();
    expect(loading.value).toBe(true);
    await promise;

    expect(overview.value).toEqual(fakeOverview);
    expect(loading.value).toBe(false);
  });

  it("loadOverview sets error on failure", async () => {
    mocks.fetchGraphOverview.mockRejectedValue(new Error("timeout"));

    const { loadOverview, error, loading } = useGraph();
    await loadOverview();

    expect(error.value).toBe("timeout");
    expect(loading.value).toBe(false);
  });

  it("buildGraph triggers build and returns task_id", async () => {
    mocks.triggerGraphBuild.mockResolvedValue({ task_id: "t1" });

    const { buildGraph, building } = useGraph();
    const promise = buildGraph("conv1");
    expect(building.value).toBe(true);
    const result = await promise;

    expect(result).toEqual({ task_id: "t1" });
    expect(building.value).toBe(false);
    expect(mocks.triggerGraphBuild).toHaveBeenCalledWith("conv1");
  });

  it("buildGraph sets error on failure", async () => {
    mocks.triggerGraphBuild.mockRejectedValue(new Error("build failed"));

    const { buildGraph, error, building } = useGraph();
    const result = await buildGraph("conv1");

    expect(result).toBeNull();
    expect(error.value).toBe("build failed");
    expect(building.value).toBe(false);
  });
});
