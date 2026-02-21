# `polyn` Package Review

## Scope
This document summarizes the Go sub-package `github.com/npillmayer/arithm/polyn` as it exists today in:
- `/Users/npi/prg/go/arithm/polyn/polyn.go`
- `/Users/npi/prg/go/arithm/polyn/lineq.go`
- `/Users/npi/prg/go/arithm/polyn/lineq_test.go`

## Purpose / Intent
`polyn` implements two related concerns:
1. **Linear polynomial arithmetic** on expressions of form `c + a1*x1 + ... + an*xn`.
2. **Incremental solving of linear equation systems** (`0 = p`) inspired by MetaFont/MetaPost style constraint solving.

The solver maintains a set of dependent equations and solved variables, substitutes newly learned equations/values, and can notify client code via a `VariableResolver` callback when a variable becomes solved.

## High-Level Structure

### 1) Polynomial core (`polyn.go`)
Main data model:
- `type Polynomial struct { Terms *treemap.Map }`
- `Terms` maps term index `int` to coefficient `float64`, where index `0` is the constant term.

Key exported operations:
- Construction: `NewConstantPolynomial`, `New` (with helper `X{I,C}` terms)
- Algebra: `Add`, `Subtract`, `Multiply`, `Divide`, `Zap`
- Queries: `IsConstant`, `IsVariable`, `GetCoeffForTerm`, `GetConstantValue`, `IsValid`
- Formatting: `String`, `TraceString`, `TraceStringVar`

Implementation notes:
- Uses `github.com/emirpasic/gods/maps/treemap` (pre-generics era ordered map).
- Uses `float64` + epsilon helpers from `/Users/npi/prg/go/arithm/arithm.go` for rounding/zero checks.
- Uses panic in some invalid-state paths.

### 2) Linear equation solver (`lineq.go`)
Main type:
- `type LinEqSolver struct` with:
- `dependents`: map of unsolved equations (`x.i = polynomial`)
- `solved`: map of solved variables (`x.i = constant polynomial`)
- optional `VariableResolver`

Core workflow:
- `AddEq` / `AddEqs` normalize a new equation, substitute solved values, choose a pivot variable (max coefficient heuristic), activate equation toward that variable, then propagate substitutions across dependent equations.
- When RHS collapses to constants, values are moved into `solved` and reported to resolver via `SetVariableSolved`.
- “Capsule” cleanup (`harvestCapsules`) attempts to remove equations involving out-of-scope vars reported by resolver.

### 3) Tests (`lineq_test.go`)
Single test file covers both polynomial operations and solver flows.

## Public API Snapshot

### Types and interfaces
- `type X struct { I int; C float64 }`
- `type Polynomial struct { Terms *treemap.Map }`
- `type LinEqSolver struct { ... }`
- `type VariableResolver interface { GetVariableName(int) string; SetVariableSolved(int,float64); IsCapsule(int) bool }`

### Constructors / helpers
- `New(c float64, tms ...X) (Polynomial, error)`
- `NewConstantPolynomial(c float64) Polynomial`
- `NewLinEqSolver() *LinEqSolver`
- `ArityComparator(polyn1, polyn2 interface{}) int`
- `T() tracing.Trace`

### Main methods
- Polynomial ops: `SetTerm`, `Add`, `Subtract`, `Multiply`, `Divide`, `Zap`, `CopyPolynomial`
- Polynomial checks/format: `IsConstant`, `IsVariable`, `IsValid`, `String`, `TraceString`
- Solver ops: `SetVariableResolver`, `AddEq`, `AddEqs`, `VarString`, `PolynString`, `Dump`

## Testing Situation

### Current status
- `go test ./polyn` passes.
- `go test ./...` passes for whole repo.
- Statement coverage for `polyn`: **74.6%**.

### What is covered well
- Basic polynomial construction/arithmetic.
- Substitution behavior in several solver paths.
- Contradiction panic path (`equation off by ...`) in `TestLEQ3`.
- Several small multi-equation scenarios.

### What is weak or untested
Functions at 0% coverage include:
- `/Users/npi/prg/go/arithm/polyn/polyn.go`: `Subtract`, `IsVariable`, `IsValid`, `ArityComparator`, `TraceStringVar`
- `/Users/npi/prg/go/arithm/polyn/lineq.go`: `getSolvedVars`, `AddEqs`, `retractVariable`, `Dump`

Additional testing gaps:
- No fuzz/property tests for numeric stability.
- No benchmark coverage for algorithmic hot paths.
- Panic/error behavior is only partially validated and not always strongly asserted.

## Quality Assessment

### Strengths
- Clear conceptual separation between polynomial arithmetic and LEQ solver.
- Incremental solver design is practical for constraint-style workflows.
- Consistent use of epsilon-based rounding helpers to mitigate floating point noise.
- Existing tests exercise meaningful end-to-end solver behavior, not only unit fragments.

### Main issues (idioms, design, maintainability)
1. **Go API naming and style are dated/non-idiomatic**
- Several comments contain outdated terms/typos (“polygon”, “polynomias”, “detructive”).

2. **Encapsulation leak in core model**
- `Polynomial.Terms` is exported and mutable (`*treemap.Map`), so invariants can be broken externally.

3. **Mutation semantics are hard to reason about**
- Many methods use value receivers but mutate reference-backed state (`Terms` map).
- “`destructive bool`” flags create mixed mutability patterns and make call-sites easy to misuse.

4. **Error handling relies heavily on panic**
- Expected/operational states (e.g., inconsistent equations, illegal divisor, unsupported multiply) panic instead of returning errors.
- This is acceptable for internal algorithms but rough for a reusable package API.

5. **Algorithm/documentation mismatch and ambiguity**
- Package header says “linear polynomials,” but helper `X.I` is described as exponent and examples suggest generalized terms; in practice this acts like variable index, not polynomial degree arithmetic.

6. **Data-structure choice is legacy-driven**
- Use of external pre-generics treemap introduces type assertions (`interface{}` casts) and extra dependency weight.

7. **Known performance debt is acknowledged but unresolved**
- `retractVariable` / capsule harvesting includes TODO noting inefficiency (multiple scans/nested loops).

8. **Observability mixed into logic**
- Solver logic is interleaved with tracing/log calls, increasing cognitive load and coupling to tracing infrastructure.

### Potential correctness / behavior defects
1. **Likely typo-bug in `ArityComparator` nil handling**
- In `/Users/npi/prg/go/arithm/polyn/polyn.go`, `ArityComparator` checks `if p1.Terms == nil { if p1.Terms == nil { ... } }`; the inner check appears intended to be `p2.Terms == nil`.

2. **Unexpected input mutation in arithmetic methods**
- `Multiply` and `Divide` can mutate the second operand (`p2`) as part of computation, even when callers may expect non-destructive behavior.
- This behavior is partly documented but still easy to misuse because mutability is not explicit in signatures.

3. **Partial-construction pattern in `New`**
- `New` returns a partially built polynomial plus only the last encountered error when term indices are invalid.
- This can hide earlier invalid inputs and makes error handling ambiguous for callers.

## Suggested Modernization Roadmap

1. **API cleanup (non-breaking first)**
- Use idiomatic constructor name `NewLinEqSolver` and remove the old constructor name (breaking change accepted per remark).
- Clarify docs around exponent (domain model) vs key/index (implementation model) to avoid conflating concept and storage.

2. **Encapsulation and mutability pass**
- Make polynomial terms unexported.
- Normalize to either immutable-returning operations or explicit pointer-receiver mutation APIs; avoid dual-mode `destructive` flags.  ( **Remark**: I strongly prefer immutability and pure functions, but not sure it will always be sensible here )

3. **Error model pass**
- Convert panic-prone public operations to `(..., error)` variants.
- Keep panic only for true internal invariants.

4. **Testing expansion**
- Add focused tests for currently uncovered functions.
- Add property/fuzz tests (e.g., algebraic identities within epsilon tolerance).
- Add capsule-related tests using a resolver with real `IsCapsule` behavior.

5. **Optional post-generics refactor**
- Replace `gods/treemap` with Go-native structures (`map[int]float64` + sorted key slices when needed), or a generic ordered map if ordering is essential.

## Overall
`polyn` is a functional and interesting early package with a solid mathematical intent and working tests, but it shows typical first-package traits: mixed mutability semantics, panic-heavy control flow, weak encapsulation, and partial test coverage. It is a good candidate for a staged modernization that preserves behavior while improving API ergonomics and long-term maintainability.

## Appendix: Step 5 Plan (Replace `gods/treemap`)

### Current shape (confirmed)
- There is no direct map-of-maps value type in one container.
- There is an effective nested structure:
- solver maps are `int -> Polynomial`
- each `Polynomial` internally stores terms as `int -> float64`
- Ordering is semantically important for deterministic behavior (`String`/`TraceString`, `Dump`, `IsVariable` positional logic, and some tie-break behavior in scans).

### Target typed model
1. Introduce typed aliases:
```go
type TermMap map[int]float64
type EquationMap map[int]Polynomial   // variable id -> RHS polynomial
type SolvedMap map[int]Polynomial     // variable id -> constant polynomial
```
2. Add one shared ordered-key helper:
```go
func sortedKeys[T any](m map[int]T) []int
```
3. Keep all public semantics unchanged in first migration pass.

### Migration phases
1. Add adapters first (no behavior change):
- Add `forEachTermAscending`, `forEachEquationAscending`, and `size` helpers.
- Route existing iteration sites through helpers while still backed by treemap.

2. Swap polynomial storage:
- Replace `Polynomial.terms *treemap.Map` with `Polynomial.terms map[int]float64`.
- Replace all term `Get/Put/Remove/Keys/Iterator/Size` with typed map operations plus sorted keys where order is required.

3. Swap solver storage:
- Replace `dependents` and `solved` with typed `map[int]Polynomial`.
- Replace temporary treemaps `D`, `S`, `eqs` with typed maps.
- For every loop that previously relied on treemap ordering, iterate over `sortedKeys(...)`.

4. Remove legacy interfaces:
- Remove `github.com/emirpasic/gods/maps` and `treemap` imports.
- Update signatures currently exposing `maps.Map`/`*treemap.Map` (`maxCoeff`, `getSolvedVars`, `updateDependency`, `updateDependentVariables`, `substituteSolved`) to typed maps.

5. Determinism safeguards:
- Keep explicit ascending-key iteration in:
- `TraceString`/`String`
- `Dump`
- `IsVariable` (replace `positions[1]` logic with sorted exponent list)
- any comparator or scan path that depends on deterministic traversal

### Implementation progress
- Step 5.1: completed.
- Step 5.2: completed.
- Step 5.3: completed.
- Step 5.4: completed (legacy `maps.Map`/`treemap` signatures removed from `polyn` package code).
- Step 5.5: completed (determinism safeguards/regression tests for ordering and `maxCoeff` tie-behavior).

### Test and verification plan
1. Before migration, capture baseline snapshots:
- `go test ./polyn`
- `go test ./polyn -coverprofile=/tmp/polyn.cover.out`
- save representative `Dump` and `TraceString` outputs for fixed inputs

2. During migration, keep tests green after each phase.

3. After migration:
- Run `go test ./...`
- Add focused tests for deterministic ordering in `TraceString` and `Dump`.
- Add one regression test for `maxCoeff` tie behavior to lock current deterministic outcome.

### Risk notes
- Biggest risk is silent behavior drift from implicit treemap ordering assumptions.
- Main mitigation is to make ordering explicit via `sortedKeys` everywhere order matters.
- This refactor will significantly reduce runtime type assertions and improve reasoning/type safety.
