import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import {
  applyNodeChanges,
  Background,
  BackgroundVariant,
  Controls,
  MiniMap,
  ReactFlow,
  useReactFlow,
  type Edge,
  type NodeChange,
} from "@xyflow/react";
import { AlertTriangle, ChevronRight, LoaderCircle, RefreshCw, Workflow } from "lucide-react";
import { ApiError, fetchLayer, fetchStatus, PAGE_SIZE, refreshGraph, type Status, type Target } from "./api";
import { FOCUS_W, frameFor, layoutLayer, type Frame, type GraphNode, type Layer } from "./layout";
import { ActionsContext, nodeTypes, type CanvasActions } from "./nodes";

// ---- Path <-> URL hash, so the browser's back button walks up the graph. ----

function readPath(): string[] {
  const h = window.location.hash.replace(/^#/, "");
  if (!h) return [];
  return h.split(",").map((s) => decodeURIComponent(s)).filter(Boolean);
}

function writePath(path: string[]) {
  const h = path.map(encodeURIComponent).join(",");
  if (h === window.location.hash.replace(/^#/, "")) return;
  window.location.hash = h;
  if (!h) history.replaceState(null, "", window.location.pathname + window.location.search);
}

const delay = (ms: number) => new Promise((r) => setTimeout(r, ms));

function useMediaQuery(query: string): boolean {
  const [match, setMatch] = useState(() => window.matchMedia(query).matches);
  useEffect(() => {
    const mq = window.matchMedia(query);
    const on = () => setMatch(mq.matches);
    mq.addEventListener("change", on);
    return () => mq.removeEventListener("change", on);
  }, [query]);
  return match;
}

const CAPTION_H = 76; // Room above the layer for its caption.

interface Loaded {
  key: string | null; // null = roots
  version: string;
  items: Target[];
  total: number;
}

type LayerError = { key: string | null; message: string };

export function App() {
  const [path, setPath] = useState<string[]>(readPath);
  const [status, setStatus] = useState<Status | null>(null);
  const [serverDown, setServerDown] = useState(false);
  const [loaded, setLoaded] = useState<Loaded | null>(null);
  const [layerError, setLayerError] = useState<LayerError | null>(null);
  const [layerLoading, setLayerLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [leaving, setLeaving] = useState<string | null>(null);
  const [hovered, setHovered] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [retryTick, setRetryTick] = useState(0);
  const known = useRef(new Map<string, Target>());
  const { setViewport } = useReactFlow();
  const canvasRef = useRef<HTMLElement>(null);
  const narrow = useMediaQuery("(max-width: 720px)");
  const reduceMotion = useMediaQuery("(prefers-reduced-motion: reduce)");
  const maxCols = narrow ? 1 : 8;

  const key = path.length ? path[path.length - 1] : null;
  const version = status?.loadedAt ?? "";
  const ready = status?.state === "ready";

  // Follow the hash.
  useEffect(() => {
    const onHash = () => setPath(readPath());
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  // Poll status: it drives the first load, reloads and the status pill.
  useEffect(() => {
    let alive = true;
    let timer: ReturnType<typeof setTimeout>;
    const tick = async () => {
      try {
        const s = await fetchStatus();
        if (!alive) return;
        setStatus(s);
        setServerDown(false);
      } catch {
        if (alive) setServerDown(true);
      }
      if (alive) timer = setTimeout(tick, 2000);
    };
    tick();
    return () => {
      alive = false;
      clearTimeout(timer);
    };
  }, []);

  // Load the layer for the current path whenever it or the graph changes.
  useEffect(() => {
    if (!ready) return;
    const ctl = new AbortController();
    const minLeave = leaving ? delay(200) : Promise.resolve();
    setLayerLoading(true);
    setLayerError(null);
    (async () => {
      try {
        const [page] = await Promise.all([fetchLayer(key, 0, ctl.signal), minLeave]);
        for (const t of page.nodes) known.current.set(t.label, t);
        if (page.target) known.current.set(page.target.label, page.target);
        setLoaded({ key, version, items: page.nodes, total: page.total });
      } catch (err) {
        if (ctl.signal.aborted) return;
        if (err instanceof ApiError && err.status === 404 && key !== null) {
          setNotice(`${key} is no longer in the graph, so you're back at its parent.`);
          writePath(path.slice(0, -1));
          return;
        }
        setLayerError({ key, message: err instanceof Error ? err.message : String(err) });
      } finally {
        if (!ctl.signal.aborted) {
          setLayerLoading(false);
          setLeaving(null);
        }
      }
    })();
    return () => ctl.abort();
    // `leaving` and `path` are read once per navigation, not dependencies.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, version, ready, retryTick]);

  useEffect(() => {
    if (!notice) return;
    const t = setTimeout(() => setNotice(null), 6000);
    return () => clearTimeout(t);
  }, [notice]);

  const current = loaded && loaded.key === key ? loaded : null;
  const shown = current ?? loaded; // Keep the old layer on screen while the next one loads.

  const layer: Layer | null = useMemo(() => {
    if (!shown) return null;
    let focus: Target | null = null;
    if (shown.key !== null) {
      const k = known.current.get(shown.key);
      focus = {
        label: shown.key,
        kind: k?.kind ?? "",
        external: k?.external ?? shown.key.startsWith("@"),
        depCount: k?.depCount ?? shown.total,
        parentCount: k?.parentCount ?? 0,
      };
    }
    return { focus, items: shown.items, total: shown.total };
  }, [shown]);

  const { nodes, edges, frame } = useMemo(() => {
    if (!layer) return { nodes: [] as GraphNode[], edges: [] as Edge[], frame: null as Frame | null };
    const out = layoutLayer(layer, loadingMore, maxCols);
    if (leaving) {
      // The opened target becomes the focus and glides to its slot while the
      // rest fades out; the camera moves with it (see `open`).
      out.nodes = out.nodes.map((n): GraphNode =>
        n.id === leaving && n.type === "target"
          ? {
              ...n,
              data: { ...n.data, role: "focus" },
              position: { x: -FOCUS_W / 2, y: 0 },
              className: "is-opening",
              zIndex: 10,
            }
          : { ...n, className: "is-leaving" },
      );
      out.edges = out.edges.map((e) => ({ ...e, className: "is-leaving" }));
    } else if (hovered) {
      out.edges = out.edges.map((e) => (e.target === hovered ? { ...e, className: "is-hot", zIndex: 5 } : e));
    }
    return out;
  }, [layer, leaving, hovered, loadingMore, maxCols]);

  // React Flow stores each node's measured size on the node object. Keep it
  // across rebuilds (hover, glide, load more); otherwise every rebuild hides the
  // nodes until they are re-measured, and clicks fall through to the canvas.
  const [rfNodes, setRfNodes] = useState<GraphNode[]>([]);
  useEffect(() => {
    setRfNodes((prev) => {
      const byId = new Map(prev.map((n) => [n.id, n]));
      return nodes.map((n) => {
        const old = byId.get(n.id);
        return old?.measured ? ({ ...n, measured: old.measured } as GraphNode) : n;
      });
    });
  }, [nodes]);
  const onNodesChange = useCallback(
    (changes: NodeChange<GraphNode>[]) => setRfNodes((ns) => applyNodeChanges(changes, ns)),
    [],
  );

  // Frame a layer: the focus and its first rows, centred, never zoomed past
  // 100%. The layout is known up front, so no measuring is needed.
  const framedFor = useRef<string | null>(null);
  const frameLayer = useCallback(
    (f: Frame, animate: boolean) => {
      const el = canvasRef.current;
      if (!el) return;
      const { width: w, height: h } = el.getBoundingClientRect();
      const fit = Math.min(1, (w - 32) / f.width, (h - CAPTION_H - 40) / f.height);
      const zoom = Math.max(narrow ? 0.6 : 0.35, fit);
      setViewport({ x: w / 2, y: CAPTION_H, zoom }, { duration: animate && !reduceMotion ? 460 : 0 });
    },
    [narrow, reduceMotion, setViewport],
  );

  // Layers reached without a click (first load, back, breadcrumb, reload).
  useEffect(() => {
    if (!current || !frame || leaving) return;
    const id = `${current.key}@${current.version}@${maxCols}`;
    if (framedFor.current === id) return;
    const first = framedFor.current === null;
    framedFor.current = id;
    frameLayer(frame, !first);
  }, [current, frame, leaving, maxCols, frameLayer]);

  const open = useCallback(
    (label: string) => {
      if (leaving) return;
      setHovered(null);
      setLeaving(label);
      // Move the camera now, in step with the glide: the new layer's size is
      // known from the target's dep count before its deps arrive.
      const deps = known.current.get(label)?.depCount ?? 0;
      frameLayer(frameFor(true, Math.min(deps, PAGE_SIZE), deps, maxCols), true);
      framedFor.current = `${label}@${version}@${maxCols}`;
      writePath([...path, label]);
    },
    [path, leaving, frameLayer, maxCols, version],
  );

  const goTo = useCallback((depth: number) => writePath(path.slice(0, depth)), [path]);

  const loadMore = useCallback(async () => {
    if (!current || loadingMore) return;
    setLoadingMore(true);
    try {
      const page = await fetchLayer(current.key, current.items.length);
      for (const t of page.nodes) known.current.set(t.label, t);
      setLoaded((prev) =>
        prev && prev.key === current.key
          ? { ...prev, items: [...prev.items, ...page.nodes], total: page.total }
          : prev,
      );
    } catch (err) {
      setNotice(`Couldn't load more: ${err instanceof Error ? err.message : String(err)}`);
    } finally {
      setLoadingMore(false);
    }
  }, [current, loadingMore]);

  const actions: CanvasActions = useMemo(() => ({ open, loadMore, hover: setHovered }), [open, loadMore]);

  // Escape steps back up one layer.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape" && path.length > 0) writePath(path.slice(0, -1));
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [path]);

  const reload = useCallback(async () => {
    await refreshGraph();
    setStatus((s) => (s ? { ...s, loading: true } : s));
  }, []);

  return (
    <div className="app">
      <TopBar path={path} narrow={narrow} onCrumb={goTo} status={status} serverDown={serverDown} onReload={reload} />
      <main className="canvas" ref={canvasRef} aria-busy={layerLoading}>
        <ActionsContext.Provider value={actions}>
          <ReactFlow
            nodes={rfNodes}
            onNodesChange={onNodesChange}
            edges={edges}
            nodeTypes={nodeTypes}
            // Node-level handlers also keep pointer events on: React Flow turns
            // them off for nodes that are neither draggable nor selectable.
            onNodeMouseEnter={(_, n) => n.type === "target" && setHovered(n.id)}
            onNodeMouseLeave={() => setHovered(null)}
            nodesDraggable={false}
            nodesConnectable={false}
            nodesFocusable={false}
            edgesFocusable={false}
            elementsSelectable={false}
            minZoom={0.1}
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
              nodeColor={(n) => (n.id === key ? "var(--accent)" : "var(--mini-node)")}
              nodeBorderRadius={4}
              maskColor="var(--mini-mask)"
              ariaLabel="Overview of this layer"
            />
          </ReactFlow>
        </ActionsContext.Provider>

        {shown && ready && <LayerCaption loaded={shown} loading={layerLoading} />}

        <Overlay
          status={status}
          serverDown={serverDown}
          layer={current}
          layerError={layerError && layerError.key === key ? layerError : null}
          onRetryGraph={reload}
          onRetryLayer={() => setRetryTick((n) => n + 1)}
        />

        {notice && (
          <div className="toast" role="status">
            {notice}
          </div>
        )}
      </main>
    </div>
  );
}

// ---- Top bar ----

function TopBar(props: {
  path: string[];
  narrow: boolean;
  onCrumb(depth: number): void;
  status: Status | null;
  serverDown: boolean;
  onReload(): void;
}) {
  const { path, onCrumb } = props;
  const [expanded, setExpanded] = useState(false);
  const crumbsRef = useRef<HTMLElement>(null);
  useEffect(() => setExpanded(false), [path.length]);
  // Keep the current step in view when the trail overflows.
  useEffect(() => {
    const el = crumbsRef.current;
    if (el) el.scrollLeft = el.scrollWidth;
  }, [path, expanded]);

  // Long trails keep the first step and the last two (on a phone, only the
  // current one); the middle folds away.
  const visible = path.map((label, i) => ({ label, depth: i + 1 }));
  let items: ({ label: string; depth: number } | null)[] = visible;
  if (!expanded && props.narrow && path.length > 1) items = [null, ...visible.slice(-1)];
  else if (!expanded && path.length > 4) items = [visible[0], null, ...visible.slice(-2)];

  return (
    <header className="topbar">
      <div className="brand">
        <Workflow size={18} strokeWidth={2} aria-hidden="true" />
        <span>Bazel Graph</span>
      </div>
      <nav className="crumbs" aria-label="Breadcrumb" ref={crumbsRef}>
        <ol>
          <li>
            <button
              type="button"
              className="crumb"
              aria-current={path.length === 0 ? "page" : undefined}
              onClick={() => onCrumb(0)}
            >
              Roots
            </button>
          </li>
          {items.map((it, i) => (
            <li key={it ? it.label + it.depth : "fold" + i}>
              <ChevronRight size={14} className="crumb-sep" aria-hidden="true" />
              {it ? (
                <button
                  type="button"
                  className="crumb mono"
                  title={it.label}
                  aria-current={it.depth === path.length ? "page" : undefined}
                  onClick={() => onCrumb(it.depth)}
                >
                  {it.label}
                </button>
              ) : (
                <button
                  type="button"
                  className="crumb"
                  title={`Show ${path.length - (items.length - 1)} hidden steps`}
                  onClick={() => setExpanded(true)}
                >
                  …
                </button>
              )}
            </li>
          ))}
        </ol>
      </nav>
      <StatusPill status={props.status} serverDown={props.serverDown} onReload={props.onReload} />
    </header>
  );
}

function formatTime(iso?: string): string {
  if (!iso) return "";
  return new Date(iso).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function StatusPill({ status, serverDown, onReload }: { status: Status | null; serverDown: boolean; onReload(): void }) {
  let tone = "ok";
  let text: string;
  if (serverDown) {
    tone = "bad";
    text = "Server unreachable";
  } else if (!status) {
    tone = "busy";
    text = "Connecting…";
  } else if (status.state === "loading") {
    tone = "busy";
    text = "Loading graph…";
  } else if (status.state === "error") {
    tone = "bad";
    text = "Graph failed to load";
  } else {
    text = `${status.targets.toLocaleString()} targets · loaded ${formatTime(status.loadedAt)}`;
    if (status.loading) {
      tone = "busy";
      text = `Reloading… · showing graph from ${formatTime(status.loadedAt)}`;
    } else if (status.error) {
      tone = "warn";
      text = `Reload failed · showing graph from ${formatTime(status.loadedAt)}`;
    }
  }
  const busy = !serverDown && (status?.loading ?? false);
  return (
    <div className="status">
      {status?.state === "ready" && status.warning && (
        <span className="chip warn" title={status.warning}>
          <AlertTriangle size={13} aria-hidden="true" />
          Incomplete graph
        </span>
      )}
      <span className={`status-text tone-${tone}`} title={status?.error || undefined} role="status">
        <span className="dot" aria-hidden="true" />
        {text}
      </span>
      <button
        type="button"
        className="icon-btn"
        onClick={onReload}
        disabled={busy || serverDown}
        title="Reload the graph after editing BUILD files"
        aria-label="Reload graph"
      >
        <RefreshCw size={15} className={busy ? "spin" : undefined} />
      </button>
    </div>
  );
}

// ---- Canvas overlays ----

function LayerCaption({ loaded, loading }: { loaded: Loaded; loading: boolean }) {
  const name = loaded.key === null ? null : loaded.key.slice(loaded.key.lastIndexOf(":") + 1) || loaded.key;
  const partial = loaded.items.length < loaded.total;
  return (
    <div className="caption">
      <h1>
        {name === null ? (
          "Workspace roots"
        ) : (
          <>
            Deps of <span className="mono">:{name}</span>
          </>
        )}
      </h1>
      {(name === null || partial || loading) && (
        <p>
          {name === null && `${loaded.total.toLocaleString()} entry point${loaded.total === 1 ? "" : "s"}`}
          {name === null && partial && " · "}
          {partial && `showing ${loaded.items.length.toLocaleString()} of ${loaded.total.toLocaleString()}`}
          {loading && <LoaderCircle size={13} className="spin" aria-label="Loading" />}
        </p>
      )}
    </div>
  );
}

function Overlay(props: {
  status: Status | null;
  serverDown: boolean;
  layer: Loaded | null;
  layerError: LayerError | null;
  onRetryGraph(): void;
  onRetryLayer(): void;
}) {
  const { status, serverDown, layer, layerError } = props;

  if (serverDown && status?.state !== "ready") {
    return (
      <Panel title="Can't reach the viewer server">
        <p>Is <code>bazel run //cmd/viewer</code> still running? This page reconnects on its own.</p>
      </Panel>
    );
  }
  if (!status || status.state === "loading") {
    return (
      <Panel title="Loading the build graph" spinner>
        <p>
          Running <code>bazel query 'deps(//...)'</code> in your workspace. Large workspaces can take a minute.
        </p>
      </Panel>
    );
  }
  if (status.state === "error") {
    return (
      <Panel title="Couldn't load the build graph" tone="bad">
        {status.error && <pre>{status.error}</pre>}
        <p>Fix the reported problem, then try again.</p>
        <button type="button" className="btn" onClick={props.onRetryGraph}>
          <RefreshCw size={14} /> Try again
        </button>
      </Panel>
    );
  }
  if (layerError) {
    return (
      <Panel title={layerError.key ? `Couldn't load the deps of ${layerError.key}` : "Couldn't load the roots"} tone="bad">
        <pre>{layerError.message}</pre>
        <button type="button" className="btn" onClick={props.onRetryLayer}>
          <RefreshCw size={14} /> Try again
        </button>
      </Panel>
    );
  }
  if (layer && layer.key === null && layer.total === 0) {
    return (
      <Panel title="No targets in this workspace">
        <p>
          <code>bazel query</code> found no rule targets under <code>//...</code>.
        </p>
      </Panel>
    );
  }
  return null;
}

function Panel(props: { title: string; tone?: "bad"; spinner?: boolean; children: ReactNode }) {
  return (
    <div className="overlay">
      <section className={`panel ${props.tone === "bad" ? "panel-bad" : ""}`} role={props.tone === "bad" ? "alert" : "status"}>
        <h2>
          {props.spinner && <LoaderCircle size={16} className="spin" aria-hidden="true" />}
          {props.title}
        </h2>
        {props.children}
      </section>
    </div>
  );
}
