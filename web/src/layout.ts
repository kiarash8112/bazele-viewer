// Places one layer on the canvas. Looking down, the focused target sits on top
// (none for the roots) with its direct deps in centred rows below it. Looking
// up, its direct dependents sit in rows above it instead, nearest row first. A
// layer is a star, so a wrapped grid keeps wide layers readable where a single
// layered row would not.

import type { Edge, Node } from "@xyflow/react";
import type { Direction, Target } from "./api";

export const NODE_W = 248;
export const NODE_H = 68;
export const FOCUS_W = 360;
const STUB = 40; // Room for the dotted stub under (or over) a node that leads on.
const GAP_X = 28;
const GAP_Y = 36;
const FOCUS_GAP = 96;
const TREE_X = -FOCUS_W / 2 + 56; // Item indent in the one-column tree.

export type TargetRole = "focus" | "child" | "parent" | "root";

export interface TargetNodeData extends Record<string, unknown> {
  target: Target;
  role: TargetRole;
  /** The layer's direction; the focus offers a stub to flip it. */
  dir: Direction;
  order: number;
}

export interface MoreNodeData extends Record<string, unknown> {
  remaining: number;
  loading: boolean;
  order: number;
}

export interface EmptyNodeData extends Record<string, unknown> {
  label: string;
  dir: Direction;
}

export type GraphNode =
  | Node<TargetNodeData, "target">
  | Node<MoreNodeData, "more">
  | Node<EmptyNodeData, "empty">;

export interface Layer {
  focus: Target | null;
  dir: Direction;
  items: Target[];
  total: number;
}

function columnsFor(count: number, maxCols: number): number {
  if (count <= Math.min(6, maxCols)) return Math.max(count, 1);
  return Math.min(maxCols, Math.ceil(Math.sqrt(count * 2.4)));
}

/**
 * The part of a layer worth framing first: the focus, its stub and its first
 * rows. `top` is the world y of the frame's top edge.
 */
export interface Frame {
  top: number;
  width: number;
  height: number;
  dir: Direction;
}

/** Slots the layer occupies: its items, plus a "show more" or empty slot. */
function slotCount(hasFocus: boolean, loaded: number, total: number): number {
  if (hasFocus && total === 0) return 1;
  return loaded + (total > loaded ? 1 : 0);
}

/** World y of an item row: below the focus looking down, above it looking up. */
function rowY(dir: Direction, hasFocus: boolean, row: number): number {
  const step = row * (NODE_H + STUB + GAP_Y);
  if (dir === "up") return -(FOCUS_GAP + NODE_H) - step;
  return (hasFocus ? NODE_H + FOCUS_GAP : 0) + step;
}

/**
 * The frame a layer will need, known before its items arrive (from the
 * target's dep or dependent count), so the camera can move while the clicked
 * node glides.
 */
export function frameFor(
  hasFocus: boolean,
  dir: Direction,
  loaded: number,
  total: number,
  maxCols: number,
): Frame {
  const count = slotCount(hasFocus, loaded, total);
  const cols = columnsFor(count, maxCols);
  const rows = Math.min(Math.ceil(count / cols), 3);
  const items = rows * (NODE_H + STUB) + (rows - 1) * GAP_Y;
  const width = Math.max(cols * NODE_W + (cols - 1) * GAP_X, hasFocus ? FOCUS_W : 0);
  if (!hasFocus) return { top: 0, width, height: items, dir };
  // The focus carries a stub that flips the direction: above it looking down,
  // below it looking up.
  const height = STUB + NODE_H + FOCUS_GAP + items;
  return { top: dir === "up" ? -(items + FOCUS_GAP) : -STUB, width, height, dir };
}

export function layoutLayer(
  layer: Layer,
  loadingMore: boolean,
  maxCols: number,
): { nodes: GraphNode[]; edges: Edge[]; frame: Frame } {
  const nodes: GraphNode[] = [];
  const edges: Edge[] = [];
  const { focus, dir } = layer;
  const hasFocus = focus !== null;
  const up = hasFocus && dir === "up";
  const remaining = layer.total - layer.items.length;
  const count = slotCount(hasFocus, layer.items.length, layer.total);
  const cols = columnsFor(count, maxCols);
  // One column (phones): items hang off a left gutter like a file tree, so each
  // edge visibly meets its own item instead of reading as a chain.
  const tree = hasFocus && cols === 1 && maxCols === 1;

  // Edges always run from the dependent to its dep: focus -> item looking
  // down, item -> focus looking up.
  const link = (item: string): Edge => {
    if (!focus) throw new Error("link without a focus");
    if (up) {
      return tree
        ? { id: `${item}->${focus.label}`, source: item, sourceHandle: "side-out", target: focus.label, targetHandle: "gutter-in", type: "smoothstep" }
        : { id: `${item}->${focus.label}`, source: item, sourceHandle: "bottom", target: focus.label, targetHandle: "top-in" };
    }
    return tree
      ? { id: `${focus.label}->${item}`, source: focus.label, sourceHandle: "gutter", target: item, targetHandle: "side", type: "smoothstep" }
      : { id: `${focus.label}->${item}`, source: focus.label, sourceHandle: "down", target: item, targetHandle: "top" };
  };

  if (focus) {
    nodes.push({
      id: focus.label,
      type: "target",
      position: { x: -FOCUS_W / 2, y: 0 },
      data: { target: focus, role: "focus", dir, order: 0 },
    });
    if (layer.total === 0) {
      nodes.push({
        id: "__empty__",
        type: "empty",
        position: { x: -NODE_W / 2, y: rowY(dir, true, 0) },
        data: { label: focus.label, dir },
      });
      edges.push(
        up
          ? { id: "__empty__", source: "__empty__", sourceHandle: "bottom", target: focus.label, targetHandle: "top-in", className: "is-ghost" }
          : { id: "__empty__", source: focus.label, sourceHandle: "down", target: "__empty__", targetHandle: "top", className: "is-ghost" },
      );
    }
  }

  for (let i = 0; i < count && !(hasFocus && layer.total === 0); i++) {
    const row = Math.floor(i / cols);
    const inRow = Math.min(cols, count - row * cols);
    const rowStart = tree ? TREE_X : -(inRow * NODE_W + (inRow - 1) * GAP_X) / 2;
    const x = rowStart + (i % cols) * (NODE_W + GAP_X);
    const y = rowY(dir, hasFocus, row);
    const t = layer.items[i];
    if (t) {
      nodes.push({
        id: t.label,
        type: "target",
        position: { x, y },
        data: { target: t, role: !focus ? "root" : up ? "parent" : "child", dir, order: i },
      });
      if (focus) edges.push(link(t.label));
    } else {
      nodes.push({
        id: "__more__",
        type: "more",
        position: { x, y },
        data: { remaining, loading: loadingMore, order: i },
      });
    }
  }
  return { nodes, edges, frame: frameFor(hasFocus, dir, layer.items.length, layer.total, maxCols) };
}
