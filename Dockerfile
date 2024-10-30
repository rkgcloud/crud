# Stage 1: Build the Go application
FROM golang:1.20-alpine AS builder

# Set environment variables
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Create an app directory and copy code files
WORKDIR /app
COPY . .

# Download Go modules and build the application
RUN go mod download
RUN go build -o crud cmd/main.go

# Stage 2: Create the final lightweight image
FROM alpine:latest

# Set environment variables
ENV PORT=8080
ENV DATABASE_URL="host=localhost user=postgres password=postgres dbname=testdb port=5432 sslmode=disable"

# Copy the binary from the builder stage
COPY --from=builder /app/crud /crud

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["/crud"]
