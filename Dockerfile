# ==========================
# Build Stage
# ==========================
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary
RUN go build -o main cmd/server/main.go

# ==========================
# Production Stage
# ==========================
FROM alpine:3.20

WORKDIR /app

# Copy binary
COPY --from=builder /app/main .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

#Expose port
EXPOSE 4000

#Run app
CMD ["./main"]
