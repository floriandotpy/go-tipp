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
| `just migrate` | Run migrations without starting the server |
| `just build` | Compile binaries to `bin/` |

## Fetching Results Automatically

The CLI tool fetches match results from [openligadb.de](https://www.openligadb.de/) and updates scores in the database.

For production, set up a cron job:

```
*/2 17-23 * * * export DATABASE_URL_GO="user:pass@tcp(host:3306)/gotipp?parseTime=true"; cd /path/to/go-tipp; go run ./cmd/cli -dsn=$DATABASE_URL_GO
```

(Runs every two minutes between 17:00–23:59)

## Resources

- Favicon source (licensed under CC-BY 4.0): https://favicon.io/emoji-favicons/soccer-ball

## License

All code that is not included from a third party is licensed under the MIT License.
