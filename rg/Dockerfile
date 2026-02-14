FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /webshop ./cmd/webshop/

# --- Runtime stage ---
FROM alpine:3.21

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /webshop .
COPY configs/ ./configs/

EXPOSE 8080

# TODO: Add health check
# HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://localhost:8080/health || exit 1

CMD ["./webshop"]
