# Issue #2073 — SLR trans scenes 404 after switch to 100% API scraping

Origin: https://github.com/xbapps/xbvr/issues/2073
Branch: `fix/2073-slr-trans` (from `origin/master`)

## What

After the SLR redesign, scraping any `/trans/` scene or trans studio 404s
(`API returned non-200 status for scene 1722 : 404`) and indexes 0 scenes.

## Why

The v3 scene API requires the `project` header matching the scene's project
(1 = main, 3 = trans, 4 = gay). `pkg/scrape/slrstudios.go` picked the header
from `isTransScene`/`isGayScene` (parsed from the `/trans/`/`/gay/` URL
prefix), but the 404 fallback only retried `project=0` when the header had
been `1`. A trans scene URL without the `/trans/` prefix was therefore
fetched with `project=1`, got 404, retried `project=0` (also 404), and was
dropped — the one header that works (`3`) was never tried.

## Fix (`pkg/scrape/slrstudios.go` only)

- `processSceneFromAPI`: on any 404, retry the remaining projects in order
  `1 → 3 → 4 → 0` (skipping the one already tried) and adopt the header
  that returns 200 for the follow-up legacy-endpoint request.
- `fetchScenesFromAPI` (studio listing): same chain on page 1, keeping the
  working header for subsequent pages (previously only `1 → 0`).
- After a successful v3 fetch, the response's authoritative `project` field
  corrects `isTransScene`/`isGayScene`, so a trans scene found via fallback
  still gets the `slr-trans-` `SceneID`, the `/trans/` homepage URL, and
  trans filename variants.

## How to test

1. `go build ./pkg/scrape/` (passes; `go test` not run — broken repo-wide,
   pre-existing).
2. `curl -H "project: 3" https://api.sexlikereal.com/v3/scenes/hot-in-the-kitchen-1722 -w "%{http_code}"`
   → 200 (vs 404 with `project: 1`).
3. UI: Scrapers > TSVirtualLovers > scrape single scene with the issue URL,
   expect the scene indexed; full studio scrape of a trans studio expects
   >0 new scenes.

## Live verification (2026-09-14, curl, desktop User-Agent, JSON only)

- `GET /v3/scenes/hot-in-the-kitchen-1722`: `project: 1` → 404
  (`{"error":"Scene not found"}`), `project: 3` → 200
  (`id:1722, project:3, title:"Hot In The Kitchen"`), `project: 0` → 404,
  `project: 4` → 404. So the old `1 → 0` fallback could never succeed, and
  the new chain (`1,3,4,0`) stops at `3`. API was reachable (not blocking).
- Legacy endpoint `/virtualreality/video/id/1722` → 404 under both
  `project: 3` and `1`, confirming the v3 chain is the effective fix path.
- Sanity: main scene `unwiring-her-desires-68823` with `project: 1` → 200
  (`project:1`), so existing main-site behavior is unchanged.
- No fetched content committed.
