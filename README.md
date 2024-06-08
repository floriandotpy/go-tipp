## go-tipp

A self-hosted sport betting game for me an my friends, written in Go.

# Requirements and Setup

1. Install Go
2. Install MySQL
3. Create user and database (suggested: db name `gotipp` and user name `gotipp`).
4. Run database setup (see below)
5. Setup a local certificate for https (see below)

## Database setup

1. Add the database connection string to your environment variables

```
export DATABASE_URL="mysql://DB_USER:DB_PASSWORD/DB_NAME?parseTime=true"
```

Note: Replace DB_USER, DB_PASSWORD and DB_NAME with the values for your system.

2. Install dbmate for database migrations: https://github.com/amacneil/dbmate
3. (optional) Run `dbmate create` to create a new database (if you haven't done that manually)
4. Run `dbmate up` to run the migrations which create the schema and insert initial data

## TLS setup

For local development, create a self-signed certificate.

```
mkdir tls && cd tls
go run /usr/local//Cellar/go/1.22.3/libexec/src/crypto/tls/generate_cert.go --rsa-bits=2048 --host=localhost
```

Note: The path to `generate_cert.go` may be different on your system, but it should come included with your Go installation.

# Run

Note: Replace db name, user name and password with your own names in the following statement:

```sh
go run ./cmd/web -addr=":8090" -dsn="gotipp:password@/gotipp?parseTime=true"
```

In your browser, open:

```
https://localhost:8090/
```
