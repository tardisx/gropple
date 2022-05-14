# Start from golang base image
FROM golang:1.18.2-alpine3.15 as builder

# Install git. (alpine image does not have git in it)
RUN apk update && apk add --no-cache git curl

# Set current working directory
WORKDIR /app

RUN curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /app/yt-dlp
RUN chmod a+x /app/yt-dlp

# Note here: To avoid downloading dependencies every time we
# build image. Here, we are caching all the dependencies by
# first copying go.mod and go.sum files and downloading them,
# to be used every time we build the image if the dependencies
# are not changed.

# Copy go mod and sum files
COPY go.mod ./
COPY go.sum ./

# Download all dependencies.
RUN go mod download

# Now, copy the source code
COPY . .

# Note here: CGO_ENABLED is disabled for cross system compilation
# It is also a common best practise.

# Build the application.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/gropple .

# Finally our multi-stage to build a small image
# Start a new stage from scratch
FROM alpine:3.15.4

# Copy the Pre-built binary file
COPY --from=builder /app/bin/gropple .
COPY --from=builder /app/yt-dlp /bin/

# Install things we need to support yt-dlp
RUN apk update && apk add --no-cache python3 ffmpeg

# Run executable
CMD ["./gropple", "--config-path", "/config/gropple.json"]
