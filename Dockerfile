FROM golang:1.24-bookworm AS build

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && \
    apt-get install -y --no-install-recommends build-essential pkg-config libwebp-dev && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ENV CGO_ENABLED=1 GOFLAGS="-ldflags=-s -w"
RUN go build -o /app/server ./main.go

# ---------- runtime ----------
FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && \
    apt-get install -y --no-install-recommends libwebp7 && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=build /app/server /app/server
EXPOSE 8080
CMD ["/app/server"]

