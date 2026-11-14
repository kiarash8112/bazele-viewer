import { useEffect, useRef, useState } from "react";
import { LoaderCircle, Plus, RefreshCw, Search } from "lucide-react";
import { fetchHotspots, HOTSPOT_PAGE_SIZE, type HotspotKind, type Target } from "./api";
import { ExternalBadge, Label } from "./nodes";

interface Loaded {
  key: string; // Which list this is: kind, external filter, graph version and search.
  search: string;
  items: Target[];
  total: number;
}

// The window the server counts changes over (changesSince in cmd/viewer).
const CHANGES_WINDOW = "6 months";

const DESCRIPTIONS: Record<HotspotKind, string> = {
  targets:
    "Targets ranked by how many targets depend on them, directly or through other targets. " +
    "A change to one near the top affects that many targets. " +
    `Changes counts the commits in the last ${CHANGES_WINDOW} that touched its source files.`,
  files:
    "Source files ranked by how many targets depend on the rules that use them. " +
    "Editing one near the top affects that many targets. " +
    `Changes counts the commits in the last ${CHANGES_WINDOW} that touched the file.`,
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
  const [input, setInput] = useState(""); // What is typed in the search box.
  const [search, setSearch] = useState(""); // The search the list shows, trailing `input`.
  const [loaded, setLoaded] = useState<Loaded | null>(null);
  const [tops, setTops] = useState<Record<string, number>>({});
  const [error, setError] = useState<string | null>(null);
  const [loadingMore, setLoadingMore] = useState(false);
  const [retryTick, setRetryTick] = useState(0);
  const searchRef = useRef<HTMLInputElement>(null);
  const withExternal = of === "targets" && external;
  const listKey = `${of}@${withExternal}@${version}`; // The ranking, before any search.
  const key = `${listKey}@${search}`;

  // Search as the user types, without a request per keystroke.
  useEffect(() => {
    const query = input.trim();
    const t = setTimeout(() => setSearch(query), 180);
    return () => clearTimeout(t);
  }, [input]);

  // "/" jumps to the search box, as on many developer sites.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const el = e.target as HTMLElement;
      if (e.key !== "/" || e.ctrlKey || e.metaKey || e.altKey) return;
      if (el.isContentEditable || ["INPUT", "TEXTAREA", "SELECT"].includes(el.tagName)) return;
      e.preventDefault();
      searchRef.current?.focus();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  useEffect(() => {
    if (!ready) return;
    const ctl = new AbortController();
    setError(null);
    fetchHotspots(of, 0, withExternal, search, ctl.signal)
      .then((page) => {
        setLoaded({ key, search, items: page.nodes, total: page.total });
        // Remember the top of the unfiltered ranking, so bars keep one scale while searching.
        if (!search) setTops((prev) => ({ ...prev, [listKey]: page.nodes[0]?.dependents ?? 0 }));
      })
      .catch((err) => {
        if (!ctl.signal.aborted) setError(errorText(err));
      });
    return () => ctl.abort();
    // `key` covers `of`, `withExternal`, `search` and the graph version.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, ready, retryTick]);

  const current = loaded && loaded.key === key ? loaded : null;
  // While the next list loads, keep showing the previous one, dimmed.
  const shown = current ?? loaded;
  const stale = shown !== null && shown !== current;

  const loadMore = async () => {
    if (!current || loadingMore) return;
    setLoadingMore(true);
    try {
      const page = await fetchHotspots(of, current.items.length, withExternal, current.search);
      setLoaded((prev) =>
        prev && prev.key === current.key ? { ...prev, items: [...prev.items, ...page.nodes], total: page.total } : prev,
      );
    } catch (err) {
      setError(`Couldn't load more: ${errorText(err)}`);
    } finally {
      setLoadingMore(false);
    }
  };

  // Bars are scaled to the top of the whole ranking: rank 1, which an
  // unfiltered first page holds.
  const top = tops[listKey] ?? shown?.items[0]?.dependents ?? 0;
  const remaining = current ? current.total - current.items.length : 0;
  const noun = of === "files" ? "source files" : "targets";

  return (
    <div className="hot">
      <div className="hot-inner">
        <header className="hot-head">
          <div>
            <h1>{of === "files" ? "Source file hotspots" : "Target hotspots"}</h1>
            <p>{DESCRIPTIONS[of]}</p>
          </div>
        </header>

        <div className="hot-toolbar">
          <label className="search">
            <Search size={14} aria-hidden="true" />
            <input
              ref={searchRef}
              type="search"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Escape") {
                  if (input) setInput("");
                  else e.currentTarget.blur();
                }
              }}
              placeholder={`Search ${noun} by label, e.g. utils/metric or :spark`}
              aria-label={`Search ${noun} by label`}
              spellCheck={false}
              autoComplete="off"
            />
            {stale && <LoaderCircle size={14} className="spin" aria-label="Searching" />}
            {!input && (
              <kbd className="search-key" aria-hidden="true">
                /
              </kbd>
            )}
          </label>
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
        </div>

        {error && (
          <section className="panel panel-bad hot-panel" role="alert">
            <h2>Couldn't load hotspots</h2>
            <pre>{error}</pre>
            <button type="button" className="btn" onClick={() => setRetryTick((n) => n + 1)}>
              <RefreshCw size={14} /> Try again
            </button>
          </section>
        )}

        {!error && ready && !shown && (
          <p className="hot-note" role="status">
            <LoaderCircle size={14} className="spin" aria-hidden="true" /> Ranking…
          </p>
        )}

        {shown && shown.total === 0 && (
          <p className="hot-note" role="status">
            {shown.search ? (
              <span>
                No {noun} match <code>{shown.search}</code>.
              </span>
            ) : (
              `No ${noun} in this workspace.`
            )}
          </p>
        )}

        {shown && shown.total > 0 && (
          <>
            <p className="hot-note" role="status">
              <span>
                {shown.search ? (
                  <>
                    {shown.total.toLocaleString()} {shown.total === 1 ? "match" : "matches"} for{" "}
                    <code>{shown.search}</code>
                    {shown.items.length < shown.total && `, showing ${shown.items.length.toLocaleString()}`}
                  </>
                ) : shown.items.length < shown.total ? (
                  `Top ${shown.items.length.toLocaleString()} of ${shown.total.toLocaleString()}`
                ) : (
                  `All ${shown.total.toLocaleString()}`
                )}
                {of === "targets" && " · click a target to open its deps in the graph"}
              </span>
            </p>
            <ol className={`hot-list${stale ? " is-stale" : ""}`} aria-busy={stale}>
              {shown.items.map((t, i) => (
                <HotRow
                  key={t.label}
                  target={t}
                  rank={t.rank ?? i + 1}
                  top={top}
                  file={of === "files"}
                  onOpen={props.onOpen}
                />
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
      <span
        className="hot-changes"
        title={
          target.changes === undefined
            ? "Change counts are unavailable: the workspace's git history couldn't be read"
            : `${plural(target.changes, "commit")} in the last ${CHANGES_WINDOW}`
        }
      >
        {target.changes === undefined ? (
          <strong>—</strong>
        ) : (
          <span>
            <strong>{target.changes.toLocaleString()}</strong> {target.changes === 1 ? "change" : "changes"}
          </span>
        )}
        <small>last {CHANGES_WINDOW}</small>
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
