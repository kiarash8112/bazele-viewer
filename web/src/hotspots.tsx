import { useEffect, useState } from "react";
import { LoaderCircle, Plus, RefreshCw } from "lucide-react";
import { fetchHotspots, HOTSPOT_PAGE_SIZE, type HotspotKind, type Target } from "./api";
import { ExternalBadge, Label } from "./nodes";

interface Loaded {
  key: string; // Which list this is: kind, external filter and graph version.
  items: Target[];
  total: number;
}

const DESCRIPTIONS: Record<HotspotKind, string> = {
  targets:
    "Targets ranked by how many targets depend on them, directly or through other targets. " +
    "A change to one near the top affects that many targets.",
  files:
    "Source files ranked by how many targets depend on the rules that use them. " +
    "Editing one near the top affects that many targets.",
};

function errorText(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

function plural(n: number, word: string): string {
  return `${n.toLocaleString()} ${word}${n === 1 ? "" : "s"}`;
}

/** Ranks targets or source files by their transitive dependents. */
export function Hotspots(props: {
  of: HotspotKind;
  ready: boolean;
  version: string;
  onOpen(label: string): void;
  onSwitch(of: HotspotKind): void;
}) {
  const { of, ready, version } = props;
  const [external, setExternal] = useState(false);
  const [loaded, setLoaded] = useState<Loaded | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loadingMore, setLoadingMore] = useState(false);
  const [retryTick, setRetryTick] = useState(0);
  const withExternal = of === "targets" && external;
  const key = `${of}@${withExternal}@${version}`;

  useEffect(() => {
    if (!ready) return;
    const ctl = new AbortController();
    setError(null);
    fetchHotspots(of, 0, withExternal, ctl.signal)
      .then((page) => setLoaded({ key, items: page.nodes, total: page.total }))
      .catch((err) => {
        if (!ctl.signal.aborted) setError(errorText(err));
      });
    return () => ctl.abort();
    // `key` covers `of`, `withExternal` and the graph version.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, ready, retryTick]);

  const current = loaded && loaded.key === key ? loaded : null;

  const loadMore = async () => {
    if (!current || loadingMore) return;
    setLoadingMore(true);
    try {
      const page = await fetchHotspots(of, current.items.length, withExternal);
      setLoaded((prev) =>
        prev && prev.key === current.key ? { ...prev, items: [...prev.items, ...page.nodes], total: page.total } : prev,
      );
    } catch (err) {
      setError(`Couldn't load more: ${errorText(err)}`);
    } finally {
      setLoadingMore(false);
    }
  };

  // Bars are scaled to the top entry, which the first page always holds.
  const top = current?.items[0]?.dependents ?? 0;
  const remaining = current ? current.total - current.items.length : 0;

  return (
    <div className="hot">
      <div className="hot-inner">
        <header className="hot-head">
          <div>
            <h1>{of === "files" ? "Source file hotspots" : "Target hotspots"}</h1>
            <p>{DESCRIPTIONS[of]}</p>
          </div>
          <div className="hot-controls">
            <div className="seg" role="group" aria-label="What to rank">
              <button type="button" aria-pressed={of === "targets"} onClick={() => props.onSwitch("targets")}>
                Targets
              </button>
              <button type="button" aria-pressed={of === "files"} onClick={() => props.onSwitch("files")}>
                Source files
              </button>
            </div>
            {of === "targets" && (
              <label className="check">
                <input type="checkbox" checked={external} onChange={(e) => setExternal(e.target.checked)} />
                External repositories
              </label>
            )}
          </div>
        </header>

        {error && (
          <section className="panel panel-bad hot-panel" role="alert">
            <h2>Couldn't load hotspots</h2>
            <pre>{error}</pre>
            <button type="button" className="btn" onClick={() => setRetryTick((n) => n + 1)}>
              <RefreshCw size={14} /> Try again
            </button>
          </section>
        )}

        {!error && ready && !current && (
          <p className="hot-note" role="status">
            <LoaderCircle size={14} className="spin" aria-hidden="true" /> Ranking…
          </p>
        )}

        {current && current.total === 0 && (
          <p className="hot-note">
            {of === "files" ? "No source files in this workspace." : "No targets in this workspace."}
          </p>
        )}

        {current && current.total > 0 && (
          <>
            <p className="hot-note">
              {current.items.length < current.total
                ? `Top ${current.items.length.toLocaleString()} of ${current.total.toLocaleString()}`
                : `All ${current.total.toLocaleString()}`}
              {of === "targets" && " · click a target to open its deps in the graph"}
            </p>
            <ol className="hot-list">
              {current.items.map((t, i) => (
                <HotRow key={t.label} target={t} rank={i + 1} top={top} file={of === "files"} onOpen={props.onOpen} />
              ))}
            </ol>
            {remaining > 0 && (
              <button type="button" className="hot-more" onClick={loadMore} disabled={loadingMore}>
                {loadingMore ? <LoaderCircle size={15} className="spin" /> : <Plus size={15} strokeWidth={2.25} />}
                <span>
                  {loadingMore ? "Loading…" : `Show ${Math.min(remaining, HOTSPOT_PAGE_SIZE).toLocaleString()} more`}
                  <small>{remaining.toLocaleString()} not loaded yet</small>
                </span>
              </button>
            )}
          </>
        )}
      </div>
    </div>
  );
}

function HotRow(props: { target: Target; rank: number; top: number; file: boolean; onOpen(label: string): void }) {
  const { target, rank, top, file } = props;
  const dependents = target.dependents ?? 0;
  const share = top > 0 ? dependents / top : 0;
  const body = (
    <>
      <span className="hot-rank">{rank.toLocaleString()}</span>
      <span className="hot-target">
        <span className="t-meta">
          {target.kind && <span className="t-kind">{target.kind}</span>}
          {target.external && <ExternalBadge />}
        </span>
        <Label label={target.label} />
      </span>
      <span className="hot-score">
        <span className="hot-num">
          <strong>{dependents.toLocaleString()}</strong> {dependents === 1 ? "dependent" : "dependents"}
        </span>
        <span className="hot-bar" aria-hidden="true">
          <span style={{ width: `${share * 100}%` }} />
        </span>
        <span className="hot-direct">
          {file ? `used by ${plural(target.parentCount, "rule")}` : `${target.parentCount.toLocaleString()} direct`}
        </span>
      </span>
    </>
  );
  return (
    <li>
      {file ? (
        <div className="hot-row">{body}</div>
      ) : (
        <button
          type="button"
          className="hot-row hot-open"
          onClick={() => props.onOpen(target.label)}
          aria-label={`Open ${target.label} in the graph, ${plural(dependents, "dependent")}`}
        >
          {body}
        </button>
      )}
    </li>
  );
}
