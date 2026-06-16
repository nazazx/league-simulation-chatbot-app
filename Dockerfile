# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

ARG GOPROXY=direct
ENV GOPROXY=${GOPROXY}

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server/main.go

# Run stage
FROM alpine:3.19

WORKDIR /app

# Copy binary from builder
COPY --from=builder /server .

# Copy frontend files
COPY frontend/ ./frontend/

EXPOSE 8080

CMD ["./server"]
