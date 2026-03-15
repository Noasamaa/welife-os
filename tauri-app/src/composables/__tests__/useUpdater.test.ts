import { describe, it, expect, beforeEach } from "vitest";
import { useUpdater } from "../useUpdater";
import { withSetup } from "../../test-utils/with-setup";

describe("useUpdater", () => {
  beforeEach(() => {
    // no-op: composable now has pure local state
  });

  function setup() {
    return withSetup(() => useUpdater());
  }

  it("starts in disabled mode", () => {
    const { result } = setup();
    expect(result.enabled.value).toBe(false);
    expect(result.updateAvailable.value).toBe(false);
    expect(result.error.value).toContain("未启用签名更新");
  });

  it("checkForUpdate keeps disabled reason", async () => {
    const { result } = setup();
    await result.checkForUpdate();
    expect(result.error.value).toContain("未启用签名更新");
  });

  it("downloadAndInstall keeps disabled reason", async () => {
    const { result } = setup();
    await result.downloadAndInstall();
    expect(result.error.value).toContain("未启用签名更新");
  });
});
