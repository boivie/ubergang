import { test, expect } from "@playwright/test";
import { MockHelper } from "../../utils/mock-helpers";
import { TestHelper } from "../../utils/test-helpers";

test.describe("Authentication Frontend Tests (Mocked)", () => {
  let mockHelper: MockHelper;
  let testHelper: TestHelper;

  test.beforeEach(async ({ page }) => {
    mockHelper = new MockHelper(page);
    testHelper = new TestHelper(page);

    page.on("console", (msg) => console.log(msg.text()));

    const mockedCredential = {
      id: "mocked-credential-id",
      rawId: new ArrayBuffer(8),
      response: {
        clientDataJSON: new ArrayBuffer(8),
        attestationObject: new ArrayBuffer(8),
      },
      type: "public-key",
    };

    await page.addInitScript((mockedData) => {
      Object.defineProperty(window.navigator.credentials, "get", {
        value: (e: any) => {
          if (e.mediation == "conditional") {
            console.log("Not returning conditional assertion");
            return;
          }
          console.log("Returning mocked assertion");
          return Promise.resolve(mockedData);
        },
        writable: true,
      });
    }, mockedCredential);

    await page.goto("/signin");
  });

  test("successful login flow", async ({ page }) => {
    await mockHelper.mockSuccessfulAuth();
    await testHelper.waitForPageLoad();

    await testHelper.fillEmailForm("hello@example.com");
    await testHelper.clickLoginButton();

    await testHelper.expectToBeOnDashboard();
  });
});
