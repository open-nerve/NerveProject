import { expect, type Page } from "@playwright/test";

/** Add a condition through the real toolbar, then return its chip with the newly opened value control. */
export async function addCondition(page: Page, name: string) {
  await page.getByRole("button", { name: "Add filter", exact: true }).click();
  const option = page.getByRole("option", { name, exact: true });
  await expect(option).toBeVisible();
  await option.click();
  return page.getByRole("button", { name: "Remove filter", exact: true }).last().locator("..");
}
