# Contributing to Ubergang

First off, thanks for taking the time to contribute!

All types of contributions are encouraged and valued. Whether you're fixing
bugs, adding features, improving documentation, or helping other users, your
contributions help make Ubergang better for the homelab community.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Submitting Changes](#submitting-changes)
- [Reporting Bugs](#reporting-bugs)
- [Suggesting Features](#suggesting-features)

## Code of Conduct

Be respectful, constructive, and collaborative. We're all here to build
something useful.

## How Can I Contribute?

### Ways to Help

- **Report bugs** - Found something broken? Let us know!
- **Suggest features** - Have an idea for making Ubergang better? Open an issue.
- **Improve documentation** - Help others understand and use Ubergang.
- **Write code** - Fix bugs, add features, or improve existing functionality.
- **Test and provide feedback** - Try out new features and report your
  experience.
- **Share your experience** - Write blog posts, create tutorials, or share on
  social media.

## Development Setup

Ubergang consists of a Go backend and a React frontend. For the best development
experience, run them separately.

### Prerequisites

- **Go 1.25+** - [Install Go](https://golang.org/doc/install)
- **Node.js v24++** and npm - [Install Node](https://nodejs.org/)
- **Git** - [Install Git](https://git-scm.com/)

### Clone and Setup

```bash
# Clone the repository
git clone https://github.com/yourusername/ubergang.git
cd ubergang

# Install Go dependencies
go mod download

# Install frontend dependencies
cd web
npm install
cd ..
```

### Running in Development Mode

#### Terminal 1: Frontend Development Server

```bash
cd web
npm run dev
```

This starts the Vite dev server with hot-reload on <http://localhost:5173>

#### Terminal 2: Backend Server

```bash
go run server.go --local-dev --verbose
```

The `--local-dev` flag configures the backend to work with the Vite dev server.

### Building for Production

```bash
# Build the frontend
cd web
npm run build
cd ..

# Build the complete server (includes bundled frontend)
go build -o ubergang .
```

### Running Tests

#### Unit Tests

```bash
# Run Go tests
go test ./...

# Run frontend linting
cd web
npm run lint
```

#### Integration Tests

In one terminal:

```bash
go run server.go --test-mode --local-dev --db=test.db --verbose
```

In another terminal:

```bash
cd e2e
npm install
npm test
```

## Project Structure

```txt
ubergang/
├── server/           # Go backend
│   ├── api/         # API interfaces
│   ├── auth/        # Authentication (WebAuthn, SSH)
│   ├── backends/    # Backend service management
│   ├── cert/        # TLS certificate management
│   ├── db/          # Database layer
│   ├── proxy/       # HTTP reverse proxy
│   ├── rest/        # REST API endpoints
│   ├── session/     # Session management
│   ├── ssh_server/  # SSH gateway implementation
│   └── wa/          # WebAuthn integration
├── web/             # React frontend (Vite + TypeScript)
├── protos/          # Protocol Buffer definitions
└── tools/           # Additional CLI tools
```

### General Guidelines

- **Security first**: This is an authentication proxy - security is paramount
  - Never log sensitive data (passwords, tokens, keys)
  - Always validate and sanitize input
- **Keep it simple**: Ubergang is designed for homelabs, not enterprises
  - Avoid over-engineering or adding unnecessary complexity
  - Prefer straightforward solutions over clever ones
- **No external dependencies**: One of Ubergang's core principles
  - Avoid adding dependencies that require external services

## Submitting Changes

### Pull Request Process

- **Fork the repository** and create a branch from `main`

```bash
git checkout -b feature/my-new-feature
```

- **Make your changes** following the code style guidelines

- **Test your changes** thoroughly

  - Run existing tests: `go test ./...`
  - Test manually with `--local-dev` mode
  - Verify the production build works

- **Commit your changes** with clear, descriptive messages

```bash
git commit -m "Add MQTT connection pooling for improved performance"
```

- **Push to your fork**

```bash
git push origin feature/my-new-feature
```

- **Open a Pull Request** with:
  - Clear title describing what the PR does
  - Description explaining why the change is needed
  - Any relevant issue numbers (e.g., "Fixes #123")
  - Screenshots for UI changes

NOTE: We don't allow merge commits. Rewrite your branch and amend your commits
after fixing code review comments. Your commits will be cherry-picked/rebased
onto the main branch.

## Reporting Bugs

Found a bug? Please open an issue with:

- **Clear title** - Summarize the problem
- **Description** - What happened vs. what you expected
- **Steps to reproduce** - How can we recreate the issue?
- **Environment** - OS, Docker version, Ubergang version
- **Logs** - Include relevant log output (redact sensitive info!)
- **Configuration** - Share relevant config (redact secrets!)

## Suggesting Features

Have an idea? Open an issue with:

- **Use case** - What problem does this solve for homelab users?
- **Proposed solution** - How do you envision it working?
- **Alternatives** - What other approaches did you consider?
- **Scope** - Is this a small enhancement or major feature?

Remember: Ubergang is focused on being a simple, secure gateway for homelabs.
Features should align with this goal.

## Questions?

If you have questions about contributing, feel free to:

- Open a discussion issue
- Ask in your pull request
- Check existing issues for similar questions

Thank you for contributing to Ubergang!
