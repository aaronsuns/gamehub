# Architecture Diagram Reference for Figma

This document provides a structured reference for creating an architecture diagram in Figma.

## System Overview

**GameHub** - HTTP API wrapper around Abios Atlas esports data API

## Components

### 1. Client Layer
- **External Clients** (HTTP clients making requests)
  - Browser, curl, load testing tools
  - Multiple concurrent clients

### 2. HTTP Server Layer
- **mainMux** (Main HTTP Router)
  - Routes: `/health`, `/monitor`, `/stats`, `/` (API routes)
  - Metrics middleware (wraps all handlers)
  
- **Rate Limiter Middleware**
  - Token bucket per IP
  - Returns 429 when limit exceeded
  - Eviction of stale buckets

- **apiMux** (API Router)
  - Routes: `/series/live`, `/players/live`, `/teams/live`

### 3. Handler Layer
- **Health Handler** (`/health`)
  - No rate limiting
  - Returns 200 OK
  
- **SeriesLive Handler** (`/series/live`)
  - Direct call to Atlas API
  
- **PlayersLive Handler** (`/players/live`)
  - Uses Live Service
  
- **TeamsLive Handler** (`/teams/live`)
  - Uses Live Service

### 4. Service Layer
- **Live Service**
  - Derives live teams/players from live series
  - Uses TTL cache (10s default)
  - Cache miss triggers: Series → Rosters → Teams/Players

### 5. Client Layer
- **Atlas Client**
  - HTTP client for Atlas API
  - Handles pagination automatically
  - Reactive rate limiting (backoff on 429)
  - Respects Retry-After headers

### 6. External API
- **Atlas API** (Abios)
  - Base URL: `https://atlas.abiosgaming.com/v3`
  - Endpoints: `/series`, `/players`, `/teams`, `/rosters`
  - Rate limited (returns 429 with Retry-After)

### 7. Storage/Cache
- **Live Context Cache**
  - In-memory TTL cache
  - Stores: TeamIDs, PlayerIDs
  - Auto-refreshes on expiry

- **Rate Limiter Buckets**
  - Per-IP token buckets
  - In-memory map
  - Eviction of stale entries

### 8. Observability
- **Metrics Collection**
  - Request counters
  - Rate limit counters (inbound/outbound)
  - Retry-After tracking
  - Time-series history (120 samples)

- **Monitor Dashboard** (`/monitor`)
  - Real-time graphs
  - Request rate visualization
  - Rate limit events

- **Stats Endpoint** (`/stats`)
  - JSON metrics API

## Data Flow Diagrams

### Request Flow: `/series/live`
```
Client → mainMux → Metrics Middleware → Rate Limiter → apiMux → SeriesLive Handler → Atlas Client → Atlas API
                                                                                                    ↓
Client ← JSON Response ←───────────────────────────────────────────────────────────────────────────┘
```

### Request Flow: `/players/live` (Cache Hit)
```
Client → mainMux → Metrics Middleware → Rate Limiter → apiMux → PlayersLive Handler → Live Service → Cache (Hit)
                                                                                                    ↓
Client ← JSON Response ←────────────────────────────────────────────────────────────────────────────┘
```

### Request Flow: `/players/live` (Cache Miss)
```
Client → mainMux → Metrics Middleware → Rate Limiter → apiMux → PlayersLive Handler → Live Service → Cache (Miss)
                                                                                                    ↓
                                                                                    Load Live Context:
                                                                                    1. Atlas Client → GetSeriesAll (lifecycle=live)
                                                                                    2. Extract Roster IDs
                                                                                    3. Atlas Client → GetRostersAll (filter by roster IDs)
                                                                                    4. Extract Team/Player IDs
                                                                                    5. Cache LiveContext
                                                                                                    ↓
                                                                                    Atlas Client → GetPlayersAll (filter by player IDs)
                                                                                                    ↓
Client ← JSON Response ←────────────────────────────────────────────────────────────────────────────┘
```

### Rate Limiting Flow
```
Request → getClientIP → Get Bucket for IP
                              ↓
                    Token Available?
                    ├── Yes → Consume Token → Allow Request
                    └── No → Return 429 + Retry-After
```

### Outbound Rate Limiting Flow
```
Atlas Client.Get() → waitOutbound (check backoff)
                              ↓
                    Backoff Active?
                    ├── Yes → Sleep until elapsed
                    └── No → Send Request
                              ↓
                    Response Status?
                    ├── 429 → Parse Retry-After → Set Backoff → Return ErrRateLimited
                    └── 200 → Return Body
```

## Component Relationships

### Dependencies
- **Handlers** depend on:
  - Atlas Client
  - Live Service
  
- **Live Service** depends on:
  - Atlas Client
  - Cache (internal)
  
- **Atlas Client** depends on:
  - HTTP Client
  - Config (timeouts, page size)

- **Rate Limiter** depends on:
  - Config (rate limit values)
  - Metrics (for tracking 429s)

### Communication Patterns
- **Synchronous**: All HTTP handlers are synchronous
- **Caching**: TTL-based cache reduces external API calls
- **Rate Limiting**: Token bucket algorithm with refill over time
- **Error Propagation**: Errors bubble up from Atlas Client → Handlers → HTTP response

## Key Design Decisions

1. **Caching Strategy**: TTL-based cache for live context (10s) to reduce Atlas API calls
2. **Rate Limiting**: Token bucket per IP for inbound, reactive backoff for outbound
3. **Pagination**: Automatic page fetching in Atlas Client
4. **Error Handling**: Propagate Atlas 429s to clients with Retry-After headers
5. **Observability**: Built-in metrics dashboard for monitoring

## Visual Elements for Figma

### Colors
- **Client Layer**: Light blue
- **Server Layer**: Light green
- **Service Layer**: Light yellow
- **Client/API Layer**: Light orange
- **External API**: Light red
- **Cache/Storage**: Light purple
- **Observability**: Light gray

### Shapes
- **Rectangles**: Components/Services
- **Cylinders**: Storage/Cache
- **Clouds**: External APIs
- **Arrows**: Data flow (label with request type)
- **Diamonds**: Decision points (rate limit checks)

### Layout Suggestions
- **Top**: External Clients
- **Middle Left**: HTTP Server Components (stacked)
- **Middle Right**: Service Layer
- **Bottom Left**: Atlas Client
- **Bottom Right**: Atlas API
- **Side Panel**: Observability components

## Example Diagram Structure

```
┌─────────────────────────────────────────────────────────┐
│                    External Clients                      │
│              (Browser, curl, loadtest)                   │
└────────────────────┬────────────────────────────────────┘
                     │ HTTP Requests
                     ▼
┌─────────────────────────────────────────────────────────┐
│                    HTTP Server Layer                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │  mainMux     │  │   Metrics    │  │ Rate Limiter │  │
│  │  (Router)    │→ │  Middleware  │→ │  Middleware  │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
│         │                                    │          │
│         └────────────────┬───────────────────┘          │
│                          ▼                               │
│                  ┌──────────────┐                        │
│                  │   apiMux     │                        │
│                  │  (API Router)│                        │
│                  └──────────────┘                        │
└────────────────────┬────────────────────────────────────┘
                     │
         ┌───────────┼───────────┐
         ▼           ▼           ▼
┌─────────────┐ ┌──────────┐ ┌──────────┐
│   Handlers  │ │   Live   │ │  Atlas   │
│             │ │  Service │ │  Client  │
└─────────────┘ └────┬─────┘ └────┬─────┘
                     │            │
                     │            │
         ┌───────────┴────────────┴──────┐
         │      Live Context Cache       │
         │    (TTL: 10s, Team/Player IDs)│
         └───────────────────────────────┘
                     │
                     ▼
         ┌───────────────────────┐
         │     Atlas API         │
         │  (Abios Gaming)       │
         └───────────────────────┘
```

Use this structure as a starting point for your Figma diagram!
