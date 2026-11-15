import { createContext, Fragment, memo, useContext, type CSSProperties } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { ChevronDown, ChevronUp, Globe, LoaderCircle, Plus } from "lucide-react";
import type { Direction, Target } from "./api";
import type { EmptyNodeData, GraphNode, MoreNodeData, TargetNodeData } from "./layout";
import { FOCUS_W, NODE_W } from "./layout";

export interface CanvasActions {
  open(label: string, dir: Direction): void;
  loadMore(): void;
  hover(label: string | null): void;
}

export const ActionsContext = createContext<CanvasActions>({
  open: () => {},
  loadMore: () => {},
  hover: () => {},
});

/** Splits "//pkg/path:name" so the target name can carry the emphasis. */
function splitLabel(label: string): { pkg: string; name: string } {
  const i = label.lastIndexOf(":");
  if (i <= 0) return { pkg: label, name: "" };
  return { pkg: label.slice(0, i), name: label.slice(i) };
}

function plural(n: number, word: string): string {
  return `${n.toLocaleString()} ${word}${n === 1 ? "" : "s"}`;
}

function entranceStyle(order: number): CSSProperties {
  return { "--enter-delay": `${Math.min(order * 18, 320)}ms` } as CSSProperties;
}

/** Lets long labels wrap after "/" instead of mid-word. */
function withBreaks(path: string) {
  return path.split("/").map((part, i, all) => (
    <Fragment key={i}>
      {part}
      {i < all.length - 1 && (
        <>
          /<wbr />
        </>
      )}
    </Fragment>
  ));
}

export function Label({ label }: { label: string }) {
  const { pkg, name } = splitLabel(label);
  return (
    <span className="t-label" title={label}>
      <span className="t-pkg">{withBreaks(pkg)}</span>
      <span className="t-name">{name}</span>
    </span>
  );
}

export const TargetNode = memo(function TargetNode({ data }: NodeProps<GraphNode & { type: "target" }>) {
  const { target, role, dir, order } = data as TargetNodeData;
  const actions = useContext(ActionsContext);

  if (role === "focus") return <FocusNode target={target} dir={dir} />;

  // Dependents keep leading up; deps and roots keep leading down.
  const up = role === "parent";
  const onward = up ? target.parentCount : target.depCount;
  const drillable = onward > 0;
  const body = (
    <>
      <div className="t-meta">
        {target.kind && <span className="t-kind">{target.kind}</span>}
        {target.external && <ExternalBadge />}
      </div>
      <Label label={target.label} />
    </>
  );
  const stub = drillable && (
    <div className={`t-stub${up ? " t-stub-up" : ""}`} aria-hidden="true">
      <span className="t-stub-line" />
      <span className="t-stub-count">
        {up ? (
          <>
            <ChevronUp size={12} strokeWidth={2.25} />
            used by {onward.toLocaleString()}
          </>
        ) : (
          <>
            {plural(onward, "dep")}
            <ChevronDown size={12} strokeWidth={2.25} />
          </>
        )}
      </span>
    </div>
  );

  return (
    <div className="t-wrap" style={{ width: NODE_W }}>
      {role === "child" && (
        <>
          <Handle id="top" type="target" position={Position.Top} isConnectable={false} />
          <Handle id="side" type="target" position={Position.Left} isConnectable={false} style={{ top: 34 }} />
        </>
      )}
      {up && (
        <>
          <Handle id="bottom" type="source" position={Position.Bottom} isConnectable={false} />
          <Handle id="side-out" type="source" position={Position.Left} isConnectable={false} style={{ top: 34 }} />
        </>
      )}
      <div className="t-enter" style={entranceStyle(order)}>
        {drillable ? (
          <button
            type="button"
            className="t-node t-drill nodrag"
            onClick={() => actions.open(target.label, up ? "up" : "down")}
            onFocus={() => actions.hover(target.label)}
            onBlur={() => actions.hover(null)}
            aria-label={
              up
                ? `Open ${target.label}, used by ${plural(onward, "target")}`
                : `Open ${target.label}, ${plural(onward, "direct dep")}`
            }
          >
            {body}
          </button>
        ) : (
          <div className="t-node t-leaf" aria-label={`${target.label}, ${up ? "nothing depends on it" : "no deps"}`}>
            {body}
          </div>
        )}
        {stub}
      </div>
    </div>
  );
});

/**
 * The layer's target. It carries a stub that flips the layer: "used by N"
 * above it while looking at its deps, "N deps" below it while looking at its
 * dependents.
 */
function FocusNode({ target, dir }: { target: Target; dir: Direction }) {
  const actions = useContext(ActionsContext);
  const up = dir === "up";
  return (
    <div className="t-focus-wrap" style={{ width: FOCUS_W }}>
      {!up && target.parentCount > 0 && (
        <button
          type="button"
          className="t-stub t-stub-up t-stub-btn nodrag"
          onClick={() => actions.open(target.label, "up")}
          title={`Show the ${plural(target.parentCount, "target")} that depend on it directly`}
        >
          <span className="t-stub-line" />
          <span className="t-stub-count">
            <ChevronUp size={12} strokeWidth={2.25} />
            used by {target.parentCount.toLocaleString()}
          </span>
        </button>
      )}
      <div className="t-node t-focus">
        <div className="t-meta">
          {target.kind && <span className="t-kind">{target.kind}</span>}
          {target.external && <ExternalBadge />}
          <span className="t-counts">
            {plural(target.depCount, "direct dep")}
            {target.parentCount > 0 && <> · used by {target.parentCount.toLocaleString()}</>}
          </span>
        </div>
        <Label label={target.label} />
      </div>
      {up && target.depCount > 0 && (
        <button
          type="button"
          className="t-stub t-stub-down t-stub-btn nodrag"
          onClick={() => actions.open(target.label, "down")}
          title={`Show its ${plural(target.depCount, "direct dep")}`}
        >
          <span className="t-stub-line" />
          <span className="t-stub-count">
            {plural(target.depCount, "dep")}
            <ChevronDown size={12} strokeWidth={2.25} />
          </span>
        </button>
      )}
      <Handle id="down" type="source" position={Position.Bottom} isConnectable={false} />
      <Handle id="gutter" type="source" position={Position.Bottom} isConnectable={false} style={{ left: 22 }} />
      <Handle id="top-in" type="target" position={Position.Top} isConnectable={false} />
      <Handle id="gutter-in" type="target" position={Position.Top} isConnectable={false} style={{ left: 22 }} />
    </div>
  );
}

export function ExternalBadge() {
  return (
    <span className="t-ext" title="Target from an external repository">
      <Globe size={11} strokeWidth={2} />
      external
    </span>
  );
}

export const MoreNode = memo(function MoreNode({ data }: NodeProps<GraphNode & { type: "more" }>) {
  const { remaining, loading, order } = data as MoreNodeData;
  const actions = useContext(ActionsContext);
  return (
    <div className="t-wrap t-enter" style={{ ...entranceStyle(order), width: NODE_W }}>
      <button type="button" className="t-node t-more nodrag" onClick={actions.loadMore} disabled={loading}>
        {loading ? (
          <LoaderCircle size={16} className="spin" />
        ) : (
          <Plus size={16} strokeWidth={2.25} />
        )}
        <span>
          {loading ? "Loading…" : <>Show {Math.min(remaining, 200).toLocaleString()} more</>}
          <small>{remaining.toLocaleString()} not loaded yet</small>
        </span>
      </button>
    </div>
  );
});

/** Stands in the item slot when the focus has none: the layer ends here. */
export const EmptyNode = memo(function EmptyNode({ data }: NodeProps<GraphNode & { type: "empty" }>) {
  const { label, dir } = data as EmptyNodeData;
  const name = <code>{splitLabel(label).name || label}</code>;
  return (
    <div className="t-wrap" style={{ width: NODE_W }}>
      {dir === "up" ? (
        <Handle id="bottom" type="source" position={Position.Bottom} isConnectable={false} />
      ) : (
        <Handle id="top" type="target" position={Position.Top} isConnectable={false} />
      )}
      <div className="t-enter">
        <div className="t-node t-empty" role="status">
          {dir === "up" ? (
            <>
              <strong>Not used</strong>
              <span>No other target depends on {name}.</span>
            </>
          ) : (
            <>
              <strong>No deps</strong>
              <span>{name} doesn't depend on any other target.</span>
            </>
          )}
        </div>
      </div>
    </div>
  );
});

export const nodeTypes = { target: TargetNode, more: MoreNode, empty: EmptyNode };
