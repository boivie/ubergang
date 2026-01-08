import { test, expect, CDPSession } from "@playwright/test";
import { TestHelper } from "../../utils/test-helpers";
import { TestingSetupResponse } from "../../../web/src/api/api_types";

test.describe("Authentication Frontend Tests (Full stack)", () => {
  let testHelper: TestHelper;
  let devTools: CDPSession;
  let authenticatorId: string;

  test.beforeEach(async ({ page }) => {
    testHelper = new TestHelper(page);

    console.log("Setting up");
    const response = await page.request.post("/api/testing/setup", {});
    const responseBody = (await response.json()) as TestingSetupResponse;
    const signinUrl = responseBody.signinUrl;

    devTools = await page.context().newCDPSession(page);
    await devTools.send("WebAuthn.enable");
    const response2 = await devTools.send("WebAuthn.addVirtualAuthenticator", {
      options: {
        protocol: "ctap2",
        transport: "internal",
        hasUserVerification: true,
        isUserVerified: true,
        hasResidentKey: true,
      },
    });
    authenticatorId = response2.authenticatorId;

    page.on("console", (msg) => console.log(msg.text()));

    await page.goto(signinUrl);
  });

  test.afterEach(async ({ page }) => {
    if (authenticatorId) {
      console.log("Removing virtual authenticator", authenticatorId);
      await devTools.send("WebAuthn.removeVirtualAuthenticator", {
        authenticatorId,
      });
    }
    await devTools.detach();
  });

  test.describe("when user is enrolled", () => {
    test.beforeEach(async ({ page }) => {
      await testHelper.waitForPageLoad();
      await testHelper.expectToBeOnDashboard();
      await page.getByRole("link", { name: "Create a passkey" }).click();
      await page.getByRole("button", { name: "Start enrollment" }).click();
      await expect(page.locator("p")).toHaveText(
        /Your passkey has been successfully saved/,
      );
    });

    test("enroll new credential", async ({ page }) => {
      await page.fill("input#passkey-name", "Test Key 1");
      await page.getByRole("button", { name: "Finish" }).click();
      await page.getByRole("link", { name: "Go To Dashboard" }).click();
      await testHelper.expectToBeOnDashboard();
      const passkeysSection = page
        .locator("h2", { hasText: "Passkeys" })
        .locator("+ div");
      await expect(passkeysSection.locator("h4")).toHaveText("Test Key 1");
    });

    test("signin with enrolled credential", async ({ page }) => {
      await page.context().clearCookies();

      await page.goto("/signin");
      await testHelper.expectToBeOnDashboard();
    });

    test("signin with pin", async ({ page }) => {
      const browser = page.context().browser();
      const context2 = await browser!.newContext();
      const page2 = await context2.newPage();
      page2.on("console", (msg) => console.log(msg.text()));
      await page2.goto("/signin");
      const testHelper2 = new TestHelper(page2);
      await testHelper2.fillEmailForm("hello@example.com");
      await testHelper2.clickLoginButton();
      await page2
        .getByRole("button", { name: "Sign in with another device" })
        .click();

      await expect(
        page2.getByRole("heading", { name: "Or Enter PIN" }),
      ).toBeVisible();

      const pinCode = await page2.locator("#pin-container").textContent();
      if (!pinCode) {
        throw new Error("PIN code not found");
      }

      await page.goto("/confirm");
      await expect(
        page.getByRole("heading", { name: "Confirm Sign In" }),
      ).toBeVisible();
      await page.keyboard.type(pinCode);
      await page.keyboard.press("Enter");

      await page.getByRole("button", { name: "Confirm with Passkey" }).click();
      await testHelper2.expectToBeOnDashboard();
    });
  });
});
