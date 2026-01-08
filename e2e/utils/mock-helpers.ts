import { Page, Route } from "@playwright/test";
import { mockResponses } from "../fixtures/mock-responses";

export class MockHelper {
  constructor(private page: Page) {}

  async mockSuccessfulAuth() {
    await this.page.route("**/api/signin/start", async (route: Route) => {
      await route.fulfill(mockResponses.signin.start);
    });

    await this.page.route("**/api/signin/email", async (route: Route) => {
      await route.fulfill(mockResponses.signin.email);
    });

    await this.page.route("**/api/signin/webauthn", async (route: Route) => {
      await route.fulfill(mockResponses.signin.webauthn);
    });

    await this.page.route("**/api/user/me", async (route: Route) => {
      await route.fulfill(mockResponses.user.get);
    });
  }

  async clearAllMocks() {
    await this.page.unroute("**/*");
  }
}
