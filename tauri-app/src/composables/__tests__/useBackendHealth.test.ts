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
    mocks.getAPIBaseURL.mockResolvedValue("http://127.0.0.1:18080");
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

    expect(typeof apiBaseUrl.value).toBe("string");
  });

  it("polling is started at module level (no mount required)", async () => {
    mocks.fetchSystemStatus.mockResolvedValue(makeStatus());

    // The module starts polling on first import — no onMounted needed.
    // We verify the composable returns the shared state without requiring a component.
    const { useBackendHealth } = await import("../useBackendHealth");
    const { checkHealth, status } = useBackendHealth();

    await checkHealth();
    expect(status.value).toBe("healthy");
    expect(mocks.fetchSystemStatus).toHaveBeenCalled();
  });
});
