import { describe, it, expect, vi, beforeEach } from "vitest";
import { useUpdater } from "../useUpdater";
import { withSetup } from "../../test-utils/with-setup";

const mockCheck = vi.fn();

vi.mock("@tauri-apps/plugin-updater", () => ({
  check: () => mockCheck(),
}));

vi.mock("../../services/tauri", () => ({
  isTauriRuntime: () => true,
}));

describe("useUpdater", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  function setup() {
    return withSetup(() => useUpdater());
  }

  it("has correct initial state", () => {
    const { result } = setup();
    expect(result.updateAvailable.value).toBe(false);
    expect(result.updateVersion.value).toBe("");
    expect(result.checking.value).toBe(false);
    expect(result.downloading.value).toBe(false);
    expect(result.error.value).toBeNull();
  });

  it("sets updateAvailable when update found", async () => {
    const mockUpdate = { version: "2.0.0", downloadAndInstall: vi.fn() };
    mockCheck.mockResolvedValue(mockUpdate);

    const { result } = setup();
    await result.checkForUpdate();

    expect(result.updateAvailable.value).toBe(true);
    expect(result.updateVersion.value).toBe("2.0.0");
    expect(result.checking.value).toBe(false);
  });

  it("does not set updateAvailable when no update", async () => {
    mockCheck.mockResolvedValue(null);

    const { result } = setup();
    await result.checkForUpdate();

    expect(result.updateAvailable.value).toBe(false);
    expect(result.updateVersion.value).toBe("");
  });

  it("sets error on check failure", async () => {
    mockCheck.mockRejectedValue(new Error("network error"));

    const { result } = setup();
    await result.checkForUpdate();

    expect(result.error.value).toBe("network error");
    expect(result.checking.value).toBe(false);
  });

  it("downloadAndInstall calls update.downloadAndInstall", async () => {
    const mockInstall = vi.fn().mockResolvedValue(undefined);
    mockCheck.mockResolvedValue({ version: "2.0.0", downloadAndInstall: mockInstall });

    const { result } = setup();
    await result.checkForUpdate();
    await result.downloadAndInstall();

    expect(mockInstall).toHaveBeenCalled();
    expect(result.downloading.value).toBe(false);
  });

  it("downloadAndInstall sets error when no update cached", async () => {
    const { result } = setup();
    await result.downloadAndInstall();

    expect(result.error.value).toBe("没有可用的更新");
  });

  it("downloadAndInstall sets error on failure", async () => {
    const mockInstall = vi.fn().mockRejectedValue(new Error("disk full"));
    mockCheck.mockResolvedValue({ version: "2.0.0", downloadAndInstall: mockInstall });

    const { result } = setup();
    await result.checkForUpdate();
    await result.downloadAndInstall();

    expect(result.error.value).toBe("disk full");
    expect(result.downloading.value).toBe(false);
  });
});
