# Full app in one image: the Go backend serves the API, Swagger and the built
# React admin UI on the same origin (required by the session cookie).
#
#   docker build -t claimconfig .
#   docker run -p 8080:8080 -v claimconfig-data:/app/data claimconfig
#   open http://localhost:8080   (admin@papaya.com / 123456)
#
# Build context: this directory ("AI Code test"), see .dockerignore.

# 1. Frontend: Vite build -> /web
FROM node:24-alpine AS web
WORKDIR /src
COPY workspace/frontend/package.json workspace/frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY workspace/frontend/ ./
RUN npm run build

# 2. Backend: static Go binary (pure-Go SQLite, no CGO)
FROM golang:1.27-alpine AS api
WORKDIR /src
COPY workspace/backend/go.mod ./
COPY workspace/backend/ ./
RUN go mod tidy && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/claimconfig .

# 3. Runtime
FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -u 10001 app \
 && mkdir -p /app/data && chown app /app/data
WORKDIR /app
COPY --from=api /out/claimconfig /app/claimconfig
COPY --from=web /src/dist /app/web
USER app

# Defaults; override with -e. DATABASE_URL may also be a Turso libsql:// URL.
ENV PORT=8080 \
    STATIC_DIR=/app/web \
    DATABASE_URL="file:/app/data/app.db?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"

# SQLite file lives here: mount a volume so data survives container restarts.
VOLUME ["/app/data"]
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:${PORT}/api/v1/health >/dev/null || exit 1
ENTRYPOINT ["/app/claimconfig"]
CMD ["serve"]
