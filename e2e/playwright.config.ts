import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./tests",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: "html",

  use: {
    trace: "on-first-retry",
    screenshot: "only-on-failure",
  },

  projects: [
    {
      name: "frontend-mocked",
      testDir: "./tests/frontend",
      use: {
        ...devices["Desktop Chrome"],
        baseURL: "http://localhost:5173", // Vite dev server
      },
    },
    {
      name: "integration-real-backend",
      testDir: "./tests/integration",
      use: {
        ...devices["Desktop Chrome"],
        baseURL: "https://localhost:10443", // Your Go server (HTTPS)
        ignoreHTTPSErrors: true, // For self-signed certs in dev
      },
      workers: 1,
    },
  ],

  webServer: [
    {
      command: "cd ../web && npm run dev",
      url: "http://localhost:5173",
      reuseExistingServer: !process.env.CI,
      timeout: 12 * 1000,
    },
  ],
});
