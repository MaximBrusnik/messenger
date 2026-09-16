# ---- Stage 1: Frontend build ----
FROM node:20-alpine AS frontend
WORKDIR /app
COPY frontend/frontend/package.json frontend/frontend/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --prefer-offline
COPY frontend/frontend/ ./
RUN --mount=type=cache,target=/root/.npm npm run build

# ---- Stage 2: Go build ----
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.work ./
COPY services/ ./services/
ARG SERVICE
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd services/${SERVICE} && CGO_ENABLED=0 go build -o /app/service ./cmd/

# ---- Stage 3a: Runtime (default) ----
FROM alpine:3.19 AS runtime
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/service /app/service
ENTRYPOINT ["/app/service"]

# ---- Stage 3b: Gateway runtime (with frontend + uploads) ----
FROM alpine:3.19 AS runtime-gateway
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/service /app/service
COPY --from=frontend /app/dist /app/dist
RUN mkdir -p /app/uploads
ENTRYPOINT ["/app/service"]
