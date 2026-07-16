# xbvr Recommendation Engine — Design

Native (in-fork) recommender with two outputs:
- **For You** — owned, available scenes you're most likely to want to watch, ranked.
- **Cleanup** — owned, available scenes you're least likely to miss, ranked (list-only; no auto-delete).

Both surface as **deo-enabled smart playlists** so HereSphere/DeoVR list them automatically, sorted by a true score (not a coarse tier).

## How it surfaces (the xbvr constraint)

HereSphere/DeoVR lists = playlists with `is_deo_enabled = true`; each runs its `search_params`
(`RequestSceneList`) live and returns scenes in the sort order baked into those params
(`heresphere.go:992`, `deovr.go:627`). Playlists are filter-only (no explicit membership table),
and sort is limited to built-in fields.

So we add **two persisted score columns** on `scenes` and make them sortable/filterable:
- `rec_watch_score REAL` and `rec_delete_score REAL` (+ `rec_scored_at`).
- New sort keys `rec_watch_desc` / `rec_delete_desc` (`model_scene.go` sort switch).
- New filter attributes `Recommended To Watch` / `Recommend To Delete`
  → `rec_watch_score > 0` / `rec_delete_score > 0` (`model_scene.go` attribute switch).
- Two seeded system playlists ("For You", "Cleanup") that filter the attribute and sort by score,
  `is_deo_enabled = true`, `limit` from config.

A background task (cron + on-demand API) recomputes the scores and writes them back. The lists then
reflect the latest scores with zero per-request cost.

## Signals available (from this DB)

- Explicit (sparse, high value): `favourite` (391), `star_rating` (49), `watchlist` (132), `wishlist`.
- Implicit (dense): `histories` (190k rows / 4.9k scenes), `total_watch_time`, `is_watched`,
  `last_opened`. This is the primary fuel.
- Content: `scene_cast` (actors), `scene_tags` (tags), `site`/`studio`, `release_date`/`added_date`,
  file resolution/FOV/duration, `is_scripted`.

## Algorithm

### 1. Per-scene affinity (how much you liked a scene you've engaged with)

```
affinity(s) = clamp(
    favourite_bonus(s)            // +1.0 if favourite
  + rating_term(s)                // (star-3)/2 in [-1,+1] when rated, else 0
  + engagement_term(s)            // implicit watch signal, see below
  , -1, +1)

engagement_term(s):
    completion = min(1, total_watch_time / max(duration, 1))
    sessions   = history session count for s
    if sessions == 0:            return 0
    if completion < 0.10 and sessions == 1:  return -0.3   // sampled & abandoned
    base = 0.4*completion + 0.3*min(1, (sessions-1))       // rewatch is a strong like
    return min(0.8, base)
```

Only scenes with affinity != 0 contribute to the taste profile (positives pull toward, mild
negatives push away).

### 2. Taste profile (weighted, IDF-damped frequency over content features)

For each feature dimension, weight = Σ affinity(s) over scenes that have that feature, then damp by
global popularity so ubiquitous tags ("blowjob", "pov") don't dominate:

```
actorWeight[a]  = Σ_s∈scenes(a) affinity(s)
tagWeight[t]    = ( Σ_s∈scenes(t) affinity(s) ) / log(1 + globalTagCount[t])   // TF-IDF-ish
siteWeight[site]= Σ_s∈scenes(site) affinity(s)
```

Each map is then normalized to [-1, 1] by its max abs value.

### 3. Watch score (candidates: available, not recently watched)

```
rec_watch_score(s) =  Wactor * mean(actorWeight over cast(s))
                    + Wtag   * sum(tagWeight over tags(s)) / sqrt(#tags(s))
                    + Wsite  * siteWeight[site(s)]
                    + Wqual  * qualityBoost(s)        // higher resolution, scripted-if-you-like
                    + Wfresh * freshnessBoost(s)      // newer release/added, decays
                    - diversityPenalty(s)             // cap repeats of same top actor in the list
candidates excluded: not available, recently watched (configurable), favourite (already known).
Keep top N (config, default 150); everything else → 0.
```

### 4. Delete score (candidates: available scenes)

Hard protections force score = 0 (never suggest deleting): `favourite`, `watchlist`, `wishlist`,
`star_rating >= protectRating` (default 4), added within `graceDays` (default 30), or present in the
current watch top-set.

```
unloved(s) =  watchedButCold(s)   // watched, low completion, old last_opened, low/no rating
            + neverWatchedOld(s)  // never watched & added long ago
            + lowAffinity(s)      // cast/tags/site you don't favour (negative taste match)
            + lowRating(s)        // star 1-2 → strong
rec_delete_score(s) = unloved(s) * (1 + Wsize * sizeNorm(s))   // bigger files rank a bit higher
Keep top M (config, default 200); everything else → 0.
```

`sizeNorm` = file size scaled to [0,1] across the collection. Size is a *booster/tiebreaker*, not the
primary driver (you asked to surface "stuff you won't miss", with reclaim value as a secondary nudge).

### Mutual exclusion
A scene in the watch top-set is protected from the delete list. Otherwise the two lists rarely
overlap (high-affinity-unwatched vs low-affinity).

## Visual quality (no-reference, file-level)

PSNR/SSIM/VMAF need a pristine reference we don't have, so quality is a *no-reference*
sharpness measure: sample one frame every 5 min (from 5 min, capped at `VQMaxSamples`),
run a Laplacian (edge) convolution at **native resolution** and read the mean edge
energy (`signalstats.YAVG`); the per-file score is the median across samples. Works with
ffmpeg 4.x (no `blurdetect`). Stored on `files`: `visual_quality`,
`visual_quality_samples`, `visual_quality_computed_at`. Measured lazily and cached —
only for the video files of scenes that land in the two lists (expensive: ~10-13s/frame
at 8K). `vqConcurrency` files measured in parallel via xbvr's own ffmpeg.

Two-pass re-rank: after the watch/delete sets are chosen, measure their files, take each
scene's best file quality, cohort-normalize to a [0,1] percentile, then nudge the scores
(`* (1 ± WVisualQuality*(relQ-0.5))`) so crisp scenes rise in For You and soft scenes rise
in Cleanup. Membership of the lists is unchanged; only the order within each is adjusted.

## Config (persisted in KV `config`, editable in UI)

`Config.Recommendation`: enabled, weights (Wactor/Wtag/Wsite/Wqual/Wfresh/Wsize), watchListSize,
deleteListSize, protectRating, graceDays, excludeRecentlyWatched. Plus a `Cron.RecommendationSchedule`.

## Files (fork changes)

Backend:
- `pkg/models/model_scene.go` — add 3 fields; add sort cases; add attribute cases.
- `pkg/api/scenes.go` — add the two attributes to the filter list.
- `pkg/config/config.go` — add `Recommendation` struct + `Cron.RecommendationSchedule`.
- `pkg/migrations/migrations.go` — migration: add columns + seed "For You"/"Cleanup" playlists.
- `pkg/recommend/engine.go` — the scoring engine (this package).
- `pkg/tasks/recommend.go` — `GenerateRecommendations()` task wrapper.
- `pkg/server/cron.go` — schedule it.
- `pkg/api/recommendations.go` — POST recompute, GET/POST config; register in `server.go`.

UI:
- `ui/src/store/optionsRecommendations.js`, `ui/src/views/options/sections/Recommendations.vue`,
  wire into `Options.vue` + `store/index.js`.

Build/deploy:
- `Dockerfile.build` — multi-stage (node → golang+CGO → ubuntu) producing a runnable image.
```
