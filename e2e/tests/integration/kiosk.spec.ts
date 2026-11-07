import { test, expect, CDPSession, Page } from "@playwright/test";
import * as http from "http";
import { AddressInfo } from "net";
import { TestHelper } from "../../utils/test-helpers";
import { ApiTestingSetupResponse } from "../../../web/src/api/api_types";

// Chrome resolves *.localhost to the loopback address, so this reaches the
// server under test, which proxies it to the upstream server below.
const KIOSK_HOST = "kiosk.localhost";
const KIOSK_URL = `https://${KIOSK_HOST}:10443`;

test.describe("Kiosk sign-in (Full stack)", () => {
  let upstream: http.Server;
  let upstreamPort: number;
  let testHelper: TestHelper;
  let devTools: CDPSession;
  let authenticatorId: string;

  test.beforeAll(async () => {
    upstream = http.createServer((req, res) => {
      res.setHeader("Content-Type", "text/html");
      res.end(
        `<h1>Kiosk backend</h1><p id="path">${req.url}</p>` +
          `<p id="email">${req.headers["x-forwarded-email"] ?? ""}</p>`,
      );
    });
    await new Promise<void>((resolve) => upstream.listen(0, resolve));
    upstreamPort = (upstream.address() as AddressInfo).port;
  });

  test.afterAll(async () => {
    await new Promise((resolve) => upstream.close(resolve));
  });

  // Signs in as the test user, with a passkey, and creates the kiosk backend.
  test.beforeEach(async ({ page }) => {
    testHelper = new TestHelper(page);
    page.on("console", (msg) => console.log(msg.text()));

    const response = await page.request.post("/api/testing/setup", {});
    const { signinUrl } = (await response.json()) as ApiTestingSetupResponse;

    devTools = await page.context().newCDPSession(page);
    await devTools.send("WebAuthn.enable");
    const authenticator = await devTools.send(
      "WebAuthn.addVirtualAuthenticator",
      {
        options: {
          protocol: "ctap2",
          transport: "internal",
          hasUserVerification: true,
          isUserVerified: true,
          hasResidentKey: true,
        },
      },
    );
    authenticatorId = authenticator.authenticatorId;

    await page.goto(signinUrl);
    await testHelper.waitForPageLoad();
    await testHelper.expectToBeOnDashboard();
    await page.getByRole("link", { name: "Create a passkey" }).click();
    await page.getByRole("button", { name: "Start enrollment" }).click();
    await expect(page.locator("p")).toHaveText(
      /Your passkey has been successfully saved/,
    );
    await page.fill("input#passkey-name", "Test Key");
    await page.getByRole("button", { name: "Finish" }).click();

    const created = await page.request.post(`/api/backend/${KIOSK_HOST}`, {
      data: {
        upstreamUrl: `http://localhost:${upstreamPort}`,
        allowKioskSignin: true,
      },
    });
    expect(created.ok()).toBeTruthy();
  });

  test.afterEach(async () => {
    await devTools.send("WebAuthn.removeVirtualAuthenticator", {
      authenticatorId,
    });
    await devTools.detach();
  });

  async function openKiosk(page: Page): Promise<{ kiosk: Page; code: string }> {
    const context = await page
      .context()
      .browser()!
      .newContext({ ignoreHTTPSErrors: true });
    const kiosk = await context.newPage();
    kiosk.on("console", (msg) => console.log("[kiosk]", msg.text()));
    await kiosk.goto(`${KIOSK_URL}/dashboard?room=kitchen&size=large`);

    await expect(
      kiosk.getByRole("heading", { name: "Or sign in from another device" }),
    ).toBeVisible();
    const code = (await kiosk.locator("#kiosk-code").textContent())!.replace(
      /\s/g,
      "",
    );
    expect(code).toMatch(/^\d{6}$/);
    return { kiosk, code };
  }

  test("kiosk is signed in to the backend after approval", async ({ page }) => {
    const { kiosk, code } = await openKiosk(page);

    await page.goto("/kiosk/");
    await expect(
      page.getByRole("heading", { name: "Sign In a Device" }),
    ).toBeVisible();
    await page.keyboard.type(code);
    await page.getByRole("button", { name: "Continue" }).click();
    await expect(
      page.getByText(`wants to sign in to ${KIOSK_HOST}`),
    ).toBeVisible();
    // Both browsers are on the same machine.
    await expect(
      page.getByText("The device seems to be on the same network as you."),
    ).toBeVisible();
    await page.fill("input#kiosk-name", "Kitchen tablet");
    await page.getByText("Until revoked").click();
    await page.getByRole("button", { name: "Approve with Passkey" }).click();
    await expect(
      page.getByText(`The device is now signed in to ${KIOSK_HOST}.`),
    ).toBeVisible();

    // The kiosk goes to the page it wanted, without any input.
    await expect(kiosk.locator("h1")).toHaveText("Kiosk backend");
    await expect(kiosk.locator("#path")).toHaveText(
      "/dashboard?room=kitchen&size=large",
    );
    await expect(kiosk.locator("#email")).toHaveText("hello@example.com");

    // The kiosk's session doesn't work on the admin host, even if its cookie
    // is sent there.
    const cookie = (await kiosk.context().cookies(KIOSK_URL)).find(
      (c) => c.name === "__ug_sess",
    );
    expect(cookie).toBeDefined();
    const me = await kiosk.request.get("https://localhost:10443/api/user/me", {
      headers: { Cookie: `__ug_sess=${cookie!.value}` },
    });
    expect(me.status()).toBe(403);

    // It's listed among the approver's sessions.
    await page.goto("/");
    await expect(page.getByText("Kitchen tablet")).toBeVisible();
    await expect(page.getByText(`Only ${KIOSK_HOST}`)).toBeVisible();

    await kiosk.context().close();
  });

  test("approver that isn't signed in comes back to the approval page", async ({
    page,
  }) => {
    const { kiosk, code } = await openKiosk(page);

    await page.context().clearCookies();
    await page.goto(`/kiosk/${code}`);
    await page.getByRole("link", { name: "Sign in" }).click();
    // The virtual authenticator signs in with the passkey automatically.
    await expect(
      page.getByText(`wants to sign in to ${KIOSK_HOST}`),
    ).toBeVisible();

    await kiosk.context().close();
  });

  test("signing in returns to the original URL", async ({ page }) => {
    await page.context().clearCookies();
    await page.goto(`${KIOSK_URL}/some/page?a=b&c=d`);
    // The virtual authenticator signs in with the passkey automatically.
    await expect(page.locator("h1")).toHaveText("Kiosk backend");
    await expect(page.locator("#path")).toHaveText("/some/page?a=b&c=d");
  });
});
