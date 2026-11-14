// Client for the viewer's HTTP API (internal/api).

export interface Target {
  label: string;
  kind: string;
  external: boolean;
  depCount: number;
  parentCount: number;
  /** Targets that depend on this one, directly or transitively (hotspots only). */
  dependents?: number;
  /** Commits in the last 6 months that changed the file or the target's source files (hotspots only; absent without git history). */
  changes?: number;
  /** 1-based position in the full hotspot ranking, kept under a search (hotspots only). */
  rank?: number;
}

export type HotspotKind = "targets" | "files";
export type HotspotSort = "dependents" | "changes";

export interface Page {
  /** The queried target itself (children requests only). */
  target?: Target;
  nodes: Target[];
  total: number;
  offset: number;
  limit: number;
}

export type GraphState = "loading" | "ready" | "error";

export interface Status {
  state: GraphState;
  loading: boolean;
  targets: number;
  loadedAt?: string;
  warning?: string;
  error?: string;
}

export const PAGE_SIZE = 200;

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
  ) {
    super(message);
  }
}

async function getJSON<T>(url: string, signal?: AbortSignal): Promise<T> {
  const res = await fetch(url, { signal });
  if (!res.ok) {
    let msg = `${res.status} ${res.statusText}`;
    try {
      const body = (await res.json()) as { error?: string };
      if (body.error) msg = body.error;
    } catch {
      // Keep the status line.
    }
    throw new ApiError(res.status, msg);
  }
  return (await res.json()) as T;
}

export function fetchStatus(signal?: AbortSignal): Promise<Status> {
  return getJSON<Status>("/api/status", signal);
}

export async function refreshGraph(): Promise<void> {
  await fetch("/api/refresh", { method: "POST" });
}

/** Fetches the roots (label === null) or one target's direct deps. */
export function fetchLayer(label: string | null, offset: number, signal?: AbortSignal): Promise<Page> {
  const q = new URLSearchParams({ offset: String(offset), limit: String(PAGE_SIZE) });
  if (label === null) return getJSON<Page>(`/api/roots?${q}`, signal);
  q.set("label", label);
  return getJSON<Page>(`/api/children?${q}`, signal);
}

export const HOTSPOT_PAGE_SIZE = 100;

/**
 * Fetches targets or source files ranked by their transitive dependents, or by
 * their recent changes, keeping only labels that contain `search` (ignoring
 * case) when it is set.
 */
export function fetchHotspots(
  of: HotspotKind,
  sort: HotspotSort,
  offset: number,
  external: boolean,
  search: string,
  signal?: AbortSignal,
): Promise<Page> {
  const q = new URLSearchParams({ of, sort, offset: String(offset), limit: String(HOTSPOT_PAGE_SIZE) });
  if (external) q.set("external", "true");
  if (search) q.set("q", search);
  return getJSON<Page>(`/api/hotspots?${q}`, signal);
}
