#!/bin/bash
set -e

echo "Starting Ubergang server..."
# Start the app in background with flags matching Playwright config (port 10443)
ubergang --test-mode --db=test.db --verbose --https=10443 --http=10080 2>&1 &
PID=$!

echo "Waiting for server to be ready on port 10443..."
# Wait for the server to be responsive
timeout 30 sh -c "until nc -z localhost 10443; do sleep 1; done"

echo "Server is up. Running tests..."
export SKIP_FRONTEND_DEV_SERVER=true
npm run test:integration
TEST_EXIT_CODE=$?

echo "Tests finished with code $TEST_EXIT_CODE"
kill $PID
exit $TEST_EXIT_CODE
