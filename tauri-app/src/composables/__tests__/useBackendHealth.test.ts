import { describe, it, expect, vi, beforeEach } from "vitest";
import { mockAllApi } from "../../test-utils/mock-api";
import type { SystemStatusResponse } from "../../types/api";

// useBackendHealth uses module-level state, so we need to isolate each test
// by dynamically importing the module after resetting mocks.

function makeStatus(overrides?: Partial<SystemStatusResponse>): SystemStatusResponse {
  return {
    backend: { status: "ok", version: "1.0.0" },
    storage: { driver: "sqlcipher", ready: true, path: "/tmp/test.db" },
    llm: { provider: "ollama", reachable: true, base_url: "http://localhost:11434", model: "qwen3.5:9b" },
    ...overrides,
  };
}

describe("useBackendHealth", () => {
  let mocks: ReturnType<typeof mockAllApi>;

  beforeEach(() => {
    vi.restoreAllMocks();
    mocks = mockAllApi();
  });

  it("checkHealth sets healthy status on success", async () => {
    mocks.fetchSystemStatus.mockResolvedValue(makeStatus());

    // Import dynamically to get fresh module-level state
    const { useBackendHealth } = await import("../useBackendHealth");
    const { checkHealth, status, systemStatus, errorMessage } = useBackendHealth();

    await checkHealth();

    expect(status.value).toBe("healthy");
    expect(systemStatus.value).toBeDefined();
    expect(systemStatus.value?.backend.status).toBe("ok");
    expect(errorMessage.value).toBeNull();
  });

  it("checkHealth sets unreachable on API error", async () => {
    mocks.fetchSystemStatus.mockRejectedValue(new Error("connection refused"));

    const { useBackendHealth } = await import("../useBackendHealth");
    const { checkHealth, status, errorMessage } = useBackendHealth();

    await checkHealth();

    expect(status.value).toBe("unreachable");
    expect(errorMessage.value).toBe("connection refused");
  });

  it("checkHealth sets unreachable when backend status is not ok", async () => {
    mocks.fetchSystemStatus.mockResolvedValue(
      makeStatus({ backend: { status: "degraded", version: "1.0.0" } }),
    );

    const { useBackendHealth } = await import("../useBackendHealth");
    const { checkHealth, status } = useBackendHealth();

    await checkHealth();

    expect(status.value).toBe("unreachable");
  });

  it("exposes apiBaseUrl", async () => {
    const { useBackendHealth } = await import("../useBackendHealth");
    const { apiBaseUrl } = useBackendHealth();

    expect(typeof apiBaseUrl).toBe("string");
  });

  it("auto-checks health on mount and starts polling", async () => {
    vi.useFakeTimers();
    mocks.fetchSystemStatus.mockResolvedValue(makeStatus());

    const { useBackendHealth } = await import("../useBackendHealth");
    const { withSetup } = await import("../../test-utils/with-setup");
    const { result, unmount } = withSetup(() => useBackendHealth());

    // onMounted fires checkHealth once — flush microtasks
    await vi.advanceTimersByTimeAsync(0);
    expect(mocks.fetchSystemStatus).toHaveBeenCalledTimes(1);
    expect(result.status.value).toBe("healthy");

    // Advance 5s — poll fires again
    await vi.advanceTimersByTimeAsync(5000);
    expect(mocks.fetchSystemStatus).toHaveBeenCalledTimes(2);

    unmount();
    vi.useRealTimers();
  });
});
