# Base image
FROM alpine:3.16.2

# Gin environment
ENV GIN_MODE=release

# Change workdir
WORKDIR /go/src/app

# Copy dist
COPY app .

# Entrypoint
ENTRYPOINT ["./app"]
