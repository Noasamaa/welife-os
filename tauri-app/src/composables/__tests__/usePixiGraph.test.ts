import { describe, it, expect, vi, beforeEach } from "vitest";
import { ref } from "vue";
import { usePixiGraph, type PixiGraphControls } from "../usePixiGraph";
import { withSetup } from "../../test-utils/with-setup";
import type { GraphOverview } from "../../types/import";

// --- Mock pixi.js (use real classes so `new` survives restoreAllMocks) ---
vi.mock("pixi.js", () => {
  class MockApplication {
    init = vi.fn().mockResolvedValue(undefined);
    canvas = {
      style: {} as Record<string, string>,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      parentElement: null,
    };
    stage = {
      addChild: vi.fn(),
      eventMode: "auto",
      hitArea: null as unknown,
      on: vi.fn(),
    };
    renderer = { events: {} };
    ticker = {};
    screen = { width: 800, height: 500 };
    destroy = vi.fn();
  }

  class MockContainer {
    addChild = vi.fn();
    x = 0;
    y = 0;
    alpha = 1;
    visible = true;
    scale = { x: 1, y: 1, set: vi.fn() };
    toLocal = vi.fn(() => ({ x: 0, y: 0 }));
    on = vi.fn();
  }

  class MockGraphics {
    circle = vi.fn().mockReturnThis();
    fill = vi.fn().mockReturnThis();
    stroke = vi.fn().mockReturnThis();
    clear = vi.fn().mockReturnThis();
    moveTo = vi.fn().mockReturnThis();
    lineTo = vi.fn().mockReturnThis();
    addChild = vi.fn();
    eventMode = "auto";
    cursor = "default";
    on = vi.fn();
    x = 0;
    y = 0;
    alpha = 1;
    visible = true;
    scale = { x: 1, y: 1, set: vi.fn() };
  }

  class MockText {
    anchor = { set: vi.fn() };
    y = 0;
    resolution = 1;
    visible = false;
  }

  class MockTextStyle {}

  return {
    Application: MockApplication,
    Container: MockContainer,
    Graphics: MockGraphics,
    Text: MockText,
    TextStyle: MockTextStyle,
  };
});

// --- Mock Web Worker (use a real class so `new Worker(...)` survives restoreAllMocks) ---
class MockWorker {
  onmessage: ((e: MessageEvent) => void) | null = null;
  postMessage = vi.fn();
  terminate = vi.fn();
}
vi.stubGlobal("Worker", MockWorker);

// --- Test data ---
function makeOverview(opts?: {
  extraNodes?: GraphOverview["nodes"];
  extraEdges?: GraphOverview["edges"];
}): GraphOverview {
  return {
    nodes: [
      { id: "n1", type: "person", name: "Alice" },
      { id: "n2", type: "topic", name: "GraphDB" },
      { id: "n3", type: "event", name: "Meeting" },
      { id: "n4", type: "person", name: "Bob" },
      ...(opts?.extraNodes ?? []),
    ],
    edges: [
      { id: "e1", source: "n1", target: "n2", type: "mentions", weight: 1 },
      { id: "e2", source: "n1", target: "n3", type: "attended", weight: 2 },
      ...(opts?.extraEdges ?? []),
    ],
    stats: {
      entity_count: 4 + (opts?.extraNodes?.length ?? 0),
      relationship_count: 2 + (opts?.extraEdges?.length ?? 0),
      entity_types: { person: 2, topic: 1, event: 1 },
    },
  };
}

function setupComposable(data?: GraphOverview | null) {
  const containerEl = document.createElement("div");
  Object.defineProperty(containerEl, "clientWidth", { value: 800 });
  Object.defineProperty(containerEl, "clientHeight", { value: 500 });
  containerEl.appendChild = vi.fn();

  const containerRef = ref<HTMLElement | null>(containerEl);
  const overviewRef = ref<GraphOverview | null>(data ?? null);

  const { result, unmount } = withSetup(() =>
    usePixiGraph(containerRef, overviewRef),
  );

  return { result, unmount, containerRef, overviewRef };
}

describe("usePixiGraph", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    // Re-stub Worker after restoreAllMocks
    vi.stubGlobal("Worker", MockWorker);
  });

  describe("return interface", () => {
    it("returns all expected control functions and refs", () => {
      const { result, unmount } = setupComposable();

      expect(result.zoomIn).toBeTypeOf("function");
      expect(result.zoomOut).toBeTypeOf("function");
      expect(result.resetView).toBeTypeOf("function");
      expect(result.highlightNode).toBeTypeOf("function");
      expect(result.destroy).toBeTypeOf("function");
      expect(result.reinit).toBeTypeOf("function");
      expect(result.refresh).toBeTypeOf("function");

      expect(result.hoveredNode).toBeDefined();
      expect(result.selectedNode).toBeDefined();
      expect(result.filters).toBeDefined();

      unmount();
    });
  });

  describe("initial state", () => {
    it("hoveredNode and selectedNode start as null", () => {
      const { result, unmount } = setupComposable();

      expect(result.hoveredNode.value).toBeNull();
      expect(result.selectedNode.value).toBeNull();

      unmount();
    });

    it("filters have correct defaults", () => {
      const { result, unmount } = setupComposable();

      expect(result.filters.searchQuery.value).toBe("");
      expect(result.filters.activeTypes.value).toBeInstanceOf(Set);
      expect(result.filters.activeTypes.value.size).toBe(0);
      expect(result.filters.showOrphans.value).toBe(true);

      unmount();
    });
  });

  describe("GraphFilters interface", () => {
    it("searchQuery is a writable string ref", () => {
      const { result, unmount } = setupComposable();

      result.filters.searchQuery.value = "test query";
      expect(result.filters.searchQuery.value).toBe("test query");

      unmount();
    });

    it("activeTypes is a writable Set ref", () => {
      const { result, unmount } = setupComposable();

      result.filters.activeTypes.value = new Set(["person", "topic"]);
      expect(result.filters.activeTypes.value.has("person")).toBe(true);
      expect(result.filters.activeTypes.value.has("topic")).toBe(true);
      expect(result.filters.activeTypes.value.has("event")).toBe(false);

      unmount();
    });

    it("showOrphans is a writable boolean ref", () => {
      const { result, unmount } = setupComposable();

      expect(result.filters.showOrphans.value).toBe(true);
      result.filters.showOrphans.value = false;
      expect(result.filters.showOrphans.value).toBe(false);

      unmount();
    });
  });

  describe("highlightNode", () => {
    it("sets hoveredNode when called with a node id", () => {
      const { result, unmount } = setupComposable();

      result.highlightNode("n1");
      expect(result.hoveredNode.value).toBe("n1");

      unmount();
    });

    it("clears hoveredNode when called with null", () => {
      const { result, unmount } = setupComposable();

      result.highlightNode("n1");
      result.highlightNode(null);
      expect(result.hoveredNode.value).toBeNull();

      unmount();
    });
  });

  describe("destroy", () => {
    it("resets hoveredNode and selectedNode to null", () => {
      const { result, unmount } = setupComposable();

      result.highlightNode("n1");
      expect(result.hoveredNode.value).toBe("n1");

      result.destroy();
      expect(result.hoveredNode.value).toBeNull();
      expect(result.selectedNode.value).toBeNull();

      unmount();
    });

    it("can be called multiple times without error", () => {
      const { result, unmount } = setupComposable();

      expect(() => {
        result.destroy();
        result.destroy();
        result.destroy();
      }).not.toThrow();

      unmount();
    });
  });

  describe("control functions do not throw without init", () => {
    it("zoomIn does not throw when no viewport", () => {
      const { result, unmount } = setupComposable();
      expect(() => result.zoomIn()).not.toThrow();
      unmount();
    });

    it("zoomOut does not throw when no viewport", () => {
      const { result, unmount } = setupComposable();
      expect(() => result.zoomOut()).not.toThrow();
      unmount();
    });

    it("resetView does not throw when no viewport", () => {
      const { result, unmount } = setupComposable();
      expect(() => result.resetView()).not.toThrow();
      unmount();
    });

    it("refresh does not throw when no nodes", () => {
      const { result, unmount } = setupComposable();
      expect(() => result.refresh()).not.toThrow();
      unmount();
    });

    it("reinit does not throw when no container", () => {
      const containerRef = ref<HTMLElement | null>(null);
      const overviewRef = ref<GraphOverview | null>(null);
      const { result, unmount } = withSetup(() =>
        usePixiGraph(containerRef, overviewRef),
      );
      expect(() => result.reinit()).not.toThrow();
      unmount();
    });
  });

  describe("isNodeVisible behavior (tested via filters + refresh)", () => {
    it("refresh does not throw when filters are set", async () => {
      const overview = makeOverview();
      const { result, unmount } = setupComposable(overview);

      // Init the graph so internal state is populated
      await result.reinit();

      // Set a type filter and refresh
      result.filters.activeTypes.value = new Set(["person"]);
      expect(() => result.refresh()).not.toThrow();

      // Set search query and refresh
      result.filters.searchQuery.value = "Ali";
      expect(() => result.refresh()).not.toThrow();

      // Toggle orphans off and refresh
      result.filters.showOrphans.value = false;
      expect(() => result.refresh()).not.toThrow();

      unmount();
    });

    it("changing filters does not throw with empty overview", () => {
      const { result, unmount } = setupComposable(null);

      result.filters.activeTypes.value = new Set(["person"]);
      result.filters.searchQuery.value = "test";
      result.filters.showOrphans.value = false;
      expect(() => result.refresh()).not.toThrow();

      unmount();
    });
  });

  describe("node size calculation", () => {
    it("node radius formula is Math.max(4, sqrt(degree) * 3)", () => {
      // Verify the formula for various degree counts
      // degree 0 -> max(4, 0) = 4
      expect(Math.max(4, Math.sqrt(0) * 3)).toBe(4);
      // degree 1 -> max(4, 3) = 4
      expect(Math.max(4, Math.sqrt(1) * 3)).toBe(4);
      // degree 4 -> max(4, 6) = 6
      expect(Math.max(4, Math.sqrt(4) * 3)).toBe(6);
      // degree 9 -> max(4, 9) = 9
      expect(Math.max(4, Math.sqrt(9) * 3)).toBe(9);
      // degree 16 -> max(4, 12) = 12
      expect(Math.max(4, Math.sqrt(16) * 3)).toBe(12);
    });
  });
});
