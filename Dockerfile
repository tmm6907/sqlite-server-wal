# syntax=docker/dockerfile:latest

# Build stage
ARG GO_VERSION=1.23.4
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build

WORKDIR /src
COPY ./db /src/db

# Download dependencies
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

# Install dependencies for CGO
RUN apt-get update && apt-get install -y gcc libc6-dev && rm -rf /var/lib/apt/lists/* 

# Build the binary and place it in a writable directory
ARG TARGETARCH
ENV CGO_ENABLED=1
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,target=. \
    GOARCH=$TARGETARCH go build -o /tmp/server .


################################################################################
# Runtime stage
FROM debian:bookworm-slim AS final

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    tzdata \
    sqlite3 \
    zip && \
    rm -rf /var/lib/apt/lists/*




# Create a non-privileged user
ARG UID=10001
RUN adduser --disabled-password --gecos "" --home "/nonexistent" --shell "/usr/sbin/nologin" --no-create-home --uid "${UID}" appuser

# Ensure directory setup BEFORE switching to non-root user
RUN mkdir -p /src && chown appuser:appuser /src

# Set working directory
WORKDIR /src

# Switch to the non-privileged user
USER appuser

# Copy the binary from the build stage
COPY --from=build /tmp/server /server

# Expose the port
EXPOSE 8000

# Entry point
ENTRYPOINT [ "/server" ]
