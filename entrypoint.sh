#!/bin/sh
set -e

# Run database migrations before starting the server
if [ -n "$DATABASE_URL" ]; then
    echo "Running database migrations..."
    dbmate --no-dump-schema up
    echo "Migrations complete."
else
    echo "WARNING: DATABASE_URL not set, skipping migrations"
fi

# Start the server
exec ./server
