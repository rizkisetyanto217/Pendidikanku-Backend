# syntax=docker/dockerfile:1.7

### ---------- build ----------
FROM golang:1.24.2-bookworm AS build
ENV DEBIAN_FRONTEND=noninteractive
# izinkan auto-toolchain; aman walau sudah 1.24
ENV GOTOOLCHAIN=auto

RUN apt-get update && \
    apt-get install -y --no-install-recommends build-essential pkg-config libwebp-dev && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# cache deps
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# copy source
COPY . .

# build (CGO ON karena pakai libwebp)
ENV CGO_ENABLED=1
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /app/server ./main.go
# ^ kalau entrypoint kamu di paket lain, ganti pathnya

### ---------- runtime ----------
FROM debian:bookworm-slim
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends libwebp7 && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=build /app/server /app/server

# (opsional) jalankan sebagai nonroot
# RUN useradd -u 10001 -r -s /sbin/nologin appuser
# USER 10001

EXPOSE 8080
CMD ["/app/server"]
