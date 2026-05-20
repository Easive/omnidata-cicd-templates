# syntax=docker/dockerfile:1

# ─── Builder stage ───────────────────────────────────────────────────────────
ARG GO_VERSION=1.22
FROM golang:${GO_VERSION}-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Download dependencies first (better layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
# CGO_ENABLED=0 produces a fully static binary suitable for distroless
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-w -s -extldflags=-static" \
    -trimpath \
    -o /app/server \
    ./cmd/server

# ─── Runner stage ─────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot AS runner

# Copy timezone data and CA certificates from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the compiled binary
COPY --from=builder /app/server /app/server

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/server"]
