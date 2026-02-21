# `jhobby` Refactor Baseline

Captured on: `2026-02-21`

## Command Baseline
- `go test ./jhobby` -> pass
- `go test ./jhobby -coverprofile=/tmp/jhobby.before-refactor.cover.out` -> pass, `90.9%` statements
- `go test ./...` -> pass

## Stored Artifacts
- Coverage function report: `/Users/npi/prg/go/arithm/doc/baseline/jhobby.cover.func.txt`
- Package API snapshot (`go doc ./jhobby`): `/Users/npi/prg/go/arithm/doc/baseline/jhobby.go-doc.txt`
- Concrete type API snapshot (`go doc ./jhobby.Path`): `/Users/npi/prg/go/arithm/doc/baseline/jhobby.path.go-doc.txt`

## Behavior-Lock Tests Added
The following tests were added to lock current behavior before interface/API refactor:
- `/Users/npi/prg/go/arithm/jhobby/path_test.go:49` `TestAsStringSnapshots`
- `/Users/npi/prg/go/arithm/jhobby/path_test.go:194` `TestControlsDeterministicSnapshot`
- `/Users/npi/prg/go/arithm/jhobby/path_test.go:270` `TestSegmentsSplitBaseline`
- `/Users/npi/prg/go/arithm/jhobby/path_test.go:307` `TestEmptyPathJoinPanics`

Helper:
- `/Users/npi/prg/go/arithm/jhobby/path_test.go:13` `mustPanic`

## Locked Snapshot Values
- Open skeleton string: `(1,1) .. (2,2) .. (3,1)`
- Cycle skeleton string: `(1,1) .. (2,2) .. (3,1) .. (2,0) .. cycle`
- Control points (epsilon `0.0002`) for 4-knot cycle:
- post[0] ~ `(1.0000, 1.5523)`
- pre[1] ~ `(1.4477, 2.0000)`
- post[2] ~ `(3.0000, 0.4477)`
- Segment splitting:
- non-rough sample path -> `1` segment
- explicit rough knot via `SetPreCurl(1, 2.0)` on 3-knot open path -> `2` segments with bounds `[0,1]`, `[1,2]`
- Empty-path join behavior remains panic-locked for now:
- `Nullpath().Line()`
- `Nullpath().Curve()`
- `Nullpath().TensionCurve(...)`

