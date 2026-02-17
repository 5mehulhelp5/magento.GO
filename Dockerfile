# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app

# Install git for go mod
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o gogento .

# Runtime stage
FROM alpine:3.19
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/gogento .
COPY --from=builder /app/html ./html
COPY --from=builder /app/assets ./assets

EXPOSE 8080

ENTRYPOINT ["./gogento"]
