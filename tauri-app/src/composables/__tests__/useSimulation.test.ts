import { describe, it, expect, vi, beforeEach } from "vitest";
import { useSimulation } from "../useSimulation";
import { mockAllApi } from "../../test-utils/mock-api";
import type { PersonProfile, SimulationSession, SimulationDetail } from "../../types/simulation";

describe("useSimulation", () => {
  let mocks: ReturnType<typeof mockAllApi>;

  beforeEach(() => {
    vi.restoreAllMocks();
    mocks = mockAllApi();
  });

  it("has correct initial state", () => {
    const { profiles, sessions, currentSession, loading, running, building, error } = useSimulation();
    expect(profiles.value).toEqual([]);
    expect(sessions.value).toEqual([]);
    expect(currentSession.value).toBeNull();
    expect(loading.value).toBe(false);
    expect(running.value).toBe(false);
    expect(building.value).toBe(false);
    expect(error.value).toBeNull();
  });

  it("loadProfiles fetches person profiles", async () => {
    const fakeProfiles: PersonProfile[] = [
      { id: "p1", entity_id: "e1", name: "Alice", personality: "kind", relationship_to_self: "friend", behavioral_patterns: "regular", created_at: "", updated_at: "" },
    ];
    mocks.fetchProfiles.mockResolvedValue(fakeProfiles);

    const { loadProfiles, profiles, loading } = useSimulation();
    const promise = loadProfiles();
    expect(loading.value).toBe(true);
    await promise;

    expect(profiles.value).toEqual(fakeProfiles);
    expect(loading.value).toBe(false);
  });

  it("loadProfiles sets error on failure", async () => {
    mocks.fetchProfiles.mockRejectedValue(new Error("fail"));

    const { loadProfiles, error } = useSimulation();
    await loadProfiles();

    expect(error.value).toBe("fail");
  });

  it("buildAllProfiles triggers build", async () => {
    mocks.buildProfiles.mockResolvedValue({ task_id: "t1" });

    const { buildAllProfiles, building } = useSimulation();
    const promise = buildAllProfiles();
    expect(building.value).toBe(true);
    const result = await promise;

    expect(result).toEqual({ task_id: "t1" });
    expect(building.value).toBe(false);
  });

  it("buildAllProfiles sets error on failure", async () => {
    mocks.buildProfiles.mockRejectedValue(new Error("build fail"));

    const { buildAllProfiles, error, building } = useSimulation();
    const result = await buildAllProfiles();

    expect(result).toBeNull();
    expect(error.value).toBe("build fail");
    expect(building.value).toBe(false);
  });

  it("loadSessions fetches simulation session list", async () => {
    const fakeSessions: SimulationSession[] = [
      { id: "sim1", task_id: "t1", fork_description: "test", status: "completed", step_count: 5, original_graph_snapshot: "", final_graph_snapshot: "", narrative: "", created_at: "" },
    ];
    mocks.fetchSimulations.mockResolvedValue(fakeSessions);

    const { loadSessions, sessions } = useSimulation();
    await loadSessions();

    expect(sessions.value).toEqual(fakeSessions);
  });

  it("loadSession fetches simulation detail", async () => {
    const fakeDetail: SimulationDetail = {
      session: { id: "sim1", task_id: "t1", fork_description: "test", status: "completed", step_count: 5, original_graph_snapshot: "", final_graph_snapshot: "", narrative: "", created_at: "" },
      steps: [],
    };
    mocks.fetchSimulation.mockResolvedValue(fakeDetail);

    const { loadSession, currentSession } = useSimulation();
    await loadSession("sim1");

    expect(currentSession.value).toEqual(fakeDetail);
  });

  it("startSimulation triggers run and loads both sessions and detail", async () => {
    const simResult = { session_id: "sim2", task_id: "t2" };
    mocks.runSimulation.mockResolvedValue(simResult);
    mocks.fetchSimulations.mockResolvedValue([]);
    mocks.fetchSimulation.mockResolvedValue({ session: { id: "sim2", task_id: "t2", fork_description: "", status: "running", step_count: 0, created_at: "" }, steps: [] });

    const { startSimulation, running } = useSimulation();
    const promise = startSimulation("what if", ["Alice"], { mood: "happy" }, 3);
    expect(running.value).toBe(true);
    const result = await promise;

    expect(result).toEqual(simResult);
    expect(running.value).toBe(false);
    expect(mocks.runSimulation).toHaveBeenCalledWith("what if", ["Alice"], { mood: "happy" }, 3);
    expect(mocks.fetchSimulations).toHaveBeenCalled();
    expect(mocks.fetchSimulation).toHaveBeenCalledWith("sim2");
  });

  it("startSimulation sets error on failure", async () => {
    mocks.runSimulation.mockRejectedValue(new Error("sim fail"));

    const { startSimulation, error, running } = useSimulation();
    const result = await startSimulation("what if", [], {});

    expect(result).toBeNull();
    expect(error.value).toBe("sim fail");
    expect(running.value).toBe(false);
  });
});
