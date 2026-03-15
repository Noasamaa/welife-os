import { describe, it, expect } from "vitest";
import { useAppMeta } from "../useAppMeta";

describe("useAppMeta", () => {
  it("returns app name, phase, and slogan", () => {
    const meta = useAppMeta();
    expect(meta.name).toBe("WeLife OS");
    expect(meta.phase).toBe("Phase 0");
    expect(meta.slogan).toContain("聊天记录");
  });
});
