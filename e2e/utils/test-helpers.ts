import { Page, expect } from "@playwright/test";

export class TestHelper {
  constructor(private page: Page) {}

  async waitForPageLoad() {
    await this.page.waitForLoadState("networkidle");
  }

  async fillEmailForm(email: string) {
    await this.page.fill('input[type="email"]', email);
  }

  async clickLoginButton() {
    await this.page.click('[data-testid="login-button"]');
  }

  async expectToBeOnDashboard() {
    //await expect(this.page).toHaveURL(/^\/(\?_ubergang_session=[^&]*)?$/);
    await expect(this.page.locator("h1")).toHaveText("Hello, John Doe");
  }

  async expectToBeOnBackends() {
    await expect(this.page).toHaveURL("/backends/");
  }

  async expectToBeOnUsers() {
    await expect(this.page).toHaveURL("/users/");
  }

  async takeScreenshotOnFailure(testName: string) {
    await this.page.screenshot({
      path: `screenshots/failure-${testName}-${Date.now()}.png`,
      fullPage: true,
    });
  }
}
