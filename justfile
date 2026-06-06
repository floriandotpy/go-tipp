# go-tipp development commands

# Database config
db_name := "gotipp"
db_user := "gotipp"
db_pass := "gotipp"
db_port := "3306"
db_container := "gotipp-mysql"

# DSN for Go app
export DATABASE_URL_GO := db_user + ":" + db_pass + "@tcp(127.0.0.1:" + db_port + ")/" + db_name + "?parseTime=true"
# DSN for dbmate
export DATABASE_URL := "mysql://" + db_user + ":" + db_pass + "@127.0.0.1:" + db_port + "/" + db_name

# Start MySQL container + run migrations + start the web server
dev: db-up migrate
    go run ./cmd/web -addr=":8090" -dsn="{{db_user}}:{{db_pass}}@tcp(127.0.0.1:{{db_port}})/{{db_name}}?parseTime=true"

# Start MySQL container + run migrations + start with HTTPS
dev-https: db-up migrate
    go run ./cmd/web -addr=":8090" -dsn="{{db_user}}:{{db_pass}}@tcp(127.0.0.1:{{db_port}})/{{db_name}}?parseTime=true" -https

# Start the MySQL container (idempotent)
db-up:
    #!/usr/bin/env bash
    if docker ps --format '{{{{.Names}}' | grep -q '^{{db_container}}$'; then
        echo "MySQL container already running"
    elif docker ps -a --format '{{{{.Names}}' | grep -q '^{{db_container}}$'; then
        echo "Starting existing MySQL container..."
        docker start {{db_container}}
    else
        echo "Creating new MySQL container..."
        docker run -d \
            --name {{db_container}} \
            -e MYSQL_ROOT_PASSWORD=root \
            -e MYSQL_DATABASE={{db_name}} \
            -e MYSQL_USER={{db_user}} \
            -e MYSQL_PASSWORD={{db_pass}} \
            -p {{db_port}}:3306 \
            mysql:8.0
    fi
    # Wait for MySQL to be fully ready (not just accepting connections, but able to run queries)
    echo "Waiting for MySQL to be ready..."
    for i in $(seq 1 60); do
        if docker exec {{db_container}} mysql -u{{db_user}} -p{{db_pass}} -e "SELECT 1" {{db_name}} &>/dev/null; then
            echo "MySQL is ready!"
            break
        fi
        if [ $i -eq 60 ]; then
            echo "Timed out waiting for MySQL"
            exit 1
        fi
        sleep 1
    done

# Stop the MySQL container
db-down:
    docker stop {{db_container}}

# Remove the MySQL container and its data
db-destroy:
    docker rm -f {{db_container}}

# Run database migrations
migrate:
    dbmate up

# Create a new migration
migration name:
    dbmate new {{name}}

# Open a MySQL shell
db-shell:
    docker exec -it {{db_container}} mysql -u{{db_user}} -p{{db_pass}} {{db_name}}

# Fetch live match results from API (run frequently during matches)
fetch-results:
    go run ./cmd/fetch-results -dsn="{{db_user}}:{{db_pass}}@tcp(127.0.0.1:{{db_port}})/{{db_name}}?parseTime=true"

# Sync event phases and matches from API (run 1-2x per day)
sync-phases:
    go run ./cmd/sync-phases -dsn="{{db_user}}:{{db_pass}}@tcp(127.0.0.1:{{db_port}})/{{db_name}}?parseTime=true"

# Build the project
build:
    go build -o bin/server ./cmd/web
    go build -o bin/fetch-results ./cmd/fetch-results
    go build -o bin/sync-phases ./cmd/sync-phases

# Run all tests
test:
    go test ./... -count=1

# Run tests with verbose output
test-v:
    go test ./... -v -count=1
