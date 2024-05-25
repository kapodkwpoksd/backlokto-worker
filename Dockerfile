# Use the official Golang image as the base image
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Install necessary packages for building Go applications
RUN apk add --no-cache git

# Copy the Go modules manifests
COPY go.mod go.sum ./

# Download and cache the Go modules
RUN go mod download

# Copy the source code
COPY . .

# Build the Go application with optimization flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/main ./cmd/main.go

# Use the official PostgreSQL client image to get the pg_dump utility
FROM alpine:latest AS pg_client

# Install the PostgreSQL client and necessary libraries
RUN apk add --no-cache postgresql-client

# Create a new stage from the alpine image
FROM alpine:latest

# Install necessary libraries
RUN apk add --no-cache libpq zstd-libs lz4-libs

# Copy the pg_dump utility from the PostgreSQL client stage
COPY --from=pg_client /usr/bin/pg_dump /usr/bin/pg_dump

# Copy the binary from the builder stage
COPY --from=builder /app/main /app/main

# Set the entry point for the container
ENTRYPOINT ["/app/main"]

# Add pg_dump to the PATH
ENV PATH="/usr/bin:${PATH}"
