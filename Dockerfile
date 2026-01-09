# --- Stage 1: Frontend Build ---
FROM --platform=$BUILDPLATFORM node:25-slim AS nodebuilder
WORKDIR /src/web

# Cache dependencies separately
COPY web/package.json web/package-lock.json ./
RUN npm ci

# Copy source and build
COPY web/ ./
RUN npm run build

# --- Stage 2: Backend Build ---
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
ARG TARGETOS TARGETARCH
ARG BUILD_VERSION=latest

WORKDIR /src

# 1. Cache Go modules - do this before copying the whole source!
COPY go.mod go.sum ./
RUN go mod download

# 2. Copy source and Frontend assets from previous stage
COPY . .
COPY --from=nodebuilder /src/web/dist/ /src/web/dist/

# 3. Build the binary
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-w -s -X main.version=${BUILD_VERSION}" -o /ubergang .

# --- Stage 3: Final Image ---
FROM alpine:3.21 AS final

LABEL maintainer="boivie"

RUN addgroup -g 1000 -S ubergang && adduser -u 1000 -S ubergang -G ubergang
USER ubergang

# Set up volumes and ports
EXPOSE 8080 8443 1883

# Copy the binary from the builder
COPY --from=builder --chown=ubergang:ubergang /ubergang /ubergang

# Use a clean entrypoint
ENTRYPOINT ["/ubergang"]
CMD ["--db", "/data/ubergang.db", "--https", "8443", "--http", "8080"]
