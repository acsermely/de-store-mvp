# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o storage-node cmd/storage-node/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/storage-node .
COPY --from=builder /app/migrations ./migrations

# Create data directory
RUN mkdir -p /data

# Expose ports
EXPOSE 8090

# Run the storage node
ENTRYPOINT ["./storage-node"]
CMD ["start"]