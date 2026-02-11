FROM golang:1.24-alpine AS builder
RUN apk add --no-cache build-base
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Validate openapi.json before build (fail fast if spec missing or invalid for go:embed)
RUN go run ./scripts/gen_openapi.go
RUN CGO_ENABLED=1 go build -buildvcs=false -o /server ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates wget
WORKDIR /app
COPY --from=builder /server .
ENV ADDR=:8080
# DATABASE_URL=postgres://... for PostgreSQL; omit and set DB_PATH for SQLite
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -q -O- http://localhost:8080/api/v1/health || exit 1
CMD ["./server"]
