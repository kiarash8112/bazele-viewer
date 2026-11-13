---
version: 1
slug: "web-src-app-tsx"
primary_target: "web/src/App.tsx"
related_targets: []
---

# Graph canvas (web/src)

Scope: the viewer's single screen, the drill-down dependency canvas. Mode: Operate.
Audience: engineers next to a dark editor, debugging a build. Task: start at the roots, click a target, see only its direct deps, keep drilling, step back via breadcrumb.
Constraints: v1 is the graph only (no details panel, no search). Honest loading/error/stale state stays visible. Quality bar: the Understand-Anything demo dashboard.

## Direction contract

THESIS: One layer per page. The canvas never shows more than the focused target and its direct deps; clicking clears everything else. It refuses the full-graph hairball.

OWN-WORLD: Category standard, executed straight: near-black neutral ground (#0f1115), raised panels (#171a20), hairline borders, one blue accent (#4c8dff) for focus, selection and the current breadcrumb. System UI type, monospace for Bazel labels. Dotted canvas grid, rounded 10px nodes, a count badge on nodes with deps.

STORY: The engineer sees the workspace's entry points, picks one, and walks down its deps one layer at a time, always knowing the path back and which deps go deeper.

FIRST VIEWPORT: 48px top bar: product name, breadcrumb chips (current one in accent), status on the right. Full-bleed pannable canvas below: focused target centred at top with an accent ring, direct deps in a centred wrapped grid below (one column on phones), curved edges; minimap bottom-right, zoom controls bottom-left. Roots view: grid of root nodes, no focus node.

FORM: canon (the category standard), chosen by the user over the roll; seed: seed:80d22a98a7bfd8ba. Signature interaction: click → nodes fade out, the clicked node glides to the focus slot, children fade in below.

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
