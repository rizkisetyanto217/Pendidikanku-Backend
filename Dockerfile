FROM golang:1.22-bookworm AS build
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        build-essential pkg-config libwebp-dev && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=1
RUN go build -o server ./main.go

FROM debian:bookworm-slim
RUN apt-get update && \
    apt-get install -y --no-install-recommends libwebp && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=build /app/server /app/server
EXPOSE 8080
CMD ["/app/server"]

