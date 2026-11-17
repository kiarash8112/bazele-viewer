// Expand mode: the canvas keeps what it shows. Clicking a target adds its deps
// in the level below it, or its dependents in the level above it, and clicking
// again folds them away. Each target appears once, however many expanded
// targets reach it, so shared deps join up instead of repeating.

import { createContext, memo, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import {
  applyNodeChanges,
  Background,
  BackgroundVariant,
  Controls,
  Handle,
  MiniMap,
  Position,
  ReactFlow,
  useReactFlow,
  type Edge,
  type Node,
  type NodeChange,
  type NodeProps,
} from "@xyflow/react";
import { ArrowUp, ChevronDown, ChevronUp, LoaderCircle, Plus } from "lucide-react";
import { fetchLayer, PAGE_SIZE, type Direction, type GraphKind, type Target } from "./api";
import { NODE_H, NODE_W } from "./layout";
import { ExternalBadge, Label, noun, shortName } from "./nodes";

const STUB = 40; // Room for the deps stub under a target.
const GAP_X = 28;
const GAP_Y = 28;
const LEVEL_GAP = 72;
const ROW_H = NODE_H + STUB;
const CAPTION_H = 76;

/** One target's loaded deps (down) or dependents (up). */
interface Expansion {
  items: Target[];
  total: number;
  loading: boolean;
}

const expKey = (label: string, dir: Direction) => `${dir}:${label}`;

function parseExpKey(key: string): { label: string; dir: Direction } {
  const i = key.indexOf(":");
  return { dir: key.slice(0, i) as Direction, label: key.slice(i + 1) };
}

interface XTargetData extends Record<string, unknown> {
  target: Target;
  seed: boolean;
  down?: Expansion;
  up?: Expansion;
}

interface XMoreData extends Record<string, unknown> {
  owner: string; // expKey of the expansion it continues
  remaining: number;
  loading: boolean;
}

type XNode = Node<XTargetData, "x"> | Node<XMoreData, "xmore">;

interface ExploreActions {
  toggle(label: string, dir: Direction): void;
  loadMore(owner: string): void;
}

const Actions = createContext<ExploreActions>({ toggle: () => {}, loadMore: () => {} });

function plural(n: number, word: string): string {
  return `${n.toLocaleString()} ${word}${n === 1 ? "" : "s"}`;
}

function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

/**
 * Lays out what is expanded. The seeds sit on level 0, deps one level below
 * the target that opened them and dependents one level above. A target keeps
 * the level it was first reached at. Each level is a centred grid, its targets
 * ordered by where their neighbours toward level 0 sit, which keeps edges short.
 */
function buildGraph(
  seeds: Target[],
  exps: Map<string, Expansion>,
  known: Map<string, Target>,
  cols: number,
): { nodes: XNode[]; edges: Edge[] } {
  const level = new Map<string, number>();
  const queue: string[] = [];
  const edges: Edge[] = [];
  const edgeIds = new Set<string>();
  const mores = new Map<string, XMoreData>();

  const reach = (id: string, l: number) => {
    if (level.has(id)) return;
    level.set(id, l);
    queue.push(id);
  };
  const link = (from: string, to: string, ghost = false) => {
    const id = `${from}->${to}`;
    if (edgeIds.has(id)) return;
    edgeIds.add(id);
    edges.push({ id, source: from, sourceHandle: "out", target: to, targetHandle: "in", className: ghost ? "is-ghost" : undefined });
  };

  for (const s of seeds) reach(s.label, 0);
  for (let i = 0; i < queue.length; i++) {
    const n = queue[i];
    if (mores.has(n)) continue;
    const l = level.get(n)!;
    for (const dir of ["down", "up"] as const) {
      const e = exps.get(expKey(n, dir));
      if (!e) continue;
      const next = dir === "down" ? l + 1 : l - 1;
      for (const t of e.items) {
        if (dir === "down") link(n, t.label);
        else link(t.label, n);
        reach(t.label, next);
      }
      if (e.total > e.items.length) {
        const id = `__more__${expKey(n, dir)}`;
        mores.set(id, { owner: expKey(n, dir), remaining: e.total - e.items.length, loading: e.loading });
        if (dir === "down") link(n, id, true);
        else link(id, n, true);
        reach(id, next);
      }
    }
  }

  // Group by level, in the order reached.
  const byLevel = new Map<number, string[]>();
  for (const id of queue) {
    const l = level.get(id)!;
    if (!byLevel.has(l)) byLevel.set(l, []);
    byLevel.get(l)!.push(id);
  }
  const neighbours = new Map<string, string[]>();
  for (const e of edges) {
    for (const [a, b] of [[e.source, e.target], [e.target, e.source]]) {
      if (!neighbours.has(a)) neighbours.set(a, []);
      neighbours.get(a)!.push(b);
    }
  }

  const x = new Map<string, number>(); // centre x
  const rowsOf = new Map<number, number>();
  const place = (l: number) => {
    let ids = byLevel.get(l)!;
    if (l !== 0) {
      const inner = l > 0 ? l - 1 : l + 1;
      const bary = (id: string) => {
        const xs = (neighbours.get(id) ?? []).filter((n) => level.get(n) === inner && x.has(n)).map((n) => x.get(n)!);
        return xs.length ? xs.reduce((a, b) => a + b, 0) / xs.length : Infinity;
      };
      const keyed = ids.map((id, i) => ({ id, i, b: bary(id) }));
      keyed.sort((p, q) => (p.b === q.b ? p.i - q.i : p.b - q.b));
      ids = keyed.map((k) => k.id);
      byLevel.set(l, ids);
    }
    const c = Math.max(1, Math.min(cols, ids.length));
    rowsOf.set(l, Math.ceil(ids.length / c));
    ids.forEach((id, i) => {
      const row = Math.floor(i / c);
      const inRow = Math.min(c, ids.length - row * c);
      const start = -(inRow * NODE_W + (inRow - 1) * GAP_X) / 2;
      x.set(id, start + (i % c) * (NODE_W + GAP_X) + NODE_W / 2);
    });
  };
  const levels = [...byLevel.keys()];
  const maxL = Math.max(0, ...levels);
  const minL = Math.min(0, ...levels);
  for (let l = 0; l <= maxL; l++) if (byLevel.has(l)) place(l);
  for (let l = -1; l >= minL; l--) if (byLevel.has(l)) place(l);

  const height = (l: number) => {
    const r = rowsOf.get(l) ?? 0;
    return r ? r * ROW_H + (r - 1) * GAP_Y : 0;
  };
  const top = new Map<number, number>([[0, 0]]);
  for (let l = 1; l <= maxL; l++) top.set(l, top.get(l - 1)! + height(l - 1) + LEVEL_GAP);
  for (let l = -1; l >= minL; l--) top.set(l, top.get(l + 1)! - LEVEL_GAP - height(l));

  const seedSet = new Set(seeds.map((s) => s.label));
  const nodes: XNode[] = [];
  for (const [l, ids] of byLevel) {
    const c = Math.max(1, Math.min(cols, ids.length));
    ids.forEach((id, i) => {
      const position = { x: x.get(id)! - NODE_W / 2, y: top.get(l)! + Math.floor(i / c) * (ROW_H + GAP_Y) };
      const more = mores.get(id);
      if (more) {
        nodes.push({ id, type: "xmore", position, data: more });
        return;
      }
      const target = known.get(id) ?? { label: id, kind: "", external: id.startsWith("@"), depCount: 0, parentCount: 0 };
      nodes.push({
        id,
        type: "x",
        position,
        data: { target, seed: seedSet.has(id), down: exps.get(expKey(id, "down")), up: exps.get(expKey(id, "up")) },
      });
    });
  }
  return { nodes, edges };
}

export function ExploreCanvas(props: {
  /** The graph to walk. Changing it needs a new canvas: give each graph its own React key. */
  graph: GraphKind;
  /** Where to start: a target looking one way, or the roots (null). */
  seed: { label: string; dir: Direction } | null;
  version: string;
  narrow: boolean;
  reduceMotion: boolean;
  onError(message: string): void;
}) {
  const { graph, seed, version, narrow, reduceMotion, onError } = props;
  const [seeds, setSeeds] = useState<Target[] | null>(null);
  const [exps, setExps] = useState<Map<string, Expansion>>(() => new Map());
  const [hovered, setHovered] = useState<string | null>(null);
  const known = useRef(new Map<string, Target>());
  const expsRef = useRef(exps);
  expsRef.current = exps;
  const generation = useRef(0); // Drops answers that arrive after a reseed.
  const canvasRef = useRef<HTMLElement>(null);
  const { setViewport, getViewport } = useReactFlow();
  const cols = narrow ? 2 : 10;
  const seedKey = seed ? expKey(seed.label, seed.dir) : "";

  // Start over from the seed whenever it or the graph changes.
  useEffect(() => {
    const gen = ++generation.current;
    const ctl = new AbortController();
    setSeeds(null);
    setExps(new Map());
    known.current = new Map();
    framed.current = false;
    (async () => {
      try {
        const page = await fetchLayer(graph, seed?.label ?? null, seed?.dir ?? "down", 0, ctl.signal);
        if (gen !== generation.current) return;
        for (const t of page.nodes) known.current.set(t.label, t);
        if (seed && page.target) {
          known.current.set(page.target.label, page.target);
          setExps(new Map([[seedKey, { items: page.nodes, total: page.total, loading: false }]]));
          setSeeds([page.target]);
        } else {
          setSeeds(page.nodes);
        }
      } catch (err) {
        if (!ctl.signal.aborted) onError(message(err));
      }
    })();
    return () => ctl.abort();
    // `seed` is covered by `seedKey`.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [seedKey, version]);

  const { nodes, edges } = useMemo(
    () => (seeds ? buildGraph(seeds, exps, known.current, cols) : { nodes: [] as XNode[], edges: [] as Edge[] }),
    [seeds, exps, cols],
  );

  const shownEdges = useMemo(
    () =>
      hovered
        ? edges.map((e) => (e.source === hovered || e.target === hovered ? { ...e, className: "is-hot", zIndex: 5 } : e))
        : edges,
    [edges, hovered],
  );

  // Keep measured sizes across rebuilds (see App).
  const [rfNodes, setRfNodes] = useState<XNode[]>([]);
  const rfNodesRef = useRef(rfNodes);
  rfNodesRef.current = rfNodes;
  useEffect(() => {
    setRfNodes((prev) => {
      const byId = new Map(prev.map((n) => [n.id, n]));
      return nodes.map((n) => {
        const old = byId.get(n.id);
        return old?.measured ? ({ ...n, measured: old.measured } as XNode) : n;
      });
    });
  }, [nodes]);
  const onNodesChange = useCallback(
    (changes: NodeChange<XNode>[]) => setRfNodes((ns) => applyNodeChanges(changes, ns)),
    [],
  );

  // Frame the seeds once they arrive: level 0 under the caption, or mid-screen
  // when the seed opened upward.
  const framed = useRef(false);
  useEffect(() => {
    const el = canvasRef.current;
    if (framed.current || !seeds || !el || nodes.length === 0) return;
    framed.current = true;
    const { width: w, height: h } = el.getBoundingClientRect();
    const rowW = Math.min(cols, Math.max(seeds.length, 1)) * (NODE_W + GAP_X);
    const zoom = Math.max(narrow ? 0.6 : 0.35, Math.min(1, (w - 32) / rowW));
    const y = seed?.dir === "up" ? h * 0.6 : CAPTION_H + 24;
    setViewport({ x: w / 2, y, zoom }, { duration: 0 });
  }, [seeds, nodes, cols, narrow, seed, setViewport]);

  // Keep the clicked target where it is on screen while the layout around it
  // changes: move the camera by however far the target moved.
  const anchor = useRef<{ id: string; x: number; y: number } | null>(null);
  const pinAnchor = (id: string) => {
    const n = rfNodesRef.current.find((m) => m.id === id);
    anchor.current = n ? { id, x: n.position.x, y: n.position.y } : null;
  };
  useEffect(() => {
    const a = anchor.current;
    if (!a) return;
    anchor.current = null;
    const n = nodes.find((m) => m.id === a.id);
    if (!n) return;
    const dx = n.position.x - a.x;
    const dy = n.position.y - a.y;
    if (!dx && !dy) return;
    const vp = getViewport();
    setViewport(
      { x: vp.x - dx * vp.zoom, y: vp.y - dy * vp.zoom, zoom: vp.zoom },
      { duration: reduceMotion ? 0 : 460 },
    );
  }, [nodes, getViewport, setViewport, reduceMotion]);

  const toggle = useCallback(
    async (label: string, dir: Direction) => {
      const key = expKey(label, dir);
      const gen = generation.current;
      pinAnchor(label);
      if (expsRef.current.has(key)) {
        setExps((prev) => {
          const m = new Map(prev);
          m.delete(key);
          return m;
        });
        return;
      }
      setExps((prev) => new Map(prev).set(key, { items: [], total: 0, loading: true }));
      try {
        const page = await fetchLayer(graph, label, dir, 0);
        if (gen !== generation.current) return;
        for (const t of page.nodes) known.current.set(t.label, t);
        if (page.target) known.current.set(page.target.label, page.target);
        pinAnchor(label);
        setExps((prev) =>
          prev.has(key) ? new Map(prev).set(key, { items: page.nodes, total: page.total, loading: false }) : prev,
        );
      } catch (err) {
        if (gen !== generation.current) return;
        setExps((prev) => {
          const m = new Map(prev);
          m.delete(key);
          return m;
        });
        onError(message(err));
      }
    },
    [graph, onError],
  );

  const loadMore = useCallback(
    async (owner: string) => {
      const cur = expsRef.current.get(owner);
      if (!cur || cur.loading) return;
      const gen = generation.current;
      const { label, dir } = parseExpKey(owner);
      setExps((prev) => new Map(prev).set(owner, { ...cur, loading: true }));
      try {
        const page = await fetchLayer(graph, label, dir, cur.items.length);
        if (gen !== generation.current) return;
        for (const t of page.nodes) known.current.set(t.label, t);
        setExps((prev) => {
          const e = prev.get(owner);
          if (!e) return prev;
          return new Map(prev).set(owner, { items: [...e.items, ...page.nodes], total: page.total, loading: false });
        });
      } catch (err) {
        if (gen !== generation.current) return;
        setExps((prev) => {
          const e = prev.get(owner);
          return e ? new Map(prev).set(owner, { ...e, loading: false }) : prev;
        });
        onError(`Couldn't load more: ${message(err)}`);
      }
    },
    [graph, onError],
  );

  const actions = useMemo(() => ({ toggle, loadMore }), [toggle, loadMore]);
  const shown = nodes.filter((n) => n.type === "x").length;
  const seedName = seed ? shortName(seed.label) : null;

  return (
    <main className="canvas" ref={canvasRef} aria-busy={seeds === null}>
      <Actions.Provider value={actions}>
        <ReactFlow
          nodes={rfNodes}
          onNodesChange={onNodesChange}
          edges={shownEdges}
          nodeTypes={xNodeTypes}
          onNodeMouseEnter={(_, n) => n.type === "x" && setHovered(n.id)}
          onNodeMouseLeave={() => setHovered(null)}
          nodesDraggable={false}
          nodesConnectable={false}
          nodesFocusable={false}
          edgesFocusable={false}
          elementsSelectable={false}
          minZoom={0.05}
          maxZoom={2}
          proOptions={{ hideAttribution: true }}
          colorMode="dark"
        >
          <Background variant={BackgroundVariant.Dots} gap={22} size={1.2} color="var(--grid-dot)" />
          <Controls showInteractive={false} position="bottom-left" />
          <MiniMap
            position="bottom-right"
            pannable
            zoomable
            nodeColor={(n) => ((n.data as XTargetData).seed ? "var(--accent)" : "var(--mini-node)")}
            nodeBorderRadius={4}
            maskColor="var(--mini-mask)"
            ariaLabel="Overview of the expanded graph"
          />
        </ReactFlow>
      </Actions.Provider>
      {seeds && (
        <div className="caption">
          <h1>
            {seedName === null ? (
              graph === "modules" ? "Root module" : "Workspace roots"
            ) : (
              <>
                {seed?.dir === "up" ? "Used by" : "Deps of"} <span className="mono">{seedName}</span>
              </>
            )}
          </h1>
          <p>{plural(shown, graph === "modules" ? "module" : "target")} shown · click to expand, again to fold</p>
        </div>
      )}
    </main>
  );
}

const XTarget = memo(function XTarget({ data }: NodeProps<XNode & { type: "x" }>) {
  const { target, seed, down, up } = data as XTargetData;
  const a = useContext(Actions);
  const hasDeps = target.depCount > 0;
  const openDown = () => a.toggle(target.label, "down");
  return (
    <div className="t-wrap" style={{ width: NODE_W }}>
      <Handle id="in" type="target" position={Position.Top} isConnectable={false} />
      <div className="t-enter">
        <div
          className={`t-node ${hasDeps ? "t-drill" : "t-leaf"}${seed ? " x-seed" : ""}`}
          onClick={hasDeps ? openDown : undefined}
        >
          <div className="t-meta">
            {target.kind && <span className="t-kind">{target.kind}</span>}
            {target.external && <ExternalBadge />}
            {target.parentCount > 0 && (
              <button
                type="button"
                className="x-up nodrag"
                aria-pressed={!!up}
                onClick={(e) => {
                  e.stopPropagation();
                  a.toggle(target.label, "up");
                }}
                title={
                  up
                    ? "Fold away what depends on it"
                    : `Show the ${plural(target.parentCount, noun(target.label))} that depend on it directly`
                }
              >
                {up?.loading ? <LoaderCircle size={11} className="spin" /> : <ArrowUp size={11} strokeWidth={2.25} />}
                used by {target.parentCount.toLocaleString()}
              </button>
            )}
          </div>
          <Label label={target.label} />
        </div>
        {hasDeps && (
          <button
            type="button"
            className={`t-stub t-stub-btn x-down nodrag${down ? " is-open" : ""}`}
            onClick={openDown}
            aria-expanded={!!down}
            aria-label={`${down ? "Fold" : "Show"} the ${plural(target.depCount, "direct dep")} of ${target.label}`}
          >
            <span className="t-stub-line" />
            <span className="t-stub-count">
              {down?.loading && <LoaderCircle size={11} className="spin" />}
              {plural(target.depCount, "dep")}
              {down ? <ChevronUp size={12} strokeWidth={2.25} /> : <ChevronDown size={12} strokeWidth={2.25} />}
            </span>
          </button>
        )}
      </div>
      <Handle id="out" type="source" position={Position.Bottom} isConnectable={false} />
    </div>
  );
});

const XMore = memo(function XMore({ data }: NodeProps<XNode & { type: "xmore" }>) {
  const { owner, remaining, loading } = data as XMoreData;
  const a = useContext(Actions);
  return (
    <div className="t-wrap" style={{ width: NODE_W }}>
      <Handle id="in" type="target" position={Position.Top} isConnectable={false} />
      <div className="t-enter">
        <button type="button" className="t-node t-more nodrag" onClick={() => a.loadMore(owner)} disabled={loading}>
          {loading ? <LoaderCircle size={16} className="spin" /> : <Plus size={16} strokeWidth={2.25} />}
          <span>
            {loading ? "Loading…" : <>Show {Math.min(remaining, PAGE_SIZE).toLocaleString()} more</>}
            <small>{remaining.toLocaleString()} not loaded yet</small>
          </span>
        </button>
      </div>
      <Handle id="out" type="source" position={Position.Bottom} isConnectable={false} />
    </div>
  );
});

const xNodeTypes = { x: XTarget, xmore: XMore };
