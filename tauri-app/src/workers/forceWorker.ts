/**
 * Web Worker running d3-force simulation off the main thread.
 * Receives node/link data, computes force-directed layout positions,
 * and posts position updates back to the main thread at ~60fps.
 *
 * Uses Barnes-Hut approximation (theta=0.9) via d3-force's forceManyBody
 * for O(n log n) performance, matching Obsidian's approach.
 */
import {
  forceSimulation,
  forceLink,
  forceManyBody,
  forceCenter,
  forceCollide,
} from "d3-force";
import type { SimulationNodeDatum, SimulationLinkDatum } from "d3-force";

interface ForceNode extends SimulationNodeDatum {
  id: string;
  radius: number;
}

type ForceLink = SimulationLinkDatum<ForceNode>;

let simulation: ReturnType<typeof forceSimulation<ForceNode>> | null = null;
let nodes: ForceNode[] = [];
let tickHandle: ReturnType<typeof setTimeout> | null = null;

// Type-safe wrapper for Worker.postMessage (file runs in Web Worker context)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const post = (globalThis as any).postMessage.bind(globalThis) as (
  msg: Record<string, unknown>,
) => void;

function sendPositions(): void {
  post({
    type: "tick",
    positions: nodes.map((n) => ({ id: n.id, x: n.x!, y: n.y! })),
  });
}

function startTicking(): void {
  if (tickHandle !== null) return;

  function tick(): void {
    if (!simulation) {
      tickHandle = null;
      return;
    }
    simulation.tick();
    sendPositions();

    if (simulation.alpha() > simulation.alphaMin()) {
      tickHandle = setTimeout(tick, 16);
    } else {
      tickHandle = null;
      post({ type: "stabilized" });
    }
  }

  tick();
}

function stopTicking(): void {
  if (tickHandle !== null) {
    clearTimeout(tickHandle);
    tickHandle = null;
  }
}

self.addEventListener("message", (e: MessageEvent) => {
  const msg = e.data;

  switch (msg.type) {
    case "init": {
      stopTicking();
      if (simulation) {
        simulation.stop();
        simulation = null;
      }

      const w = msg.width as number;
      const h = msg.height as number;

      nodes = (msg.nodes as Array<{ id: string; radius: number }>).map(
        (n) => ({
          id: n.id,
          radius: n.radius,
          x: (Math.random() - 0.5) * w,
          y: (Math.random() - 0.5) * h,
          vx: 0,
          vy: 0,
        }),
      );

      const nodeIds = new Set(nodes.map((n) => n.id));
      const links: ForceLink[] = (
        msg.links as Array<{ source: string; target: string }>
      )
        .filter(
          (l) =>
            nodeIds.has(l.source) &&
            nodeIds.has(l.target) &&
            l.source !== l.target,
        )
        .map((l) => ({ source: l.source, target: l.target }));

      simulation = forceSimulation<ForceNode>(nodes)
        .force(
          "charge",
          forceManyBody<ForceNode>().strength(-150).theta(0.9),
        )
        .force(
          "link",
          forceLink<ForceNode, ForceLink>(links)
            .id((d) => (d as ForceNode).id)
            .distance(80)
            .strength(0.5),
        )
        .force("center", forceCenter(0, 0))
        .force(
          "collide",
          forceCollide<ForceNode>().radius((d) => d.radius + 2),
        )
        .alphaDecay(0.02)
        .velocityDecay(0.4)
        .stop();

      startTicking();
      break;
    }

    case "drag": {
      const dragNode = nodes.find((n) => n.id === msg.id);
      if (dragNode && simulation) {
        dragNode.fx = msg.x;
        dragNode.fy = msg.y;
        simulation.alpha(Math.max(simulation.alpha(), 0.3));
        startTicking();
      }
      break;
    }

    case "dragend": {
      const endNode = nodes.find((n) => n.id === msg.id);
      if (endNode) {
        endNode.fx = null;
        endNode.fy = null;
      }
      // Re-heat so neighbors settle into new equilibrium
      if (simulation) {
        simulation.alpha(Math.max(simulation.alpha(), 0.1));
        startTicking();
      }
      break;
    }

    case "stop": {
      stopTicking();
      if (simulation) {
        simulation.stop();
        simulation = null;
      }
      nodes = [];
      break;
    }
  }
});
