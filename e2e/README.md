# Ubergang End-to-End Tests

Comprehensive E2E testing setup for the Ubergang authentication proxy using Playwright.

## Test Structure

```txt
e2e/
├── tests/
│   ├── frontend/          # Mocked backend tests - fast UI testing
│   │   ├── auth.spec.ts   # Authentication UI flows
│   │   └── dashboard.spec.ts # Dashboard functionality
│   └── integration/       # Real backend tests - full integration
│       ├── auth-flow.spec.ts # Complete auth flows
│       └── proxy.spec.ts     # Proxy functionality
├── fixtures/              # Mock data and responses
├── utils/                 # Test helpers and utilities
└── playwright.config.ts   # Main configuration
```

## Test Projects

### 1. Frontend Mocked Tests (`frontend-mocked`)

- **Purpose**: Fast UI testing with mocked backend responses
- **Base URL**: `http://localhost:5173` (Vite dev server)
- **Benefits**:
  - Test error scenarios easily
  - Fast execution
  - Network timeouts, 500 errors, malformed responses
  - UI-focused validation

### 2. Integration Real Backend (`integration-real-backend`)

- **Purpose**: Full integration testing with real Go backend
- **Base URL**: `http://localhost:443` (Your Go server)
- **Benefits**:
  - Full integration validation
  - Test actual authentication flows
  - Verify proxy behavior
  - Database interactions

## Prerequisites

### 1. Install Dependencies

```bash
cd e2e
npm install
npx playwright install
```

### 2. Frontend Setup

Ensure your React frontend can run on port 5173:

```bash
cd ../web
npm install
npm run dev  # Should start on localhost:5173
```

### 3. Backend Setup (for integration tests)

Your Go server should be running with this command line:

```sh
go run server.go --test-mode --local-dev --db=test.db --ssh=11022 \
   --https=11433 --http=11080 --metrics=19090 --verbose
```

## Running Tests

### Run All Tests

```bash
npm test
```

### Run Specific Test Suites

```bash
# Frontend mocked tests only (fast)
npm run test:frontend

# Integration tests only (requires backend)
npm run test:integration

# Run with UI (interactive)
npm run test:ui

# Debug mode
npm run test:debug
```

### Run in Headed Mode

```bash
npm run test:headed
```
