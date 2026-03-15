import { describe, it, expect, vi, beforeEach } from "vitest";
import { useForum } from "../useForum";
import { mockAllApi } from "../../test-utils/mock-api";
import type { ForumSession, ForumSessionDetail } from "../../types/forum";

describe("useForum", () => {
  let mocks: ReturnType<typeof mockAllApi>;

  beforeEach(() => {
    vi.restoreAllMocks();
    mocks = mockAllApi();
  });

  it("has correct initial state", () => {
    const { sessions, currentSession, loading, debating, error } = useForum();
    expect(sessions.value).toEqual([]);
    expect(currentSession.value).toBeNull();
    expect(loading.value).toBe(false);
    expect(debating.value).toBe(false);
    expect(error.value).toBeNull();
  });

  it("loadSessions fetches session list", async () => {
    const fakeSessions: ForumSession[] = [
      { id: "s1", conversation_id: "c1", task_id: "t1", status: "completed", summary: "ok", created_at: "" },
    ];
    mocks.fetchForumSessions.mockResolvedValue(fakeSessions);

    const { loadSessions, sessions, loading } = useForum();
    const promise = loadSessions();
    expect(loading.value).toBe(true);
    await promise;

    expect(sessions.value).toEqual(fakeSessions);
    expect(loading.value).toBe(false);
  });

  it("loadSessions sets error on failure", async () => {
    mocks.fetchForumSessions.mockRejectedValue(new Error("fail"));

    const { loadSessions, error } = useForum();
    await loadSessions();

    expect(error.value).toBe("fail");
  });

  it("loadSession fetches session detail", async () => {
    const fakeDetail: ForumSessionDetail = {
      session: { id: "s1", conversation_id: "c1", task_id: "t1", status: "completed", summary: "ok", created_at: "" },
      messages: [],
    };
    mocks.fetchForumSession.mockResolvedValue(fakeDetail);

    const { loadSession, currentSession } = useForum();
    await loadSession("s1");

    expect(currentSession.value).toEqual(fakeDetail);
  });

  it("startDebate triggers debate and loads both sessions and detail", async () => {
    const debateResult = { session_id: "s2", task_id: "t2" };
    const fakeSessions: ForumSession[] = [];
    const fakeDetail: ForumSessionDetail = {
      session: { id: "s2", conversation_id: "c1", task_id: "t2", status: "running", summary: "", created_at: "" },
      messages: [],
    };

    mocks.triggerDebate.mockResolvedValue(debateResult);
    mocks.fetchForumSessions.mockResolvedValue(fakeSessions);
    mocks.fetchForumSession.mockResolvedValue(fakeDetail);

    const { startDebate, debating } = useForum();
    const promise = startDebate("c1");
    expect(debating.value).toBe(true);
    const result = await promise;

    expect(result).toEqual(debateResult);
    expect(debating.value).toBe(false);
    expect(mocks.triggerDebate).toHaveBeenCalledWith("c1");
    expect(mocks.fetchForumSessions).toHaveBeenCalled();
    expect(mocks.fetchForumSession).toHaveBeenCalledWith("s2");
  });

  it("startDebate sets error on failure", async () => {
    mocks.triggerDebate.mockRejectedValue(new Error("debate fail"));

    const { startDebate, error, debating } = useForum();
    const result = await startDebate("c1");

    expect(result).toBeNull();
    expect(error.value).toBe("debate fail");
    expect(debating.value).toBe(false);
  });
});
