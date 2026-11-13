// Places one layer on the canvas: the focused target on top (none for the
// roots) and its direct deps in centred rows below it. A layer is a star, so a
// wrapped grid keeps wide layers readable where a single layered row would not.

import type { Edge, Node } from "@xyflow/react";
import type { Target } from "./api";

export const NODE_W = 248;
export const NODE_H = 68;
export const FOCUS_W = 360;
const STUB = 40; // Room for the dotted stub under a node that has deps.
const GAP_X = 28;
const GAP_Y = 36;
const FOCUS_GAP = 96;
const TREE_X = -FOCUS_W / 2 + 56; // Child indent in the one-column tree.

export type TargetRole = "focus" | "child" | "root";

export interface TargetNodeData extends Record<string, unknown> {
  target: Target;
  role: TargetRole;
  order: number;
}

export interface MoreNodeData extends Record<string, unknown> {
  remaining: number;
  loading: boolean;
  order: number;
}

export interface EmptyNodeData extends Record<string, unknown> {
  label: string;
}

export type GraphNode =
  | Node<TargetNodeData, "target">
  | Node<MoreNodeData, "more">
  | Node<EmptyNodeData, "empty">;

export interface Layer {
  focus: Target | null;
  items: Target[];
  total: number;
}

function columnsFor(count: number, maxCols: number): number {
  if (count <= Math.min(6, maxCols)) return Math.max(count, 1);
  return Math.min(maxCols, Math.ceil(Math.sqrt(count * 2.4)));
}

/** The part of a layer worth framing first: the focus and its first rows. */
export interface Frame {
  width: number;
  height: number;
}

/** Slots the layer occupies: its deps, plus a "show more" or empty slot. */
function slotCount(hasFocus: boolean, loaded: number, total: number): number {
  if (hasFocus && total === 0) return 1;
  return loaded + (total > loaded ? 1 : 0);
}

/**
 * The frame a layer will need, known before its deps arrive (from the target's
 * dep count), so the camera can move while the clicked node glides.
 */
export function frameFor(hasFocus: boolean, loaded: number, total: number, maxCols: number): Frame {
  const count = slotCount(hasFocus, loaded, total);
  const cols = columnsFor(count, maxCols);
  const rows = Math.min(Math.ceil(count / cols), 3);
  const top = hasFocus ? NODE_H + FOCUS_GAP : 0;
  return {
    width: Math.max(cols * NODE_W + (cols - 1) * GAP_X, hasFocus ? FOCUS_W : 0),
    height: top + rows * (NODE_H + STUB) + (rows - 1) * GAP_Y,
  };
}

export function layoutLayer(
  layer: Layer,
  loadingMore: boolean,
  maxCols: number,
): { nodes: GraphNode[]; edges: Edge[]; frame: Frame } {
  const nodes: GraphNode[] = [];
  const edges: Edge[] = [];
  const hasFocus = layer.focus !== null;
  const remaining = layer.total - layer.items.length;
  const count = slotCount(hasFocus, layer.items.length, layer.total);
  const cols = columnsFor(count, maxCols);
  const top = hasFocus ? NODE_H + FOCUS_GAP : 0;
  // One column (phones): deps hang off a left gutter like a file tree, so each
  // edge visibly enters its own dep instead of reading as a chain.
  const tree = hasFocus && cols === 1 && maxCols === 1;

  if (layer.focus) {
    nodes.push({
      id: layer.focus.label,
      type: "target",
      position: { x: -FOCUS_W / 2, y: 0 },
      data: { target: layer.focus, role: "focus", order: 0 },
    });
    if (layer.total === 0) {
      nodes.push({
        id: "__empty__",
        type: "empty",
        position: { x: -NODE_W / 2, y: top },
        data: { label: layer.focus.label },
      });
      edges.push({
        id: "__empty__",
        source: layer.focus.label,
        sourceHandle: "down",
        target: "__empty__",
        targetHandle: "top",
        className: "is-ghost",
      });
    }
  }

  for (let i = 0; i < count && !(hasFocus && layer.total === 0); i++) {
    const row = Math.floor(i / cols);
    const inRow = Math.min(cols, count - row * cols);
    const rowStart = tree ? TREE_X : -(inRow * NODE_W + (inRow - 1) * GAP_X) / 2;
    const x = rowStart + (i % cols) * (NODE_W + GAP_X);
    const y = top + row * (NODE_H + STUB + GAP_Y);
    const t = layer.items[i];
    if (t) {
      nodes.push({
        id: t.label,
        type: "target",
        position: { x, y },
        data: { target: t, role: layer.focus ? "child" : "root", order: i },
      });
      if (layer.focus) {
        edges.push(
          tree
            ? {
                id: `${layer.focus.label}->${t.label}`,
                source: layer.focus.label,
                sourceHandle: "gutter",
                target: t.label,
                targetHandle: "side",
                type: "smoothstep",
              }
            : {
                id: `${layer.focus.label}->${t.label}`,
                source: layer.focus.label,
                sourceHandle: "down",
                target: t.label,
                targetHandle: "top",
              },
        );
      }
    } else {
      nodes.push({
        id: "__more__",
        type: "more",
        position: { x, y },
        data: { remaining, loading: loadingMore, order: i },
      });
    }
  }
  return { nodes, edges, frame: frameFor(hasFocus, layer.items.length, layer.total, maxCols) };
}
