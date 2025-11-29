# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

RUN go build -o mqtt-wol .

# Final stage
FROM alpine
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/mqtt-wol /usr/local/bin/mqtt-wol

# Ensure binary is executable
RUN chmod +x /usr/local/bin/mqtt-wol

ENTRYPOINT ["mqtt-wol"]
