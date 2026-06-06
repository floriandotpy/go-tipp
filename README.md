## go-tipp

A self-hosted sport betting game for me and my friends, written in Go.

## Prerequisites

- [Go](https://go.dev/dl/) (1.22+)
- [Docker](https://www.docker.com/products/docker-desktop/)
- [just](https://github.com/casey/just) (`brew install just`)
- [dbmate](https://github.com/amacneil/dbmate) (`brew install dbmate`)

## Local Development

```sh
just dev
```

This will:
1. Start a MySQL 8.0 container (or reuse an existing one)
2. Run database migrations
3. Start the web server at http://localhost:8090

### Other Commands

| Command | Description |
|---------|-------------|
| `just dev` | Start everything (DB + migrations + server) |
| `just db-shell` | Open a MySQL prompt |
| `just db-down` | Stop the MySQL container |
| `just db-destroy` | Remove container and all data |
| `just fetch-results` | Fetch live match results from API |
| `just sync-phases` | Sync event phases and matches from API |
| `just migrate` | Run migrations without starting the server |
| `just build` | Compile binaries to `bin/` |

## Deployment (Coolify)

The app ships with a Dockerfile. Point Coolify at the repo and it will build and deploy automatically.

**Required environment variables:**

| Variable | Purpose | Example |
|----------|---------|---------|
| `DATABASE_URL` | Used by dbmate for migrations | `mysql://gotipp:pass@mysql:3306/gotipp` |
| `DATABASE_URL_GO` | Used by the Go app | `gotipp:pass@tcp(mysql:3306)/gotipp?parseTime=true` |

**Coolify settings:**
- Build: Dockerfile
- Port: `8090`
- Health check path: `/health`

TLS is handled by Coolify's Traefik proxy — the app runs plain HTTP inside the container. Migrations run automatically on each deploy via the entrypoint.

## Scheduled Commands

Two CLI commands handle automatic data syncing from [openligadb.de](https://www.openligadb.de/). Both accept a `-dsn` flag or read from the `DATABASE_URL_GO` environment variable.

### fetch-results — Live Score Updates

Fetches match results and goals for matches currently in progress. Designed to run frequently (e.g., every minute) during match days.

**Behavior:**
1. Checks for an active event — exits if none
2. Checks if any match has started but isn't finished — exits immediately (no API call) if not
3. Fetches data from the event API, syncs goals and results for live matches
4. Recomputes user points when a match is newly marked as finished

```sh
# Local
just fetch-results

# Production (cron)
go run ./cmd/fetch-results
# or with compiled binary:
./bin/fetch-results
```

### sync-phases — Phase & Match Import

Syncs event phases and matches from the API. Creates new phases as they appear (e.g., knockout rounds added after group stage), and updates match details (team names, kick-off times). Run 1–2 times per day.

**Behavior:**
1. Checks for an active event — exits if none
2. Fetches all match data from the event API
3. Groups matches by phase, upserts phases and matches by `api_match_id`

```sh
# Local
just sync-phases

# Production (cron)
go run ./cmd/sync-phases
# or with compiled binary:
./bin/sync-phases
```

### Example Cron Setup (Coolify)

```
# Fetch results every minute during typical match hours
* 14-23 * * * ./bin/fetch-results

# Sync phases twice a day
0 6,18 * * * ./bin/sync-phases
```

## Resources

- Favicon source (licensed under CC-BY 4.0): https://favicon.io/emoji-favicons/soccer-ball
- Icons in main navigation are Phosphoricons in duotone mode, pick matching icons from here: https://phosphoricons.com/?weight=duotone&q=ball&color=5b7e3c

## License

All code that is not included from a third party is licensed under the MIT License.
