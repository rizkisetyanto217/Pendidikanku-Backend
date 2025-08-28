# syntax=docker/dockerfile:1
FROM golang:1.22-bookworm AS build

ENV DEBIAN_FRONTEND=noninteractive

# Retry apt biar nggak gampang timeout
RUN bash -lc 'set -euo pipefail; \
  for i in {1..4}; do \
    apt-get update && \
    apt-get install -y --no-install-recommends build-essential pkg-config libwebp-dev && \
    exit 0 || { echo "apt failed, retry $i"; sleep 10; }; \
  done; exit 1'

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# CGO wajib ON untuk chai2010/webp
ENV CGO_ENABLED=1 GOFLAGS="-ldflags=-s -w"

# kalau main.go di root:
RUN go build -o /app/server ./main.go
# atau lebih umum (auto pick package main):
# RUN go build -o /app/server .

# ---------- runtime ----------
FROM debian:bookworm-slim
ENV DEBIAN_FRONTEND=noninteractive

RUN bash -lc 'set -euo pipefail; \
  for i in {1..4}; do \
    apt-get update && \
    apt-get install -y --no-install-recommends libwebp && \
    exit 0 || { echo "apt failed, retry $i"; sleep 10; }; \
  done; exit 1'

WORKDIR /app
COPY --from=build /app/server /app/server

# Sesuaikan port dengan aplikasi kamu (umumnya 8080)
EXPOSE 8080
CMD ["/app/server"]

