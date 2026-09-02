# Engineering Change Policy

These instructions apply to every change in this repository. Treat compatibility,
correctness, safety, and measured performance as acceptance criteria, not optional
follow-up work.

## Start with impact

Before editing, identify the affected public API, encoded/decoded data behavior,
hot paths, allocations, concurrency, and callers. Classify the change as one or
more of:

- public API or behavior;
- codec correctness or interoperability;
- performance or allocation sensitive;
- concurrency or ownership sensitive;
- internal-only;
- documentation, test, or metadata only.

Use that classification to choose evidence. Do not claim that a change is safe,
compatible, or faster based only on code inspection.

## Preserve compatibility

- Keep exported names, signatures, interfaces, error behavior, defaults, and
  accepted input compatible unless the user explicitly authorizes a breaking
  change.
- Preserve Brotli and gzip/flate wire compatibility. For codec changes, test
  round trips and, where relevant, interoperability with an independent
  implementation or existing corpus.
- Add a regression test for every corrected failure mode. Include boundary,
  empty, truncated, malformed, and zero-value inputs when they are relevant.
- Run `go test ./...` and `go vet ./...`. Run `go mod tidy -diff` when module
  metadata or dependencies change.
- In the final report, state the compatibility impact and the evidence. If a
  compatibility dimension is not applicable, say why.

## Prove performance impact

For any change that can affect runtime work, memory use, compression ratio, or
I/O behavior:

1. Select the smallest relevant existing benchmarks in `brotli_test.go` or
   `flate/flate_test.go`; add a representative benchmark if none covers the
   changed path.
2. Measure the unmodified baseline and the candidate with the same Go version,
   machine, inputs, environment, benchmark regex, and iteration count. Do not
   compare results produced under materially different conditions.
3. Include allocation data and repeated samples, normally with
   `go test -run '^$' -bench '<regex>' -benchmem -count=10`.
4. Compare statistically with `benchstat` when available. Otherwise report the
   raw before/after distributions and clearly label the limitation.
5. Report `ns/op`, `B/op`, and `allocs/op`, plus throughput and compressed size
   or ratio when the change can affect them. Treat regressions as blockers unless
   they are understood, quantified, and explicitly accepted for a stated benefit.

For documentation, test-only, or metadata-only changes, record why production
runtime and allocation behavior cannot change; a synthetic benchmark is not
required. Never describe performance as improved or unchanged without evidence.

## Safety and ownership

- Prefer safe Go. New use of `unsafe`, `reflect`-based memory mutation, or
  `//go:nosplit` requires explicit justification, focused tests, and benchmarks.
- Validate lengths, indices, integer conversions, and arithmetic before slicing,
  indexing, allocating, shifting, or advancing codec state. Malformed or
  adversarial input must return a controlled error instead of panicking.
- Treat nil receivers, nil interfaces, nil readers/writers, zero values, empty
  buffers, and typed nils according to the public contract. Add focused tests
  whenever a changed path could dereference a pointer or invoke an interface.
- Do not use `panic` for input validation or recoverable errors. A panic is
  acceptable only for a proven internal invariant that cannot be triggered by
  public or malformed input; document and test that boundary.
- Make ownership of slices, byte buffers, maps, pooled objects, readers, and
  writers unambiguous. Document whether inputs are borrowed, retained, copied,
  mutated, or returned to a pool. Do not retain caller-owned mutable storage
  beyond the documented lifetime or expose pooled/internal storage after reuse.
- Avoid hidden aliasing between input, output, scratch, and pooled buffers. Clear
  references before pooling when retention or cross-request data exposure is
  possible.
- Do not introduce shared mutable state without a defined synchronization and
  ownership model. Keep lock scope and object lifecycle explicit; never copy
  synchronization primitives after first use.
- Run `go test -race ./...` for changes involving state, reuse, pools, streaming,
  callbacks, globals, or concurrency. Add a concurrent regression test when the
  race detector otherwise cannot exercise the changed ownership boundary.

## Required handoff

Summarize the impact classification, compatibility evidence, safety/ownership
analysis, tests and race-detector result, and before/after benchmark evidence.
Call out anything not run and the exact reason. Do not mark work complete while a
relevant safety, compatibility, race, or performance check is failing.
