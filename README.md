# Ubergang

[![License](https://img.shields.io/github/license/boivie/ubergang?style=for-the-badge)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker)](https://hub.docker.com/repository/docker/boivie/ubergang/tags)

Ubergang is a self-hosted, identity-aware proxy that secures your internal
services.

It acts as a single entry point for your homelab, exposing HTTP, SSH, and MQTT
services to the internet behind WebAuthn/Passkeys authentication.

Unlike complex enterprise solutions or simple Nginx setups, Ubergang is designed
to be the gateway you run at the edge of your private network. It combines a
reverse proxy, SSH gateway, and authentication server into a single Go binary
with an embedded database and zero external dependencies.

## Key Features

- **Identity Proxy**: Expose internal web services (Home Assistant, Grafana) to
  the internet, authenticated via WebAuthn/Passkeys.
- **SSH Gateway**: Access your internal servers via SSH from anywhere without a
  VPN.
- **MQTT Proxy**: Secure TLS entry point for IoT devices connecting to your
  internal MQTT broker.
- **Automatic TLS**: Built-in Let's Encrypt support with automatic certificate
  management.
- **Single Binary**: No external database or complex dependencies required.
- **Web UI**: Modern React frontend for configuration and management.

## Quick Start

### Installation

#### Using Docker (Preferred)

The easiest way to run Ubergang. Images are available for both `amd64` and
`arm64`.

```bash
docker run -p 8443:8443 -p 8080:8080 boivie/ubergang:latest
```

#### From Source

If you prefer building it yourself:

```bash
# Clone the repository
git clone https://github.com/yourusername/ubergang.git
cd ubergang

# Build the frontend
cd web
npm install
npm run build
cd ..

# Build the server (bundles the frontend)
go mod download
go build -o ubergang .
```

### Configuration

You need to bootstrap the server to create the first admin account and set the
domain.

```bash
# Configure the server (interactive)
./ubergang --configure

# Create a new administrator account (outputs a signin URL)
./ubergang --account

# Start the server
./ubergang
```

## Contributing

Interested in contributing to Ubergang? Check out our [Contributing
Guide](CONTRIBUTING.md) for development setup, code style guidelines, and how to
submit changes.

## License

This project is licensed under [Apache License 2.0](LICENSE).
