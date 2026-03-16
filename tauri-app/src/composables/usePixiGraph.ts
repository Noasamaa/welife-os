import { ref, onUnmounted, type Ref } from "vue";
import { Application, Container, Graphics, Text } from "pixi.js";
import type { GraphOverview, GraphEdge } from "../types/import";

// --- Entity type color mapping (same palette as before) ---
const TYPE_COLORS: Record<string, number> = {
  person: 0x2d6a4f,
  event: 0xe67e22,
  topic: 0x3498db,
  promise: 0x9b59b6,
  place: 0xe74c3c,
};
const DEFAULT_NODE_COLOR = 0x888888;
const EDGE_COLOR = 0xffffff;
const BG_COLOR = 0x1a1a2e;
const MIN_ZOOM = 0.1;
const MAX_ZOOM = 5;

// Alpha values for Obsidian-style hover highlighting
const ALPHA_HOVER_NEIGHBOR = 0.8;
const ALPHA_HOVER_DIM = 0.15;
const EDGE_ALPHA_NORMAL = 0.25;
const EDGE_ALPHA_BRIGHT = 0.9;
const EDGE_ALPHA_DIM = 0.05;

interface NodeGfx {
  container: Container;
  circle: Graphics;
  label: Text;
  id: string;
  type: string;
  name: string;
  radius: number;
}

export interface GraphFilters {
  searchQuery: Ref<string>;
  activeTypes: Ref<Set<string>>;
  showOrphans: Ref<boolean>;
}

export interface PixiGraphControls {
  zoomIn: () => void;
  zoomOut: () => void;
  resetView: () => void;
  highlightNode: (nodeId: string | null) => void;
  hoveredNode: Ref<string | null>;
  selectedNode: Ref<string | null>;
  filters: GraphFilters;
  refresh: () => void;
  destroy: () => void;
  reinit: () => void;
}

export function usePixiGraph(
  containerRef: Ref<HTMLElement | null>,
  overview: Ref<GraphOverview | null>,
  onNodeClick?: (nodeId: string, nodeType: string, nodeName: string) => void,
): PixiGraphControls {
  let app: Application | null = null;
  let viewport: Container | null = null;
  let edgesGfx: Graphics | null = null;
  let worker: Worker | null = null;
  let nodeGfxMap = new Map<string, NodeGfx>();
  let adjacency = new Map<string, Set<string>>();
  let edgeList: GraphEdge[] = [];
  let dragTarget: NodeGfx | null = null;
  let isPanning = false;
  let panStart = { x: 0, y: 0 };
  let destroyed = false;
  let stabilized = false;
  let tickCount = 0;
  let currentCanvas: HTMLCanvasElement | null = null;
  let initGeneration = 0;

  const hoveredNode = ref<string | null>(null);
  const selectedNode = ref<string | null>(null);

  // Filter state for search/filter panel
  const searchQuery = ref("");
  const activeTypes = ref(new Set<string>());
  const showOrphans = ref(true);
  const filters: GraphFilters = { searchQuery, activeTypes, showOrphans };

  // Degree cache for orphan detection
  let degreeCache = new Map<string, number>();

  // Cached lowercase search query for hot-path perf (avoids per-edge allocation)
  let cachedSearchLower = "";

  function isNodeVisible(entry: NodeGfx): boolean {
    if (activeTypes.value.size > 0 && !activeTypes.value.has(entry.type)) {
      return false;
    }
    if (!showOrphans.value && (degreeCache.get(entry.id) ?? 0) === 0) {
      return false;
    }
    if (cachedSearchLower.length > 0) {
      if (!entry.name.toLowerCase().includes(cachedSearchLower)) {
        return false;
      }
    }
    return true;
  }

  // --- Adjacency lookup ---
  function buildAdjacency(edges: GraphEdge[]): Map<string, Set<string>> {
    const adj = new Map<string, Set<string>>();
    for (const e of edges) {
      if (!adj.has(e.source)) adj.set(e.source, new Set());
      if (!adj.has(e.target)) adj.set(e.target, new Set());
      adj.get(e.source)!.add(e.target);
      adj.get(e.target)!.add(e.source);
    }
    return adj;
  }

  // --- Edge rendering (batched: 1-2 stroke calls, skips hidden nodes) ---
  function drawEdges(): void {
    if (!edgesGfx) return;
    edgesGfx.clear();

    const h = hoveredNode.value;
    cachedSearchLower = searchQuery.value.toLowerCase();
    const hasFilters = activeTypes.value.size > 0 || !showOrphans.value || cachedSearchLower.length > 0;

    if (h === null) {
      for (const edge of edgeList) {
        const src = nodeGfxMap.get(edge.source);
        const tgt = nodeGfxMap.get(edge.target);
        if (!src || !tgt) continue;
        if (hasFilters && (!isNodeVisible(src) || !isNodeVisible(tgt))) continue;
        edgesGfx.moveTo(src.container.x, src.container.y);
        edgesGfx.lineTo(tgt.container.x, tgt.container.y);
      }
      edgesGfx.stroke({ color: EDGE_COLOR, alpha: EDGE_ALPHA_NORMAL, width: 0.8 });
    } else {
      // Dim edges first
      for (const edge of edgeList) {
        if (edge.source === h || edge.target === h) continue;
        const src = nodeGfxMap.get(edge.source);
        const tgt = nodeGfxMap.get(edge.target);
        if (!src || !tgt) continue;
        if (hasFilters && (!isNodeVisible(src) || !isNodeVisible(tgt))) continue;
        edgesGfx.moveTo(src.container.x, src.container.y);
        edgesGfx.lineTo(tgt.container.x, tgt.container.y);
      }
      edgesGfx.stroke({ color: EDGE_COLOR, alpha: EDGE_ALPHA_DIM, width: 0.5 });
      // Bright connected edges on top
      for (const edge of edgeList) {
        if (edge.source !== h && edge.target !== h) continue;
        const src = nodeGfxMap.get(edge.source);
        const tgt = nodeGfxMap.get(edge.target);
        if (!src || !tgt) continue;
        if (hasFilters && (!isNodeVisible(src) || !isNodeVisible(tgt))) continue;
        edgesGfx.moveTo(src.container.x, src.container.y);
        edgesGfx.lineTo(tgt.container.x, tgt.container.y);
      }
      edgesGfx.stroke({ color: EDGE_COLOR, alpha: EDGE_ALPHA_BRIGHT, width: 1.2 });
    }
  }

  // --- Obsidian-style hover: highlight + neighbors, dim the rest ---
  function updateHighlight(): void {
    const h = hoveredNode.value;
    const neighbors = h ? adjacency.get(h) ?? new Set<string>() : null;
    cachedSearchLower = searchQuery.value.toLowerCase();

    for (const [id, node] of nodeGfxMap) {
      const visible = isNodeVisible(node);
      node.container.visible = visible;
      if (!visible) continue;

      if (h === null) {
        node.container.alpha = 1;
        node.circle.scale.set(1);
      } else if (id === h) {
        node.container.alpha = 1;
        node.circle.scale.set(1.2);
      } else if (neighbors?.has(id)) {
        node.container.alpha = ALPHA_HOVER_NEIGHBOR;
        node.circle.scale.set(1);
      } else {
        node.container.alpha = ALPHA_HOVER_DIM;
        node.circle.scale.set(1);
      }
    }
    drawEdges();
  }

  function applyFilters(): void {
    updateHighlight();
  }

  // --- Label visibility: hide at low zoom, show progressively ---
  function updateLabelVisibility(): void {
    if (!viewport) return;
    const scale = viewport.scale.x;
    for (const node of nodeGfxMap.values()) {
      if (scale < 0.5) {
        node.label.visible = false;
      } else if (scale < 1.0) {
        node.label.visible = node.radius > 8;
      } else {
        node.label.visible = true;
      }
    }
  }

  // --- Fit graph into viewport ---
  function fitToView(): void {
    if (!app || !viewport || nodeGfxMap.size === 0) return;

    let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
    for (const node of nodeGfxMap.values()) {
      minX = Math.min(minX, node.container.x);
      minY = Math.min(minY, node.container.y);
      maxX = Math.max(maxX, node.container.x);
      maxY = Math.max(maxY, node.container.y);
    }

    const graphW = maxX - minX;
    const graphH = maxY - minY;
    if (graphW < 1 && graphH < 1) return;

    const padding = 80;
    const canvasW = app.screen.width;
    const canvasH = app.screen.height;
    const scale = Math.min(canvasW / (graphW + padding * 2), canvasH / (graphH + padding * 2), 2);

    viewport.scale.set(scale);
    viewport.x = canvasW / 2 - ((minX + maxX) / 2) * scale;
    viewport.y = canvasH / 2 - ((minY + maxY) / 2) * scale;
    updateLabelVisibility();
  }

  // --- Zoom via mousewheel (zoom around cursor position) ---
  function handleWheel(e: WheelEvent): void {
    e.preventDefault();
    if (!viewport) return;

    const factor = e.deltaY > 0 ? 0.9 : 1.1;
    const newScale = viewport.scale.x * factor;
    if (newScale < MIN_ZOOM || newScale > MAX_ZOOM) return;

    const mx = e.offsetX;
    const my = e.offsetY;
    const worldX = (mx - viewport.x) / viewport.scale.x;
    const worldY = (my - viewport.y) / viewport.scale.y;

    viewport.scale.set(newScale);
    viewport.x = mx - worldX * newScale;
    viewport.y = my - worldY * newScale;
    updateLabelVisibility();
  }

  // --- Programmatic zoom ---
  function applyZoom(factor: number): void {
    if (!viewport || !app) return;
    const newScale = Math.max(MIN_ZOOM, Math.min(MAX_ZOOM, viewport.scale.x * factor));
    const cx = app.screen.width / 2;
    const cy = app.screen.height / 2;
    const worldX = (cx - viewport.x) / viewport.scale.x;
    const worldY = (cy - viewport.y) / viewport.scale.y;

    viewport.scale.set(newScale);
    viewport.x = cx - worldX * newScale;
    viewport.y = cy - worldY * newScale;
    updateLabelVisibility();
  }

  function zoomIn(): void { applyZoom(1.3); }
  function zoomOut(): void { applyZoom(1 / 1.3); }
  function resetView(): void { fitToView(); }

  function highlightNode(nodeId: string | null): void {
    hoveredNode.value = nodeId;
    updateHighlight();
  }

  // Wait for browser to complete layout (nextTick only waits for Vue update, not reflow)
  function waitForLayout(): Promise<void> {
    return new Promise<void>((resolve) => requestAnimationFrame(() => requestAnimationFrame(() => resolve())));
  }

  // Fallback circular layout when Web Worker fails
  function applyCircularLayout(): void {
    const nodes = Array.from(nodeGfxMap.values());
    const count = nodes.length;
    if (count === 0) return;
    const radius = Math.min(400, count * 3);
    for (let i = 0; i < count; i++) {
      const angle = (2 * Math.PI * i) / count;
      nodes[i].container.x = Math.cos(angle) * radius;
      nodes[i].container.y = Math.sin(angle) * radius;
    }
    drawEdges();
    fitToView();
  }

  // --- Initialize pixi.js Application + Web Worker ---
  async function init(): Promise<void> {
    destroy();
    destroyed = false;
    const generation = ++initGeneration;

    const container = containerRef.value;
    const data = overview.value;
    if (!container || !data || data.nodes.length === 0) return;

    // Wait for browser reflow so container has real dimensions
    await waitForLayout();
    if (destroyed || generation !== initGeneration) return;

    // Verify container has real dimensions
    if (container.clientWidth === 0 || container.clientHeight === 0) {
      console.warn("[usePixiGraph] container has 0 dimensions, retrying...");
      await waitForLayout();
      if (destroyed || generation !== initGeneration) return;
    }

    // 1. Create pixi Application (WebGL with Canvas 2D fallback)
    app = new Application();
    await app.init({
      resizeTo: container,
      backgroundColor: BG_COLOR,
      antialias: true,
      resolution: window.devicePixelRatio || 1,
      autoDensity: true,
    });

    // Guard: a newer init() or destroy() was called while we awaited
    if (destroyed || generation !== initGeneration) {
      app.destroy(true);
      app = null;
      return;
    }

    currentCanvas = app.canvas as HTMLCanvasElement;
    currentCanvas.style.borderRadius = "8px";
    container.appendChild(currentCanvas);

    // 2. Viewport container (zoom/pan transforms applied here)
    viewport = new Container();
    app.stage.addChild(viewport);

    // 3. Edges layer (rendered below nodes)
    edgesGfx = new Graphics();
    viewport.addChild(edgesGfx);

    // 4. Compute degrees & prepare edge data
    const degreeMap = new Map<string, number>();
    edgeList = data.edges.filter((e) => e.source !== e.target);
    for (const edge of edgeList) {
      degreeMap.set(edge.source, (degreeMap.get(edge.source) ?? 0) + 1);
      degreeMap.set(edge.target, (degreeMap.get(edge.target) ?? 0) + 1);
    }
    adjacency = buildAdjacency(edgeList);
    degreeCache = degreeMap;

    // 5. Create node graphics (pixi.Graphics circle + pixi.Text label)
    for (const node of data.nodes) {
      const degree = degreeMap.get(node.id) ?? 0;
      const radius = Math.max(4, Math.sqrt(degree) * 3);
      const color = TYPE_COLORS[node.type] ?? DEFAULT_NODE_COLOR;

      const nodeContainer = new Container();

      const circle = new Graphics();
      circle.circle(0, 0, radius);
      circle.fill(color);
      circle.stroke({ color: 0xffffff, width: 1.5, alpha: 0.3 });
      nodeContainer.addChild(circle);

      const label = new Text({
        text: node.name,
        style: { fontSize: 11, fill: 0xe0e0e0, fontFamily: "Inter, system-ui, sans-serif" },
      });
      label.anchor.set(0.5, 0);
      label.y = radius + 4;
      label.resolution = 2;
      nodeContainer.addChild(label);

      // Pointer interaction
      circle.eventMode = "static";
      circle.cursor = "pointer";

      const nid = node.id;
      const ntype = node.type;
      const nname = node.name;

      circle.on("pointerover", () => {
        hoveredNode.value = nid;
        updateHighlight();
      });

      circle.on("pointerout", () => {
        if (dragTarget) return;
        if (hoveredNode.value === nid) {
          hoveredNode.value = null;
          updateHighlight();
        }
      });

      circle.on("pointerdown", (e) => {
        e.stopPropagation();
        selectedNode.value = nid;
        onNodeClick?.(nid, ntype, nname);
        dragTarget = nodeGfxMap.get(nid) ?? null;
      });

      viewport.addChild(nodeContainer);
      nodeGfxMap.set(node.id, {
        container: nodeContainer, circle, label,
        id: node.id, type: node.type, name: node.name, radius,
      });
    }

    // 6. Stage interactions (pan + drag continuation)
    app.stage.eventMode = "static";
    app.stage.hitArea = app.screen;

    app.stage.on("pointerdown", (e) => {
      if (!dragTarget) {
        isPanning = true;
        panStart = { x: e.global.x - (viewport?.x ?? 0), y: e.global.y - (viewport?.y ?? 0) };
        if (currentCanvas) currentCanvas.style.cursor = "grabbing";
      }
    });

    app.stage.on("pointermove", (e) => {
      if (dragTarget && viewport) {
        const worldPos = viewport.toLocal(e.global);
        dragTarget.container.x = worldPos.x;
        dragTarget.container.y = worldPos.y;
        drawEdges();
        worker?.postMessage({ type: "drag", id: dragTarget.id, x: worldPos.x, y: worldPos.y });
      } else if (isPanning && viewport) {
        viewport.x = e.global.x - panStart.x;
        viewport.y = e.global.y - panStart.y;
      }
    });

    const handleUp = () => {
      if (dragTarget) {
        worker?.postMessage({ type: "dragend", id: dragTarget.id });
        dragTarget = null;
      }
      isPanning = false;
      if (currentCanvas) currentCanvas.style.cursor = "grab";
    };

    app.stage.on("pointerup", handleUp);
    app.stage.on("pointerupoutside", handleUp);

    // 7. Wheel zoom
    currentCanvas.addEventListener("wheel", handleWheel, { passive: false });
    currentCanvas.style.cursor = "grab";

    // 8. Start Web Worker for d3-force simulation
    let workerFailed = false;
    let workerFallbackTimer: ReturnType<typeof setTimeout> | null = null;

    try {
      worker = new Worker(
        new URL("../workers/forceWorker.ts", import.meta.url),
        { type: "module" },
      );
    } catch (err) {
      console.warn("[usePixiGraph] Worker creation failed, using circular layout:", err);
      workerFailed = true;
    }

    if (workerFailed || !worker) {
      applyCircularLayout();
      return;
    }

    worker.onerror = (err) => {
      console.warn("[usePixiGraph] Worker error, falling back to circular layout:", err);
      if (workerFallbackTimer !== null) clearTimeout(workerFallbackTimer);
      applyCircularLayout();
    };

    worker.onmessage = (e) => {
      if (workerFallbackTimer !== null) {
        clearTimeout(workerFallbackTimer);
        workerFallbackTimer = null;
      }
      if (e.data.type === "tick") {
        for (const pos of e.data.positions as Array<{ id: string; x: number; y: number }>) {
          const node = nodeGfxMap.get(pos.id);
          if (node) {
            node.container.x = pos.x;
            node.container.y = pos.y;
          }
        }
        drawEdges();
        tickCount++;
        if (!stabilized && tickCount % 10 === 0) fitToView();
      } else if (e.data.type === "stabilized") {
        stabilized = true;
        fitToView();
      }
    };

    // Send initial data to worker (use real canvas dimensions with safe minimum)
    stabilized = false;
    tickCount = 0;
    const canvasW = Math.max(app.screen.width, 800);
    const canvasH = Math.max(app.screen.height, 500);
    worker.postMessage({
      type: "init",
      nodes: data.nodes.map((n) => ({
        id: n.id,
        radius: Math.max(4, Math.sqrt(degreeMap.get(n.id) ?? 0) * 3),
      })),
      links: edgeList.map((e) => ({ source: e.source, target: e.target })),
      width: canvasW,
      height: canvasH,
    });

    // Safety net: if no tick arrives within 3 seconds, fall back to circular layout
    workerFallbackTimer = setTimeout(() => {
      workerFallbackTimer = null;
      if (tickCount === 0 && !destroyed) {
        console.warn("[usePixiGraph] No worker ticks received after 3s, using circular layout");
        applyCircularLayout();
      }
    }, 3000);
  }

  // --- Cleanup ---
  function destroy(): void {
    destroyed = true;

    if (worker) {
      worker.postMessage({ type: "stop" });
      worker.terminate();
      worker = null;
    }

    if (currentCanvas) {
      currentCanvas.removeEventListener("wheel", handleWheel);
      currentCanvas = null;
    }

    if (app) {
      app.destroy(true);
      app = null;
    }

    viewport = null;
    edgesGfx = null;
    nodeGfxMap = new Map();
    adjacency = new Map();
    edgeList = [];
    dragTarget = null;
    isPanning = false;
    hoveredNode.value = null;
    selectedNode.value = null;
    stabilized = false;
    tickCount = 0;
  }

  function reinit(): void {
    void init();
  }

  onUnmounted(() => {
    destroy();
  });

  return { zoomIn, zoomOut, resetView, highlightNode, hoveredNode, selectedNode, filters, refresh: applyFilters, destroy, reinit };
}
