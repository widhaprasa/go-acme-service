# Base image for building the Go application
FROM golang:1.23.2-alpine as build

# Enable CGO for building with C dependencies
ENV CGO_ENABLED=1

# Install GCC and necessary libraries for compiling
RUN apk add --no-cache gcc musl-dev

# Set the working directory for the build process
WORKDIR /build

# Copy the entire source code into the container
COPY . .

# Rename the default environment configuration file to env.go
ADD env/env.go.default env/env.go

# Download and tidy up Go module dependencies
RUN go mod tidy

# Build the Go application binary
RUN go build -o app .

# Base image for running the compiled application
FROM alpine:3.20.3

# Set Gin to operate in release mode
ENV GIN_MODE=release

# Copy the built application binary from the build stage
COPY --from=build /build/app .

# Expose the application port to allow external access
EXPOSE 8080

# Create a directory for SQLite database files
RUN mkdir db

# Define a volume for persistent database storage
VOLUME /db

# Set the entry point to run the compiled application binary
ENTRYPOINT ["./app"]
