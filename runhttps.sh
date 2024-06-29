#!/bin/zsh
exec go run ./cmd/web -addr=":8090" -dsn=$DATABASE_URL_GO -https

