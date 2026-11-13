---
name: Bazel Graph
description: A dark, pannable canvas that shows a Bazel workspace one dependency layer at a time.
colors:
  bg: "#0f1115"
  panel: "#171a20"
  panel-2: "#1d2128"
  panel-hover: "#20252e"
  line: "#262a33"
  line-strong: "#353b47"
  ink: "#e6e8eb"
  ink-2: "#b4bac4"
  mute: "#8b919c"
  accent: "#4c8dff"
  accent-ink: "#9cc0ff"
  accent-bg: "#15213a"
  ok: "#3fb67a"
  warn: "#e0a83a"
  bad: "#f07067"
  grid-dot: "#262b35"
  edge: "#3a404c"
  mini-mask: "rgba(15, 17, 21, 0.72)"
typography:
  title:
    fontFamily: "ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "15px"
    fontWeight: 600
    lineHeight: 1.45
    letterSpacing: "-0.01em"
  heading:
    fontFamily: "ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "14px"
    fontWeight: 600
    lineHeight: 1.45
    letterSpacing: "-0.01em"
  body:
    fontFamily: "ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "13px"
    fontWeight: 400
    lineHeight: 1.45
  crumb:
    fontFamily: "ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "12.5px"
    fontWeight: 400
    lineHeight: 1.45
  body-small:
    fontFamily: "ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "12px"
    fontWeight: 400
    lineHeight: 1.45
    fontFeature: "tnum"
  meta:
    fontFamily: "ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "11px"
    fontWeight: 400
    lineHeight: 1.45
    fontFeature: "tnum"
  label-mono:
    fontFamily: "ui-monospace, SF Mono, JetBrains Mono, Cascadia Code, Menlo, Consolas, monospace"
    fontSize: "12px"
    fontWeight: 400
    lineHeight: 1.35
  label-mono-focus:
    fontFamily: "ui-monospace, SF Mono, JetBrains Mono, Cascadia Code, Menlo, Consolas, monospace"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: 1.35
  kind-mono:
    fontFamily: "ui-monospace, SF Mono, JetBrains Mono, Cascadia Code, Menlo, Consolas, monospace"
    fontSize: "11px"
    fontWeight: 400
    lineHeight: 1.45
rounded:
  crumb: "6px"
  control: "8px"
  overlay: "10px"
  node: "12px"
  panel: "14px"
  pill: "999px"
spacing:
  xxs: "2px"
  xs: "4px"
  sm: "8px"
  md: "12px"
  lg: "16px"
  xl: "20px"
  grid: "22px"
  node-gap-x: "28px"
  node-gap-y: "36px"
  focus-gap: "96px"
components:
  topbar:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.ink}"
    height: "48px"
    padding: "0 12px 0 16px"
  crumb:
    backgroundColor: "transparent"
    textColor: "{colors.ink-2}"
    typography: "{typography.crumb}"
    rounded: "{rounded.crumb}"
    padding: "4px 8px"
  crumb-hover:
    backgroundColor: "{colors.panel-2}"
    textColor: "{colors.ink}"
  crumb-current:
    backgroundColor: "{colors.accent-bg}"
    textColor: "{colors.accent-ink}"
  target-node:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.ink}"
    typography: "{typography.label-mono}"
    rounded: "{rounded.node}"
    padding: "10px 12px 11px"
    width: "248px"
    height: "68px"
  target-node-hover:
    backgroundColor: "{colors.panel-hover}"
  target-node-leaf:
    backgroundColor: "transparent"
    textColor: "{colors.ink-2}"
    rounded: "{rounded.node}"
  target-node-focus:
    backgroundColor: "{colors.accent-bg}"
    textColor: "{colors.ink}"
    typography: "{typography.label-mono-focus}"
    rounded: "{rounded.node}"
    padding: "12px 16px 14px"
    width: "360px"
  dep-stub:
    backgroundColor: "transparent"
    textColor: "{colors.mute}"
    typography: "{typography.meta}"
    rounded: "{rounded.pill}"
    padding: "1px 6px 1px 8px"
  dep-stub-hover:
    textColor: "{colors.accent-ink}"
  show-more-node:
    backgroundColor: "transparent"
    textColor: "{colors.ink-2}"
    rounded: "{rounded.node}"
    width: "248px"
  icon-button:
    backgroundColor: "transparent"
    textColor: "{colors.ink-2}"
    rounded: "{rounded.control}"
    size: "30px"
  icon-button-hover:
    backgroundColor: "{colors.panel-2}"
    textColor: "{colors.ink}"
  button:
    backgroundColor: "{colors.panel-2}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.control}"
    padding: "6px 12px"
  status-panel:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.ink-2}"
    rounded: "{rounded.panel}"
    padding: "18px 20px"
    width: "460px"
  toast:
    backgroundColor: "{colors.panel-2}"
    textColor: "{colors.ink}"
    rounded: "{rounded.overlay}"
    padding: "9px 14px"
  layer-caption:
    backgroundColor: "{colors.bg}"
    textColor: "{colors.ink}"
    typography: "{typography.title}"
    rounded: "{rounded.overlay}"
    padding: "6px 10px 7px"
  warn-chip:
    textColor: "{colors.warn}"
    typography: "{typography.meta}"
    rounded: "{rounded.pill}"
    padding: "2px 8px"
---

# Design System: Bazel Graph

## Overview

**Creative North Star: "The Single Lit Layer"**

The viewer is the category-standard dark graph canvas, executed straight: a near-black dotted field, raised graphite cards, hairline borders, and one blue accent. Everything on the canvas belongs to one layer of the build graph: the focused target sits at the top, ringed and filled in accent, and its direct deps hang below it in a centred wrapped grid. Everything else is cleared away. The accent marks "where you are"; nothing else on screen competes with it.

The density is that of a developer tool that lives beside a dark editor: 13px system UI for chrome, monospace for every Bazel label and rule kind, tabular numerals for every count. Surfaces are tonal, not shadowed; depth comes from three close steps of graphite (bg, panel, panel-2) separated by 1px lines. The one drop shadow belongs to the focused node, lifting it over the edges it emits.

Motion carries the navigation. On click, the other nodes fade out, the clicked node glides into the focus slot while the camera moves with it, and the new deps fade down into place with a short stagger. Reduced-motion users get the same states without the travel.

**Key Characteristics:**
- Near-black dotted canvas (22px grid) with graphite cards and 1px hairlines
- One blue accent reserved for focus, the current breadcrumb, hovered edges, and keyboard focus
- System UI for chrome, monospace for Bazel labels and rule kinds, tabular numerals for counts
- Tonal layering instead of shadows, except the focused node
- Drillable versus leaf is shown by fill and by a dotted stub with a count pill, never by colour alone
- Navigation is animated as a glide plus fade on a single ease-out curve

## Colors

A cool, near-neutral graphite ramp with a single saturated blue and three semantic status hues.

### Primary
- **Signal Blue** (accent): the focus ring of the focused node, the current breadcrumb's text partner, a hovered dep's edge, keyboard focus outlines, the busy status dot, and the focused target's minimap rectangle.
- **Pale Signal** (accent-ink): text set on accent-tinted ground (current breadcrumb, the focused node's package path and dep counts) and the dep stub when its node is hovered.
- **Deep Signal Wash** (accent-bg): fill of the focused node and the current breadcrumb chip.

### Neutral
- **Canvas Black** (bg): the canvas, the page body, the layer caption's backing, and code wells inside panels.
- **Graphite** (panel): top bar, drillable nodes, status panels, zoom controls, minimap.
- **Graphite Raised** (panel-2): hover fill for crumbs and icon buttons, the retry button, toasts.
- **Graphite Hover** (panel-hover): hover fill of a drillable node.
- **Hairline** (line): default 1px border on every card, bar and control.
- **Hairline Strong** (line-strong): hovered node borders, the dotted dep stub, dashed "show more" and toast borders, scrollbar thumb.
- **Grid Dot** (grid-dot): the canvas dot grid.
- **Edge Grey** (edge): resting dependency edges and minimap node rectangles.
- **Ink** (ink): primary text and target names.
- **Ink Soft** (ink-2): secondary text, leaf names, crumbs at rest, panel body copy.
- **Mute** (mute): rule kinds, package paths, captions, status text, separators.
- **Minimap Mask** (mini-mask): the area outside the minimap viewport.

### Status
- **OK Green** (ok): the ready status dot.
- **Amber** (warn): "Incomplete graph" chip (on a 12% amber wash), the external-repository badge, the reload-failed dot.
- **Coral** (bad): unreachable / failed status text and dot, error text in code wells, and a 45% coral border on error panels.

### Named Rules
**The One Accent Rule.** Blue means "here": the focused target, the current breadcrumb, the edge you are pointing at, the element with keyboard focus. Never use it for decoration or for a second category.

**The Status Is Never Hidden Rule.** Loading, stale, incomplete and error states always have a visible colour-plus-text treatment (dot and sentence, chip, or panel); colour never carries the state alone.

## Typography

**Body Font:** system UI stack (ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto)
**Label/Mono Font:** ui-monospace stack (SF Mono, JetBrains Mono, Cascadia Code, Menlo, Consolas)

**Character:** The chrome speaks in the platform's own voice so it sits quietly beside the editor; Bazel's words (labels, rule kinds) are always monospace, so anything copyable from `bazel query` looks the way Bazel prints it. There is no display face: the largest text on screen is the 15px caption.

### Hierarchy
- **Title** (600, 15px, -0.01em): the layer caption over the canvas ("Workspace roots", "Deps of :name").
- **Heading** (600, 14px, -0.01em): product name in the top bar and status panel headings.
- **Body** (400, 13px, 1.45): base text, panel copy, buttons.
- **Crumb** (400, 12.5px): breadcrumb steps; target steps switch to mono.
- **Body Small** (400, 12px, tabular): status text, caption sub-line.
- **Meta** (400, 11px, tabular): dep counts, stub pills, external badge.
- **Label Mono** (400, 12px, 1.35): target labels in nodes, clamped to two lines.
- **Label Mono Focus** (400, 14px, 1.35): the focused target's label, clamped to three lines; its target name goes white at 600.
- **Kind Mono** (400, 11px): rule kind above each label, ellipsised.

### Named Rules
**The Speak Bazel Rule.** Every label and rule kind is monospace and printed exactly as Bazel reports it. The package path is muted and the `:name` carries the emphasis.

**The Tabular Count Rule.** Every number that can change (targets, deps, used-by, shown of total) uses tabular numerals.

## Layout

One fixed 48px top bar over a full-bleed pannable canvas (CSS grid rows `48px / 1fr`). The top bar is brand, a horizontally scrolling breadcrumb that pins its current step into view, and the status cluster on the right.

A layer is laid out by fixed geometry, not a graph engine: the focus node (360px wide) is centred at the top, and deps (248px wide, 68px tall, plus 40px reserved for the stub) sit 96px below in centred rows. Column count is the item count up to six, then roughly the square root of 2.4 times the count, capped at eight; gaps are 28px horizontal and 36px vertical. The roots view is the same grid without a focus node. The camera frames the focus plus up to three rows, centred, never zoomed past 100% and never below 35% (60% on phones), leaving 76px above for the caption.

Canvas furniture is pinned: the caption top-left (12px inset), zoom controls bottom-left, minimap bottom-right, toasts bottom-centre, and status panels centred in an overlay with a 16px gutter.

At 720px and below the grid collapses to one column laid out as a file tree: deps indent from a left gutter under the focus node and connect with stepped edges. The brand word, the status text and the minimap are hidden, and the breadcrumb folds to "Roots > … > current". Above 720px, trails longer than four steps fold to first, "…", and the last two.

The spacing rhythm is small and tight: 2, 4, 8, 12, 16 and 20px, with 6px and 10px used inside chips and node padding.

## Elevation & Depth

Depth is tonal. The canvas is the darkest plane, raised elements step up through panel, panel-2 and panel-hover, and every edge between planes is a 1px hairline. Leaves drop their fill entirely and sit as outlines on the canvas, so drillable nodes read as raised and leaves as flat.

### Shadow Vocabulary
- **Focus lift** (`box-shadow: 0 10px 28px -8px rgba(0, 0, 0, 0.7)`): only on the focused node, lifting it over the edges it emits.

### Named Rules
**The Flat Except Focus Rule.** No card, bar, panel or control casts a shadow. The focused node is the single lifted object on the canvas.

## Shapes

Soft, uniform rounding that grows with the size of the object: 6px crumbs, 8px controls (buttons, zoom stack, minimap, code wells), 10px floating overlays (caption, toast), 12px nodes, 14px status panels, and full pills for chips, the dep count and status dots. Borders are 1px solid at rest; dashed borders mean "not a target" or "more below" (the dep stub pill, the "show more" node, the "No deps" placeholder), and a 1.5px dotted vertical line drops from every drillable node to its count pill. Edges are smooth Bézier curves from the focus node's bottom centre; the no-deps edge is dashed (3 4).

## Components

### Target node (drillable)
Graphite card, 248px wide, 12px radius, 1px hairline. Rule kind in mono meta above a mono label clamped to two lines. It is a real button: hover and keyboard focus lift the border to Hairline Strong and the fill to Graphite Hover, and hovering also turns that dep's incoming edge Signal Blue at 2px. Below it hangs the dep stub: a dotted line and a dashed pill reading "N deps" with a chevron, which turn accent on hover.

### Leaf node
Same geometry with no fill and Ink Soft name text, not interactive, no stub. The absence of fill and stub is the "no deeper layer" signal.

### Focus node
360px, Deep Signal Wash fill, Signal Blue 1px border, the focus lift shadow, larger padding (12px 16px 14px). The meta line carries the rule kind on the left and "N direct deps · used by M" in Pale Signal on the right. Package path in Pale Signal at 80% opacity; target name white at 600.

### Show-more and No-deps nodes
Node-sized, dashed, unfilled. "Show N more" with a plus icon and "N not loaded yet" in Mute; its border and text turn accent on hover and it shows a spinner while loading. "No deps" is a status card standing in the child slot, joined to the focus by a dashed edge.

### Breadcrumb
Text buttons in Ink Soft with a 14px chevron separator in Mute. Hover fills Graphite Raised. The current step is a chip in Deep Signal Wash with Pale Signal text. Target steps are mono and ellipsise at 320px (220px on phones).

### Status cluster
A 7px dot plus a sentence: green when ready ("17 targets · loaded 08:55 PM"), pulsing blue while loading or reloading, amber when a reload failed but the old graph is still shown, coral when unreachable or failed. An amber "Incomplete graph" pill appears when the graph carries a warning. A 30px ghost icon button reloads the graph; its icon spins while busy.

### Buttons
- **Shape:** gently rounded (8px).
- **Default:** Graphite Raised fill, Hairline Strong 1px border, 6px 12px padding, icon plus label.
- **Hover:** border turns Signal Blue.
- **Icon button:** 30px square, transparent at rest; hover gains Graphite Raised fill and a hairline.
- **Focus:** every focusable element gets a 2px Signal Blue outline at 2px offset (3px on nodes).

### Status panel
Centred overlay card, max 460px, Graphite with a hairline and 14px radius, 18px 20px padding, a 14px heading with an optional spinner. Errors swap to a coral border and show the raw error in a Canvas Black code well in coral mono.

### Canvas furniture
Dot grid in Grid Dot, 22px gap, 1.2px dots. Zoom controls are a 30px stacked column with hairline dividers, 8px radius. The minimap draws the focus in Signal Blue and everything else in Edge Grey.

### Motion
One curve, `cubic-bezier(0.16, 1, 0.3, 1)`. Node position glides take 460ms with the camera moving in step; leaving nodes fade in 180ms; entering nodes drop 10px and fade in over 380ms with an 18ms per-node stagger capped at 320ms. Hover colour changes take 120 to 150ms. Under reduced motion, node transitions, entrances and toasts are static and the camera jumps.

## Do's and Don'ts

### Do:
- **Do** keep Signal Blue for the focused target, the current breadcrumb, the hovered edge and keyboard focus only.
- **Do** set every Bazel label and rule kind in the mono stack, with the package path muted and the `:name` emphasised.
- **Do** show "deeper layer" with the dotted stub and dashed count pill, and "leaf" with an unfilled card and no stub.
- **Do** separate planes with the graphite steps and 1px hairlines; reserve the focus lift shadow for the focused node.
- **Do** pair every status colour with words (dot plus sentence, chip, or panel).
- **Do** animate navigation with the single ease-out curve and turn the travel off under reduced motion.

### Don't:
- **Don't** draw more than one layer: no grandchildren, no full-graph view.
- **Don't** introduce a second accent hue or use the status colours decoratively.
- **Don't** add shadows to cards, bars or panels.
- **Don't** rename or prettify Bazel labels and rule kinds.
