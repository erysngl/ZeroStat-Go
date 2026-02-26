# Build Stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Enable CGO locally if host metrics compilation requires it, but standard 
# gopsutil often compiles fine without it on linux setups
ENV CGO_ENABLED=0 GOOS=linux

# Install dependencies needed occasionally by OS-level data fetching
RUN apk add --no-cache gcc musl-dev

# Copy all source files first so go mod tidy can inspect imports
COPY . .

# Generate go.sum and tidy go.mod
RUN go mod tidy

# Download dependencies
RUN go mod download

# Build statically linked binary emphasizing size reduction
RUN go build -ldflags="-s -w" -o zerostat ./cmd/zerostat

# Final Minimal Stage
FROM alpine:latest

WORKDIR /app

# Add timezone data and ca-certificates for network stuff if ever needed
RUN apk add --no-cache tzdata ca-certificates

# Copy from builder
COPY --from=builder /app/zerostat .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/locales ./locales

# Ensure executable permissions
RUN chmod +x /app/zerostat

# Provide a safe default port map hint
EXPOSE 9124

# Command array passing
CMD ["./zerostat"]
