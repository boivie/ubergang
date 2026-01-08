# Ubergang - Authentication Proxy Server

## Purpose

Ubergang is a sophisticated authentication and authorization proxy server
written in Go that provides secure access to backend services. It combines
HTTP/HTTPS reverse proxying with SSH tunneling capabilities, offering
multi-factor authentication through WebAuthn, PIN-based authentication, and SSH
key management. The application serves as a security gateway that sits between
users and backend services, enforcing authentication and authorization policies.

## Key Directories Structure

### Backend

- **`server/`** - Main server implementation and business logic
  - **`api/`** - API type definitions and interfaces
  - **`auth/`** - Authentication services (WebAuthn, SSH keys)
  - **`backends/`** - Backend service management and routing
  - **`cert/`** - TLS certificate management with Let's Encrypt
  - **`common/`** - Shared utilities (credentials, HTTP, passwords, IDs)
  - **`db/`** - Database layer using BoltDB for persistence
  - **`log/`** - Structured logging implementation
  - **`models/`** - Generated protobuf models for data structures
  - **`proxy/`** - HTTP reverse proxy with authentication
  - **`rest/`** - REST API endpoints for web interface
  - **`session/`** - Session management and cookies
  - **`ssh_server/`** - SSH server implementation for tunneling
  - **`storage/`** - File-based configuration storage
  - **`wa/`** - WebAuthn integration for passwordless authentication

### Frontend Application

- **`web/`** - Modern React frontend (Vite + TypeScript)
  - Vite-based build system with Tailwind CSS

### Configuration & Data

- **`config/`** - Runtime configuration and data

### Tools & Build

- **`protos/`** - Protocol Buffer definitions for data models
- **`tools/ugcert/`** - CLI tool for SSH certificate management
- **`Dockerfile`** - Multi-stage Docker build configuration

## Technology Stack

### Backend Stack

- **Language**: Go 1.22+
- **Web Framework**: Gorilla Mux for HTTP routing
- **Database**: BoltDB (embedded key-value store)
- **Authentication**: WebAuthn for passwordless auth, JWT tokens
- **Security**: TLS with Let's Encrypt, SSH server
- **Serialization**: Protocol Buffers for data models
- **Monitoring**: Prometheus metrics, pprof debugging
- **Configuration**: YAML-based file storage

### Key Go Dependencies

- `github.com/go-webauthn/webauthn` - WebAuthn implementation
- `github.com/gliderlabs/ssh` - SSH server library
- `github.com/gorilla/mux` - HTTP request router
- `go.etcd.io/bbolt` - Pure Go key/value database
- `github.com/prometheus/client_golang` - Metrics collection
- `golang.org/x/crypto` - SSH and cryptographic functions

### Frontend Stack

- **web/**: React 18 + Vite + Tailwind CSS + TypeScript

## Available Commands

### Backend (Go)

```bash
# Build the application
go build -o ubergang .

# Run tests
go test ./...
```

### Frontend Development

```bash
# web/ (Vite)
cd web
npm install
npm run dev     # Development server
npm run build   # Production build
npm run lint    # ESLint
```

## Architecture Patterns

### Design Patterns

1. **Layered Architecture** - Clear separation between HTTP handlers, business
   logic, and data persistence
2. **Dependency Injection** - Services injected into handlers for testability
3. **Repository Pattern** - Abstract database operations through storage
   interfaces
4. **Middleware Chain** - HTTP middleware for logging, metrics, authentication
5. **Event-Driven** - Channel-based communication for session updates

### Security Architecture

1. **Defense in Depth** - Multiple authentication factors (WebAuthn + PIN/SSH)
2. **Zero-Trust Proxy** - All backend access goes through authenticated proxy
3. **Certificate Management** - Automated TLS with Let's Encrypt
4. **Session Management** - Secure cookie-based sessions with expiration
5. **SSH Tunneling** - Secure SSH connections for backend access

### Key Components

- **Multi-Protocol Server** - Simultaneous HTTP/HTTPS (port 443/80), SSH
  (10022), metrics (9090)
- **Authentication Provider** - WebAuthn, PIN codes, SSH public keys
- **Reverse Proxy** - Routes authenticated requests to configured backends
- **Configuration Management** - File-based YAML configuration with hot reload
- **Monitoring & Observability** - Prometheus metrics, structured logging, pprof
