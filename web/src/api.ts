// Client for the viewer's HTTP API (internal/api).

export interface Target {
  label: string;
  kind: string;
  external: boolean;
  depCount: number;
  parentCount: number;
}

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
