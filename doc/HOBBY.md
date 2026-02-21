# `jhobby` Package Inventory

## Scope
This review covers:
- `/Users/npi/prg/go/arithm/jhobby/doc.go`
- `/Users/npi/prg/go/arithm/jhobby/path.go`
- `/Users/npi/prg/go/arithm/jhobby/path_test.go`

Current size:
- `path.go`: 976 LOC
- `path_test.go`: 223 LOC
- `doc.go`: 101 LOC

## Purpose / Intent
`jhobby` implements MetaFont/MetaPost-style path interpolation with John Hobby's spline algorithm.

Main intent:
- Build a skeleton path (knots + optional dir/curl/tension constraints).
- Compute Bezier-like spline control points that produce a smooth curve through knots.
- Support both open and cyclic paths.

## Codebase Structure
- `doc.go`: package-level background and usage narrative.
- `path.go`: all implementation.
- `path.go` contains the public API and builder DSL (`Nullpath()...Knot()...Curve()...End()/Cycle()`).
- `path.go` contains path/control-point data structures.
- `path.go` contains solver pipeline logic for open/cyclic systems.
- `path.go` contains utility math and formatting helpers.
- `path_test.go`: unit tests + one example snapshot output.

## Public API Snapshot
Exported top-level API:
- `func Nullpath() *Path`
- `func FindHobbyControls(path HobbyPath, controls SplineControls) SplineControls`
- `func AsString(path HobbyPath, contr SplineControls) string`

Exported interfaces:
- `HobbyPath` (read-only geometric/path constraint view)
- `SplineControls` (read/write control points)
- `KnotAdder` and `JoinAdder` (builder flow interfaces)

Exported concrete type:
- `type Path struct`
- Builder methods: `Knot`, `SmoothKnot`, `CurlKnot`, `DirKnot`, `Line`, `Curve`, `TensionCurve`, `End`, `Cycle`
- Property methods: `SetPreDir`, `SetPostDir`, `SetPreCurl`, `SetPostCurl`, `SetPreTension`, `SetPostTension`
- Read methods implementing `HobbyPath`: `IsCycle`, `N`, `Z`, `PreDir`, `PostDir`, `PreCurl`, `PostCurl`, `PreTension`, `PostTension`

Notable API detail:
- `Path` exposes `Controls *splcntrls` (field is exported, type is unexported), which leaks internals awkwardly.

## Core Algorithm Flow
High-level flow in `FindHobbyControls`:
1. Split path into smooth segments (`splitSegments`) around rough knots.
2. For each segment, solve theta equations for open paths via `startOpen` -> `buildEqs` -> `endOpen`.
3. For each segment, solve theta equations for cyclic paths via `startCycle` -> `buildEqs` -> `endCycle`.
4. Convert solved angles/tensions into pre/post control points (`setControls`, `controlPoints`).

## Dependencies
Direct dependencies used in `jhobby`:
- `github.com/npillmayer/arithm` (numeric pair/complex helpers)
- `github.com/npillmayer/schuko/tracing`
- `github.com/npillmayer/schuko/gconf`
- stdlib `math`, `math/cmplx`, `fmt`

## Testing Situation
Status:
- `go test ./jhobby` passes.
- `go test ./...` passes.

Coverage:
- `go test ./jhobby -coverprofile=/tmp/jhobby.cover.out` reports `86.6%` statement coverage.

Strongly covered:
- Core solver path (`findSegmentControls`, `solveOpenPath`, `solveCyclePath`, `buildEqs`, `controlPoints`) is mostly well covered.

Coverage gaps (0%):
- `CurlKnot` (`path.go:242`)
- `AppendSubpath` (`path.go:302`)
- `SetPostTension` (`path.go:366`)
- `pathPartial.PreControl` (`path.go:508`)
- `pathPartial.PostControl` (`path.go:512`)
- `equal` (`path.go:974`)

Additional test weaknesses:
- Commented-out tests exist for `TestCurl` and `TestSegments` in `path_test.go`.
- No fuzz/property tests for numeric stability.
- No benchmarks for larger paths / many segments.

## Quality Issues / Risks
1. Incomplete functionality:
- `AppendSubpath` is a stub (`path.go:302`) and only logs an error.

2. Documented feature gap:
- Tension `"at least"` semantics are documented as incomplete (`path.go:286`), and setter behavior currently clamps values into `[0.75, 4.0]`, which prevents negative "at least" encoding from being represented literally.

3. Panic-based error handling for misuse:
- Builder methods `Line`, `Curve`, `TensionCurve` panic on empty path (`path.go:262`, `path.go:273`, `path.go:289`) instead of returning errors.

4. Edge-case safety:
- `Z(i)` performs modulo indexing but does not guard `N()==0`; empty-path calls can panic (`path.go:400`).
- Negative indices rely on `%` behavior and may remain negative before indexing.

5. Global config coupling:
- Algorithm output/logging behavior depends on global config `gconf.IsSet("tracingchoices")` (`path.go:730`, `path.go:838`), which complicates deterministic behavior in shared processes.

6. Encapsulation and maintainability:
- Main implementation is concentrated in one large file (`path.go`), mixing builder API, numerical solving, segment logic, and formatting utilities.
- Exported `Path.Controls` with unexported concrete type is awkward API surface.

7. Project-level caveats still active:
- Package docs explicitly mention slight deviations from MetaFont results and "early phase" maturity.

## Suggested Next Modernization Targets
1. Decide and implement `AppendSubpath` semantics, or remove/deprecate it.
2. Replace panic-on-misuse paths with error-returning API variants.
3. Clarify and implement true `"at least"` tension semantics, with focused tests.
4. Add explicit validation for empty/invalid paths before solving.
5. Add tests for currently uncovered functions and restore/replace commented-out tests.
6. Split `path.go` into focused files (builder, solver, segmentation, utilities) to reduce cognitive load.
