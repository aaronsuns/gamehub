# GameHub Architecture

## Request Flow

```
Client ──▶ mainMux
              ├── GET /health  ──────────────────────▶ 200 OK (no rate limit)
              │
              └── /             limiter.Middleware
                                    │
                                    ├── 429 if IP over limit (token bucket)
                                    │
                                    └── apiMux
                                           ├── GET /series/live   ──▶ Atlas GetSeriesAll ──▶ JSON
                                           ├── GET /players/live  ──▶ LiveContext ──▶ Atlas GetPlayersAll ──▶ JSON
                                           └── GET /teams/live    ──▶ LiveContext ──▶ Atlas GetTeamsAll ──▶ JSON
```

## Live Context Flow (players/live, teams/live)

```
GetLiveContext (TTL cache)
    │
    ├── cache hit ──▶ return LiveContext{TeamIDs, PlayerIDs}
    │
    └── cache miss ──▶ loadLiveContext:
                          │
                          ├── Atlas GetSeriesAll(lifecycle=live)
                          │       └── extract roster IDs from participants
                          │
                          ├── Atlas GetRostersAll(id in rosterIDs)
                          │       └── extract team IDs, player IDs
                          │
                          └── return LiveContext ──▶ cache
```

## Inbound Rate Limit (per IP)

```
Request ──▶ getClientIP ──▶ bucket for IP
                                │
                                ├── tokens > 0? ──▶ consume 1, refill over time ──▶ Allow
                                │
                                └── tokens = 0 ──▶ 429 + Retry-After
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

