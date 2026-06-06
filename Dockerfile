# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Download dependencies first (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Build the binaries
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/web
RUN CGO_ENABLED=0 GOOS=linux go build -o fetch-results ./cmd/fetch-results
RUN CGO_ENABLED=0 GOOS=linux go build -o sync-phases ./cmd/sync-phases

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata curl

# Install dbmate for running migrations on startup
RUN curl -fsSL -o /usr/local/bin/dbmate https://github.com/amacneil/dbmate/releases/download/v2.24.2/dbmate-linux-amd64 \
    && chmod +x /usr/local/bin/dbmate

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/server .
COPY --from=builder /app/fetch-results .
COPY --from=builder /app/sync-phases .

# Copy templates and static assets
COPY ui/ ./ui/

# Copy migrations
COPY db/migrations/ ./db/migrations/

# Copy entrypoint
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

EXPOSE 8090

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8090/health || exit 1

ENTRYPOINT ["./entrypoint.sh"]
