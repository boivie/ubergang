import { test, expect, CDPSession } from "@playwright/test";
import { TestHelper } from "../../utils/test-helpers";
import { ApiTestingSetupResponse } from "../../../web/src/api/api_types";

test.describe("Backend Management Tests", () => {
  let testHelper: TestHelper;

  test.beforeEach(async ({ page }) => {
    testHelper = new TestHelper(page);

    const response = await page.request.post("/api/testing/setup", {});
    const responseBody = (await response.json()) as ApiTestingSetupResponse;
    const signinUrl = responseBody.signinUrl;

    page.on("console", (msg) => console.log(msg.text()));

    await page.goto(signinUrl);
    await testHelper.waitForPageLoad();
    await testHelper.expectToBeOnDashboard();
  });

  test("shows empty state when no backends exist", async ({ page }) => {
    await page.goto("/backends/");
    await testHelper.expectToBeOnBackends();

    // Check that the empty state message is visible
    await expect(page.getByText("No backends yet")).toBeVisible();
    await expect(
      page.getByText("Get started by creating a new backend."),
    ).toBeVisible();

    // The "New backend" button should still be there
    await expect(page.getByRole("link", { name: "New backend" })).toBeVisible();
  });

  test("create new backend", async ({ page }) => {
    await page.goto("/backends/");
    await testHelper.expectToBeOnBackends();
    await page.getByRole("link", { name: "New backend" }).click();
    await expect(page).toHaveURL("/backends/new");
    await page.fill("input#fqdn", "test.example.com");
    await page.getByRole("button", { name: "Create Backend" }).click();
    await expect(page).toHaveURL("/backends/edit/test.example.com");

    await page.goto("/backends/");
    await testHelper.expectToBeOnBackends();
    const backendItem = page
      .locator("ul li")
      .filter({ hasText: "test.example.com" });
    await expect(backendItem).toBeVisible();
  });

  test.describe("when a backend is created", () => {
    test.beforeEach(async ({ page }) => {
      await page.goto("/backends/new");
      await page.fill("input#fqdn", "test.example.com");
      await page.getByRole("button", { name: "Create Backend" }).click();
      await expect(page).toHaveURL("/backends/edit/test.example.com");
    });

    test("edit upstream url", async ({ page }) => {
      await page.goto("/backends/edit/test.example.com");
      await page.fill("input#upstream", "http://localhost:8080");
      await page.getByRole("button", { name: "Save Changes" }).click();
      await expect(page).toHaveURL("/backends/");

      // Check that it's saved.
      await page.goto("/backends/edit/test.example.com");
      await expect(page.locator("input#upstream")).toHaveValue(
        "http://localhost:8080",
      );
    });

    test("add a header", async ({ page }) => {
      await page.goto("/backends/edit/test.example.com");
      await page.getByRole("button", { name: "Add Header" }).click();
      await page.getByLabel("Header name").fill("X-Test-Header");
      await page.getByLabel("Header value").fill("test-value");
      await page.getByRole("button", { name: "Save Changes" }).click();
      await expect(page).toHaveURL("/backends/");

      // Check that it's saved.
      await page.goto("/backends/edit/test.example.com");
      await expect(page.getByLabel("Header name")).toHaveValue("X-Test-Header");
      await expect(page.getByLabel("Header value")).toHaveValue("test-value");
    });

    test("add multiple headers", async ({ page }) => {
      await page.goto("/backends/edit/test.example.com");
      await page.getByRole("button", { name: "Add Header" }).click();
      await page.getByLabel("Header name").nth(0).fill("X-Test-Header-1");
      await page.getByLabel("Header value").nth(0).fill("test-value-1");

      await page.getByRole("button", { name: "Add Header" }).click();
      await page.getByLabel("Header name").nth(1).fill("X-Test-Header-2");
      await page.getByLabel("Header value").nth(1).fill("test-value-2");

      await page.getByRole("button", { name: "Save Changes" }).click();
      await expect(page).toHaveURL("/backends/");

      // Check that it's saved.
      await page.goto("/backends/edit/test.example.com");
      await expect(page.getByLabel("Header name").nth(0)).toHaveValue(
        "X-Test-Header-1",
      );
      await expect(page.getByLabel("Header value").nth(0)).toHaveValue(
        "test-value-1",
      );
      await expect(page.getByLabel("Header name").nth(1)).toHaveValue(
        "X-Test-Header-2",
      );
      await expect(page.getByLabel("Header value").nth(1)).toHaveValue(
        "test-value-2",
      );
    });

    test("delete a header", async ({ page }) => {
      // First, add two headers
      await page.goto("/backends/edit/test.example.com");
      await page.getByRole("button", { name: "Add Header" }).click();
      await page.getByLabel("Header name").nth(0).fill("X-To-Delete");
      await page.getByLabel("Header value").nth(0).fill("delete-me");

      await page.getByRole("button", { name: "Add Header" }).click();
      await page.getByLabel("Header name").nth(1).fill("X-To-Keep");
      await page.getByLabel("Header value").nth(1).fill("keep-me");
      await page.getByRole("button", { name: "Save Changes" }).click();
      await expect(page).toHaveURL("/backends/");

      // Now, delete one
      await page.goto("/backends/edit/test.example.com");
      await expect(page.getByLabel("Header name")).toHaveCount(2);
      await page
        .getByRole("button", { name: "Remove X-To-Delete header" })
        .click();

      await expect(page.getByLabel("Header name")).toHaveCount(1);
      await page.getByRole("button", { name: "Save Changes" }).click();
      await expect(page).toHaveURL("/backends/");

      // Check that it's saved
      await page.goto("/backends/edit/test.example.com");
      await expect(page.getByLabel("Header name")).toHaveCount(1);
      await expect(page.getByLabel("Header name")).toHaveValue("X-To-Keep");
      await expect(page.getByLabel("Header value")).toHaveValue("keep-me");
    });

    test("delete a backend", async ({ page }) => {
      await page.goto("/backends/");
      await testHelper.expectToBeOnBackends();

      const backendItem = page
        .locator("ul li")
        .filter({ hasText: "test.example.com" });
      await expect(backendItem).toBeVisible();

      // The delete button is inside the list item for the backend
      await backendItem.getByRole("button", { name: "Delete backend" }).click();

      // Now the modal should be visible
      const dialog = page.getByRole("alertdialog");
      await expect(dialog).toBeVisible();
      await expect(dialog).toContainText(
        "Are you sure you want to remove the backend test.example.com?",
      );

      // Click the remove button in the dialog
      await dialog.getByRole("button", { name: "Remove" }).click();

      // The page should redirect and the item should be gone
      await expect(page).toHaveURL("/backends/");
      await testHelper.expectToBeOnBackends();
      await expect(backendItem).not.toBeVisible();
    });

    test("cancel deleting a backend", async ({ page }) => {
      await page.goto("/backends/");
      await testHelper.expectToBeOnBackends();

      const backendItem = page
        .locator("ul li")
        .filter({ hasText: "test.example.com" });
      await expect(backendItem).toBeVisible();

      // The delete button is inside the list item for the backend
      await backendItem.getByRole("button", { name: "Delete backend" }).click();

      // Now the modal should be visible
      const dialog = page.getByRole("alertdialog");
      await expect(dialog).toBeVisible();

      // Click the cancel button in the dialog
      await dialog.getByRole("button", { name: "Cancel" }).click();

      // The dialog should disappear and the item should still be there
      await expect(dialog).not.toBeVisible();
      await expect(backendItem).toBeVisible();
    });
  });
});
