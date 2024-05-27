## go-tipp

A self-hosted sport betting game for me an my friends, written in Go.

# Requirements and Setup

1. Install Go
2. Install MySQL
3. Create user and database (suggested: db name `gotipp` and user name `gotipp`).
4. Run initial SQL statements (see `cmd/web/createtables.sql`)

# Run

Note: Replace db name, user name and password with your own names in the following statement:

```sh
go run ./cmd/web -addr=":8091" -dsn="gotipp:password@/gotipp?parseTime=true"
```
