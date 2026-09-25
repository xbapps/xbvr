# Fix for xbapps/xbvr#2279 — TransVR date panic kills the whole server

Origin issue: https://github.com/xbapps/xbvr/issues/2279

## What was wrong

Running all scrapers, the server died with SIGSEGV inside goment
(`parse.go:403`, called from `pkg/scrape/transvr.go:64`). Two layers:

1. **Trigger — a data race in goment, not a bad string.** `goment.New`
   fills its global parse/format tables lazily with no mutex or `sync.Once`.
   4,000+ malformed inputs parse without panic, but 64 goroutines calling
   `goment.New` concurrently trip the race detector (`-race` reproducer in
   /tmp, since retired): one reader sees a half-written table entry and
   calls a nil func value. Single-scraper runs survive because the tables
   are fully built before concurrency starts; ~150 scraper goroutines at
   once collide in the fill window. (TransVR/GroobyVR also fed page text
   straight in with the error ignored — same file, fixed alongside.)
2. **Blast radius.** The per-scraper goroutines in `runScrapers`
   (`pkg/tasks/content.go`) had no `recover`, so any scraper panic —
   this one or the next — takes down the entire server mid-run.

## What this branch changes

- New `pkg/scrape/date_parse.go`: package `init` warms both goment tables
  once, single-threaded, before any goroutine runs (after that they are
  read-only, so concurrent use is safe); plus `parseGroobyDate()`, which
  only forwards `Month D, YYYY`-shaped strings to goment and returns ""
  otherwise — a dateless scene beats a poisoned one.
- `pkg/scrape/transvr.go`, `pkg/scrape/groobyvr.go`: date parsing goes
  through the guarded helper (identical code, identical latent crash).
- `pkg/tasks/content.go`: new `runScraperSafe()` wraps both scraper spawn
  sites, converting a panic into a logged skip (`scraper X panicked and
  was skipped`). `wg.Done` stays deferred inside each Scrape, which runs
  during unwind, so batch accounting is unaffected.
- Tests: `pkg/scrape/date_parse_test.go` (valid dates format, garbage —
  including the 2-char crash shape — yields ""), and
  `pkg/tasks/scrape_safe_test.go` (panicking fake scraper is contained,
  counter returns to zero, no DB touched).

## Why I think it's fixed

The `-race` reproducer fails on the library as vendored and the warmed
tables remove the race class process-wide, not just for these two
scrapers; the 30+ other unchecked `goment.New` call sites are covered by
the warmup plus the `recover` backstop. Both suites green (see below).

## How to test

1. `go test -vet=off -c -o /tmp/s.test ./pkg/scrape/ && XBVR_APPDIR=/tmp/sbx /tmp/s.test` — PASS
   (`-vet=off`: 3 pre-existing non-constant-format vet failures on
   pristine, #2252; run the binary directly because `go test` flags trip
   `flag.Parse` in init on pristine for the same reason).
2. Same for `./pkg/tasks/` — PASS, log shows the panic-test skip line.
3. Manual: run all scrapers; a panicking scraper now logs and continues.

No pushes or PRs from this branch without explicit approval.
