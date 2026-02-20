FROM --platform=linux/amd64 golang:1.24-bookworm

# Install tools for ELF inspection and linking
RUN apt-get update && apt-get install -y --no-install-recommends \
    binutils \
    gcc \
    libc6-dev \
    nasm \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy go module files first for layer caching
COPY go.mod ./
RUN go mod download 2>/dev/null || true

# Copy the rest
COPY . .

# Build
RUN go build -o /usr/local/bin/minicc ./cmd/minicc && \
    go build -o /usr/local/bin/minilink ./cmd/minilink

CMD ["make", "test"]
