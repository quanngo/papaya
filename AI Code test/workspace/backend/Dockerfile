# Backend image. Build from workspace/backend:
#   docker build -t claimconfig .
#   docker run -p 8080:8080 -e DATABASE_URL="libsql://<db>.turso.io?authToken=..." claimconfig
# Without DATABASE_URL it uses a SQLite file inside the container (lost on restart).
# To also serve the frontend, copy its dist/ to /app/web and set server.static-dir
# (a combined image will be added with the frontend).
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN go mod tidy && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/claimconfig .

FROM alpine:3.22
RUN adduser -D -u 10001 app && mkdir -p /app/data && chown app /app/data
WORKDIR /app
COPY --from=build /out/claimconfig /app/claimconfig
USER app
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/app/claimconfig"]
CMD ["serve"]
