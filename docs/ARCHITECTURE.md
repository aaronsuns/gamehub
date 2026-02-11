# GameHub Architecture

## Request Flow

```
Client ──▶ Gin Router
              ├── GET /health, /monitor, /stats  ──▶ handlers (no rate limit)
              │
              └── API group (rate limited)
                       limiter.Middleware (fixed window per IP)
                            │
                            ├── 429 if over limit (count in window)
                            │
                            └── Handlers (all serve from Live cache when valid)
                                   ├── GET /series/live   ──▶ Live.GetLiveSeries()   ──▶ JSON (cached)
                                   ├── GET /players/live  ──▶ Live.GetLivePlayers()  ──▶ JSON (cached)
                                   └── GET /teams/live    ──▶ Live.GetLiveTeams()    ──▶ JSON (cached)
```

## Live Snapshot Flow (all three /live endpoints)

A single TTL cache holds a **full snapshot**: series JSON, players JSON, teams JSON (and derived IDs). All three endpoints use it; Atlas is only called when the snapshot is loaded or refreshed after TTL.

```
GetLiveSeries() / GetLivePlayers() / GetLiveTeams()
    │
    └── cache.Get()
            │
            ├── cache hit (now < until) ──▶ return snap.Series / snap.Players / snap.Teams  (0 Atlas calls)
            │
            └── cache miss ──▶ loadLiveSnapshot:
                                  │
                                  ├── Atlas GetSeriesAll(lifecycle=live)
                                  │       └── extract roster IDs from participants
                                  │
                                  ├── Atlas GetRostersAll(id in rosterIDs)
                                  │       └── extract team IDs, player IDs
                                  │
                                  ├── Atlas GetPlayersAll(id in playerIDs)
                                  ├── Atlas GetTeamsAll(id in teamIDs)
                                  │
                                  └── cache LiveSnapshot{Series, Players, Teams} ──▶ return requested part
```

## Inbound Rate Limit (per IP, fixed window)

```
Request ──▶ getClientIP ──▶ bucket {count, windowStart}
                                │
                                ├── now - windowStart >= per? ──yes──▶ new window (count=0, windowStart=now)
                                │
                                ├── count++ ; count > limit? ──yes──▶ 429 + Retry-After
                                │
                                └── no ──▶ Allow
```

## Outbound Backoff (Atlas 429)

```
Get() ──▶ waitOutbound ──▶ backoff active? ──yes──▶ sleep until elapsed
              │                    │
              no                   no
              │                    │
              ▼                    ▼
         send request ──▶ 429? ──yes──▶ setBackoff(retryMs), return ErrRateLimited
              │                    │
              no                   │
              │                    │   (next request waits in waitOutbound)
              ▼                    ▼
         return body          propagate 429 to client
```
