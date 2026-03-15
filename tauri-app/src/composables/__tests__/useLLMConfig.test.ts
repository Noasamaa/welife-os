import { describe, it, expect, vi, beforeEach } from "vitest";
import { useLLMConfig } from "../useLLMConfig";
import { mockAllApi } from "../../test-utils/mock-api";
import { withSetup } from "../../test-utils/with-setup";
import type { LLMConfig } from "../../types/api";

describe("useLLMConfig", () => {
  let mocks: ReturnType<typeof mockAllApi>;

  beforeEach(() => {
    vi.restoreAllMocks();
    mocks = mockAllApi();
  });

  function setup() {
    return withSetup(() => useLLMConfig());
  }

  const fakeConfig: LLMConfig = {
    provider: "ollama",
    base_url: "http://127.0.0.1:11434",
    model: "qwen3.5:9b",
    api_key: "",
    embedding_model: "",
  };

  it("has correct initial state", () => {
    const { result } = setup();
    expect(result.config.value).toBeNull();
    expect(result.loading.value).toBe(false);
    expect(result.saving.value).toBe(false);
    expect(result.saveError.value).toBeNull();
    expect(result.saveSuccess.value).toBe(false);
  });

  it("load fetches LLM config", async () => {
    mocks.fetchLLMConfig.mockResolvedValue(fakeConfig);

    const { result } = setup();
    await result.load();

    expect(result.config.value).toEqual(fakeConfig);
    expect(result.loading.value).toBe(false);
  });

  it("load sets error on failure", async () => {
    mocks.fetchLLMConfig.mockRejectedValue(new Error("network error"));

    const { result } = setup();
    await result.load();

    expect(result.saveError.value).toBe("network error");
  });

  it("save sends patch and reloads config", async () => {
    mocks.updateLLMConfig.mockResolvedValue(undefined);
    mocks.fetchLLMConfig.mockResolvedValue(fakeConfig);

    const { result } = setup();
    await result.save({ model: "gpt-4" });

    expect(mocks.updateLLMConfig).toHaveBeenCalledWith({ model: "gpt-4" });
    expect(mocks.fetchLLMConfig).toHaveBeenCalled();
    expect(result.saveSuccess.value).toBe(true);
  });

  it("save sets error on failure", async () => {
    mocks.updateLLMConfig.mockRejectedValue(new Error("save failed"));

    const { result } = setup();
    await result.save({ model: "gpt-4" });

    expect(result.saveError.value).toBe("save failed");
    expect(result.saveSuccess.value).toBe(false);
  });

  it("save resets error from previous attempt", async () => {
    mocks.updateLLMConfig.mockRejectedValueOnce(new Error("first error"));

    const { result } = setup();
    await result.save({ model: "a" });
    expect(result.saveError.value).toBe("first error");

    mocks.updateLLMConfig.mockResolvedValueOnce(undefined);
    mocks.fetchLLMConfig.mockResolvedValueOnce(fakeConfig);
    await result.save({ model: "b" });
    expect(result.saveError.value).toBeNull();
    expect(result.saveSuccess.value).toBe(true);
  });
});
