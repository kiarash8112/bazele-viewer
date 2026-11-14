# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

Backend: Go, built with Bazel (`rules_go`), entry point `cmd/viewer`.

Frontend: React + React Flow (`@xyflow/react`), built with Vite. Each layer is a star (one target and its direct deps), so it is laid out as a centred, wrapped grid computed in `web/src/layout.ts` rather than with a general graph-layout engine. The user chose this over a no-build option because the interactive canvas needs a real graph renderer. The Vite build output is embedded in the Go binary via `go:embed` (Bazel `embedsrcs`) and served by the same HTTP server as `/api/*`, so running it stays one `bazel run`. A Node toolchain is needed only to build the frontend.

## Users

Engineers working in their own Bazel monorepo checkout. They open the viewer while coding or debugging a build to understand what a target depends on, without leaving their local workflow.

## Product Purpose

Explore a Bazel workspace's dependency graph on an interactive canvas you can pan and zoom, with a node-link drawing that loads on demand. The user starts at the workspace's entry points (binaries, or other rule targets when there are none) and drills down. Clicking a target opens a fresh, uncluttered page: that target in focus with only its direct deps laid out below it, and a breadcrumb trail back to where the user started. Each dep that has deps of its own shows a dotted stub with a count, signalling another layer; clicking it drills in again. Targets with no deps are leaves and show no stub. The interaction model was inspired by Understand-Anything's dashboard (https://understand-anything.com/demo/). Only the interaction is borrowed; the look is the viewer's own. Success: an engineer can quickly find and understand a target's dependencies in a large workspace, and the UI stays fast however big the graph is.

## Positioning

Unlike raw `bazel query` output or full-graph renderers that draw the whole hairball, the viewer shows a large graph one layer per page, drilling down wherever the user clicks. The whole graph is loaded once into a compact in-memory index, and the UI only fetches the page of children it needs. So even huge workspaces stay readable and fast to navigate.

## Operating Context

- Started locally with `bazel run //cmd/viewer` (flags: `-addr`, default `127.0.0.1:8090`; `-workspace`, defaulting to `BUILD_WORKING_DIRECTORY`; `-bazel`, defaulting to `bazelisk` when it is on `PATH` so the inspected workspace's `.bazelversion` applies).
- The graph comes from `bazel query '//...:*'`: the main repository's targets and their direct deps. External repositories are not loaded (that would fetch every toolchain they register), so each `@external` target a workspace target depends on is a leaf: its label only, with no rule kind and no deps.
- Used next to the engineer's editor and terminal, against their own checkout.
- The graph loads in the background at startup. The user can trigger a reload after editing BUILD files; until the reload succeeds, requests keep being served from the previous graph.

## Capabilities and Constraints

- HTTP API (`internal/api`):
  - `GET /api/status`: `state` is `loading`, `ready` or `error`, plus `loading`, `targets`, `loadedAt`, `warning` (the graph may be incomplete) and `error`.
  - `POST /api/refresh`: starts a background reload.
  - `GET /api/roots`: the workspace's entry points.
  - `GET /api/children?label=…`: a target's direct dependencies. Unknown labels return 404.
  - `GET /api/hotspots?of=targets|files`: targets (default) or source files ranked by transitive dependents, the targets a change to them affects. A file's dependents are the rules that list it plus their transitive dependents. Computed in parallel on first request (about 25 ms for ~28k targets and ~16k files).
  - Pagination: `offset` and `limit` (default 50, max 1000). Responses include `total`.
  - Filters: `tests` (roots only) and `external` (show `@external` repository targets; roots, children and target hotspots).
  - Every node carries `label`, `kind`, `external`, `depCount` and `parentCount`; hotspot nodes also carry `dependents`. Source files have the kind `source file` and `parentCount` is the number of rules that list them.
  - Until the first load completes, data endpoints return 503.
- The graph index stores reverse dependencies (`parentCount`), but no endpoint for browsing parents exists yet.
- Local only: one user, binds to localhost, no auth or multi-user concerns.
- Terminology: Bazel's own words: target, label, rule kind, deps, external repository, test target.
- Version 1 scope (confirmed): drill-down graph canvas (one layer per page, breadcrumb navigation, pan and zoom).
- Hotspots view (`#hotspots`, `#hotspots/files`): a ranked list of targets or source files by transitive dependents, with direct counts and a bar scaled to the top entry. Clicking a target opens it on the graph canvas. Deferred: details panel and search, grouping by package, and the path finder (which needs a new `/api/path` endpoint).
- Undecided: path-finding ("why does X depend on Y"), reverse-dependency browsing, search by label.

## Evidence on Hand

- This repo includes a small example Go workspace (`cmd/server`, `internal/handler`, `pkg/greeting`, `pkg/strutil`, `pkg/store`) that works as real demo data for the viewer.
- No users, testimonials, benchmarks or performance figures exist. Do not invent any.

## Product Principles

1. Stay fast at any scale. Never fetch or render more of the graph than the user has expanded, and show what remains unloaded, never hide it.
2. Speak Bazel. Show real labels and rule kinds exactly as Bazel reports them, with no renaming or abstraction.
3. Be honest about state. Make loading, stale, incomplete (warning) and error states visible, not hidden.
4. Zero setup. Running it with one `bazel run` command is the whole install.
