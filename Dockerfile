# syntax=docker/dockerfile:1.7
FROM golang:1.22-bookworm AS build

# Bikin apt lebih stabil: noninteractive + retry + cache
ENV DEBIAN_FRONTEND=noninteractive
RUN --mount=type=cache,target=/var/cache/apt \
    --mount=type=cache,target=/var/lib/apt \
    bash -lc 'set -euo pipefail; \
      for i in {1..4}; do \
        apt-get update && \
        apt-get install -y --no-install-recommends build-essential pkg-config libwebp-dev && \
        break || { echo "apt failed, retry $i"; sleep 10; }; \
      done && \
      rm -rf /var/lib/apt/lists/*'

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ENV CGO_ENABLED=1 GOFLAGS="-ldflags=-s -w"
# kalau main.go di root:
RUN go build -o /app/server ./main.go

# ---------- runtime ----------
FROM debian:bookworm-slim
ENV DEBIAN_FRONTEND=noninteractive
RUN --mount=type=cache,target=/var/cache/apt \
    --mount=type=cache,target=/var/lib/apt \
    bash -lc 'set -euo pipefail; \
      for i in {1..4}; do \
        apt-get update && \
        apt-get install -y --no-install-recommends libwebp && \
        break || { echo "apt failed, retry $i"; sleep 10; }; \
      done && \
      rm -rf /var/lib/apt/lists/*'

WORKDIR /app
COPY --from=build /app/server /app/server

# sesuaikan dengan port app kamu
EXPOSE 8080
CMD ["/app/server"]

