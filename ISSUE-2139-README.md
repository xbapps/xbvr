# Fix for xbapps/xbvr#2139 — "database is locked" kills run-selected-scrapers

Origin issue: https://github.com/xbapps/xbvr/issues/2139

## What was wrong

SQLite was opened with zero pragmas (`pkg/models/db.go`), so every handle
ran with `journal_mode=delete` and the mattn driver's default 5s
`busy_timeout`. Selecting all scrapers fans out dozens of concurrent
scraper goroutines (`ConcurrentScrapers` defaults to 9999), each ending in
`site.Save()`, alongside the `sceneDBWriter` and UI polling reads. In
delete mode every read holds a SHARED lock that blocks the writer, and
writer queues under that load routinely exceed 5s — so `SaveWithRetry`'s
10 attempts x ~100ms all fail with `database is locked` and its
`log.Fatal` kills the whole server mid-scrape. `--concurrent_scrapers 1`
doesn't help because the flag only batches scraper goroutines; it never
serialized DB writes.

## What this branch changes

- `pkg/models/db.go`: new `sqliteWriteDSN()` appends
  `_busy_timeout=30000&_journal_mode=WAL&_synchronous=NORMAL` to the DSN for
  the sqlite3 driver (MySQL untouched), used by both `GetDB()` and
  `GetCommonDB()`. DSN params apply to every pooled connection, unlike
  post-open PRAGMAs. WAL stops readers blocking the writer; the longer
  timeout rides out writer queues.
- `SaveWithRetry`: retry policy goes from 10 x ~100ms to 30 attempts with
  1s→15s backoff. The `log.Fatal` backstop is kept deliberately: every
  call site ignores the error return, so returning it would turn loud
  death into silent skipped saves. With the pragmas, transient locks no
  longer reach it — it now only fires on persistent failures.
- New `pkg/models/db_sqlite_test.go`: DSN unit test, pragma assertions
  (WAL + 30s timeout on a real handle), and a lock-contention regression
  test holding a write lock 8s while a second handle writes (fails at ~5s
  on the pre-fix DSN with the exact reported error, waits and succeeds
  with the fix).

## Why I think it's fixed

The failure class is removed at the SQLite level: readers can no longer
stall the writer (WAL), and writer queues get 30s per statement plus
backed-off retries instead of 5s + 10 fast attempts. Verified the new
lock test fails on the plain DSN (`database is locked` at 5.5s) and the
full `pkg/models` suite passes with the fix.

## How to test

1. `go test` doesn't run on pristine base (#2252, flag.Parse in init), so
   compile and run the test binary directly with a sandboxed app dir:
   `go test -c -o /tmp/models.test ./pkg/models/`,
   then `mkdir -p /tmp/xbvr-test-sandbox &&
   XBVR_APPDIR=/tmp/xbvr-test-sandbox /tmp/models.test` — PASS.
2. Manual: Options > Scrapers > select all > Run selected scrapers —
   completes with no `level=fatal msg="Failed to save"`.
3. `sqlite3 <appdir>/main.db "PRAGMA journal_mode;"` now reports `wal`
   (set on first open after upgrade; fully backward compatible).

No pushes or PRs from this branch without explicit approval.
