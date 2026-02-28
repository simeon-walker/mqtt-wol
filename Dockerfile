# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

ENV CGO_ENABLED=0
RUN go build -ldflags="-s -w" -o mqtt-wol .

# Final stage
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/mqtt-wol /mqtt-wol
ENTRYPOINT ["/mqtt-wol"]
