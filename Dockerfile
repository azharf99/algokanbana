# Stage 1: Compile ultra-small static Svelte 5 + Tailwind v4 SPA
FROM node:24-alpine AS web-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

# Stage 2: Compile static Pure-Go binary (CGO_ENABLED=0) with embedded SPA & migrations
FROM golang:1.26-alpine AS go-builder
RUN apk add --no-cache ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /app/web/dist ./web/dist
RUN mkdir -p /out/data/uploads && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/algokanbana ./cmd/server

# Stage 3: Minimal scratch runtime (~16-22 MB total image size, zero OS/shell bloat)
FROM scratch
COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=go-builder /out/data /data
COPY --from=go-builder /out/algokanbana /algokanbana

ENV PORT=8080 \
    APP_MODE=hybrid_safe \
    DB_PATH=/data/algokanbana.db \
    UPLOAD_DIR=/data/uploads \
    SAFE_READONLY_UPSTREAM=true

VOLUME ["/data"]
EXPOSE 8080
ENTRYPOINT ["/algokanbana"]
