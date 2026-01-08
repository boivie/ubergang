import { test, expect } from "@playwright/test";
import { TestHelper } from "../../utils/test-helpers";
import { ApiTestingSetupResponse } from "../../../web/src/api/api_types";

test.describe("User Management Tests", () => {
  let testHelper: TestHelper;
  let responseBody: ApiTestingSetupResponse;

  test.beforeEach(async ({ page }) => {
    testHelper = new TestHelper(page);

    const response = await page.request.post("/api/testing/setup", {});
    responseBody = (await response.json()) as ApiTestingSetupResponse;
    const signinUrl = responseBody.signinUrl;

    page.on("console", (msg) => console.log(msg.text()));

    await page.goto(signinUrl);
    await testHelper.waitForPageLoad();
    await testHelper.expectToBeOnDashboard();
  });

  test("shows the admin user by default", async ({ page }) => {
    await page.goto("/users/");
    await testHelper.expectToBeOnUsers();

    // The admin user from the testing setup should be visible
    const userItem = page
      .locator("ul li")
      .filter({ hasText: "hello@example.com" });
    await expect(userItem).toBeVisible();
  });

  test("create new user", async ({ page }) => {
    await page.goto("/users/");
    await testHelper.expectToBeOnUsers();
    await page.getByRole("link", { name: "New user" }).click();
    await expect(page).toHaveURL("/users/new");
    await page.fill("input#email", "user1@example.com");
    await page.getByRole("button", { name: "Create User" }).click();
    await expect(page).toHaveURL(/\/users\/edit\/.+/);

    await page.goto("/users/");
    await testHelper.expectToBeOnUsers();
    const userItem = page
      .locator("ul li")
      .filter({ hasText: "user1@example.com" });
    await expect(userItem).toBeVisible();
  });

  test.describe("when a user is created", () => {
    let userId: string;

    test.beforeEach(async ({ page }) => {
      await page.goto("/users/new");
      await page.fill("input#email", "testuser@example.com");
      await page.getByRole("button", { name: "Create User" }).click();
      await expect(page).toHaveURL(/\/users\/edit\/.+/);
      const url = page.url();
      userId = url.split("/").pop()!;
    });

    test("edit user details", async ({ page }) => {
      await page.goto(`/users/edit/${userId}`);
      await page.fill("input#displayName", "Updated Test User");
      await page.check('input[name="admin"]');
      await page.getByRole("button", { name: "Save Changes" }).click();
      await expect(page).toHaveURL("/users/");

      // Check that it's saved.
      const userItem = page.locator(`[data-testid="user-row-${userId}"]`);
      await expect(userItem).toContainText("Updated Test User");
      await expect(userItem).toContainText("admin");
    });

    test("add and remove allowed hosts", async ({ page }) => {
      await page.goto(`/users/edit/${userId}`);

      // Add a host
      await page.getByRole("button", { name: "Add Host" }).click();
      await page.getByLabel("Allowed host").last().fill("host1.example.com");

      // Add another host
      await page.getByRole("button", { name: "Add Host" }).click();
      await page.getByLabel("Allowed host").last().fill("host2.example.com");

      await page.getByRole("button", { name: "Save Changes" }).click();
      await expect(page).toHaveURL("/users/");

      // Check that they are saved
      await page.goto(`/users/edit/${userId}`);
      await expect(page.getByLabel("Allowed host")).toHaveCount(2);
      expect(await page.getByLabel("Allowed host").nth(0).inputValue()).toBe(
        "host1.example.com",
      );
      expect(await page.getByLabel("Allowed host").nth(1).inputValue()).toBe(
        "host2.example.com",
      );

      // Remove a host
      await page
        .getByRole("button", { name: "Remove host host1.example.com" })
        .click();
      await expect(page.getByLabel("Allowed host")).toHaveCount(1);

      await page.getByRole("button", { name: "Save Changes" }).click();
      await expect(page).toHaveURL("/users/");

      // Check that it's saved
      await page.goto(`/users/edit/${userId}`);
      await expect(page.getByLabel("Allowed host")).toHaveCount(1);
      expect(await page.getByLabel("Allowed host").first().inputValue()).toBe(
        "host2.example.com",
      );
    });

    test("delete a user", async ({ page }) => {
      await page.goto("/users/");
      await testHelper.expectToBeOnUsers();

      const userItem = page.locator(`[data-testid="user-row-${userId}"]`);
      await expect(userItem).toBeVisible();

      await page
        .getByRole("button", { name: "Delete user hello@example.com" })
        .click();

      const dialog = page.getByRole("alertdialog");
      await expect(dialog).toBeVisible();
      await expect(dialog).toContainText(
        "Are you sure you want to remove the user hello@example.com?",
      );

      await dialog.getByRole("button", { name: "Remove" }).click();

      await expect(page).toHaveURL("/users/");
      await testHelper.expectToBeOnUsers();
      await expect(userItem).not.toBeVisible();
    });

    test("cancel deleting a user", async ({ page }) => {
      await page.goto("/users/");
      await testHelper.expectToBeOnUsers();

      const userItem = page.locator(`[data-testid="user-row-${userId}"]`);
      await expect(userItem).toBeVisible();

      await page
        .getByRole("button", { name: "Delete user hello@example.com" })
        .click();

      const dialog = page.getByRole("alertdialog");
      await expect(dialog).toBeVisible();

      await dialog.getByRole("button", { name: "Cancel" }).click();

      await expect(dialog).not.toBeVisible();
      await expect(userItem).toBeVisible();
    });
  });
});
