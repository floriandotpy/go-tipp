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
export DATABASE_URL="mysql://DB_USER:DB_PASSWORD@HOST:PORT/DB_NAME"
export DATABASE_URL_GO="DB_USER:DB_PASSWORD/DB_NAME?parseTime=true"
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

Run with https enabled (using local self-signed certificate)

```sh
sh runhttps.sh
```

Run with http disabled (for example in production: using the https of a production server, behind a reverse proxy)

```sh
sh run.sh
```

In your browser, open:

```
https://localhost:8090/
```

# Update scores automatically using API

Run this script to fetch results from

Set up cronjob (configured to run on [Uberspace](https://manual.uberspace.de/daemons-cron/)):

```
MAILTO=""
# disable emails of crontab output (errors will still be mailed)
*/2 17-23 * * * export DATABASE_URL_GO="<syntax see above>"; cd /path/to/go-tipp; /bin/bash fetch_results.sh
```

(Runs every two minutes between 17:00 to 23:59)

# Resources

- Favicon source (licensed under CC-BY 4.0): https://favicon.io/emoji-favicons/soccer-ball

# License

All code that is not included from a third party is licensed under the MIT License.
