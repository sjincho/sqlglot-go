# sqlglot-go — agent guide

A faithful, near-1:1 Go port of **[tobymao/sqlglot](https://github.com/tobymao/sqlglot) v30.12.0**
(a pure-Python SQL parser/transpiler). It exists so a SQL **column-lineage probe** can run natively
on Go instead of Python-on-a-JVM; Milestone 1 targets exactly the sqlglot API surface that probe
uses, on **MySQL + Postgres**.

## Source of truth (READ THIS FIRST, always)

- The pinned Python source is fetched to **`.reference/sqlglot-v30.12.0/`** (gitignored — run
  `scripts/fetch-reference.sh` once). It is the **exact** upstream version being ported
  (`sqlglot==30.12.0`, git SHA in `.reference/sqlglot-v30.12.0/GIT_SHA.txt`).
- Port from this reference, file by file, **as 1:1 as possible** — same file layout, same
  function/method names (Go-cased), same structure, same comments where they carry intent. When Go
  forces a divergence (static typing, no metaclasses, error/panic instead of exceptions), keep it
  minimal and note *why* in a comment that cites the reference line.
- **Port the corresponding unit tests too**, 1:1, from `.reference/sqlglot-v30.12.0/tests/`. The
  upstream tests and `tests/fixtures/*.sql` are the correctness oracle — reuse the `.sql` fixtures
  verbatim (they live under each package's `testdata/`), reimplement the loader/assertions in Go.

## Current status (Milestone 1)

**COMPLETE.** `go test ./...` is green (~122 tests). The probe's full API surface works on MySQL +
Postgres and is verified at **94/94 parity** against the real Python `probe.py` on sqlglot 30.12.0.
See `ROADMAP.md` for the slice-by-slice ledger, every known divergence, and what's deferred.

The probe API that M1 targets (all working):
- `sqlglot.Parse(sql, dialect)` / `ParseOne` and the `expressions` (`exp`) AST.
- `optimizer.Qualify(root, opts)` — the `qualify()` driver (normalize_identifiers → qualify_tables →
  qualify_columns → quote_identifiers → validate_qualify_columns).
- `optimizer.TraverseScope(root)` + the `Scope` API (`.Expression / .Sources / .Parent / .IsUnion /
  .UnionScopes / .Columns`).
- `generator` (`Expression → SQL`), `schema.MappingSchema`, dialect normalization/quoting.
- The lineage probe itself is ported to `probe/probe.go` with a Python-parity harness
  (`probe/parity_test.go` runs the real `probe.py` under the pinned reference; `probe/golden_test.go`
  guards the same output hermetically via committed `probe/testdata/golden.json`).

## JVM binding (`jvm/`)

An in-process JVM binding exposes the probe to Kotlin/Java via the Foreign Function & Memory API.
`cmd/libsqlglot/main.go` is a cgo `c-shared` entry point (`ProbeJSON` / `FreeCString`, backed by
`probe.ProbeJSONSafe` — total, never panics across the boundary); `jvm/` is a Gradle project whose
`buildNativeLib` task compiles that to `libsqlglot.{dylib,so}` and bundles it, with an FFM wrapper
`io.github.sjincho.sqlglot.Sqlglot.probeJson(sql, dialect, schemaJson): String`. Consumers vendor the
repo via `git subtree` + `includeBuild("…/jvm")` — see `docs/USING_FROM_JVM.md`. cgo is confined to
the `cmd/libsqlglot` package; pure-Go consumers of the library never pull it in.

## Central design decision — the AST node model

Upstream `Expression` is dynamically typed: an `args: dict[str, Any]` of children
(node | list | str | bool | None), a per-class `arg_types` map, a metaclass dialect registry, and
heavy reflection (`node.key`, `find_all(*types)` via isinstance). The parser (~10k LOC) and generator
(~6k LOC) manipulate every node generically through `args`. The Go port mirrors this with a **single
`*Node` struct** behind an `Expression` interface, discriminated by a `Kind` enum, with per-Kind
metadata *tables* in `expressions/kinds.go` (ordered arg keys / traits / class name). Adding a node
type = one `Kind` const + one row in each table + a one-line builder — nodes are **data**, not ~300
structs. This keeps the generic parser/generator/optimizer code a close 1:1 of the Python.

## How to continue the port

1. `scripts/fetch-reference.sh` to get the pinned Python source (needed for parity + as the oracle).
2. Read `ROADMAP.md` — it lists the remaining slices (**1d** parser tail, **4c** full
   `annotate_types`, **5b** per-dialect parser/generator override tables) and, crucially, the
   **known divergences** + **resolved-findings** ledger so you don't re-litigate settled decisions.
3. Pick a slice, port from `.reference/` 1:1, port its tests, keep `go test ./...` green.
4. For anything touching the probe path, re-run the parity harness:
   `go test ./probe/` (hermetic) and, with Python available,
   `PROBE_REGEN=1 go test ./probe/ -run TestProbeParity` to re-verify against live Python and refresh
   the goldens. Deferred parser gaps must stay **fail-closed** (an unparseable construct → the probe
   DENYs; never silently resolve).

This port was built with a multi-model review pipeline (plan → implement → integrate → adversarial
review), verifying every review finding against the pinned source before acting. Keep that rigor:
confirm a claimed bug against `.reference/` before "fixing" it — some findings are phantom.

## Conventions

- Go 1.23. Module `github.com/sjincho/sqlglot-go`. Zero third-party deps (stdlib + `testing` only).
- Comments in **English**, US spelling (`canceled`, `color`, `catalog`).
- `gofmt` + `go vet` clean; `go test ./...` green before any commit/push.
- Package layout mirrors the Python module layout (`expressions/`, `optimizer/`, `dialects/`,
  `generator/`, `parser/`, `tokens/`, `schema/`, …).
