import type { Locator, Page } from "@playwright/test";
import { expect } from "@playwright/test";

/** Uses the real cmdk roving selection; an empty root query never invokes retired search. */
export async function highlightCommand(page: Page, name: RegExp, input: Locator = page.getByRole("combobox")) {
  const target = page.getByRole("option", { name });
  await expect(target).toBeVisible();
  const count = await page.getByRole("option").count();
  const advance = async (remaining: number): Promise<void> => {
    if (remaining === 0 || (await target.getAttribute("aria-selected")) === "true") return;
    await input.press("ArrowDown");
    await advance(remaining - 1);
  };
  await advance(count + 1);
  await expect(target).toHaveAttribute("aria-selected", "true");
  return target;
}

export async function selectCommand(page: Page, name: RegExp, input: Locator = page.getByRole("combobox")) {
  await highlightCommand(page, name, input);
  await input.press("Enter");
}

export async function openCommands(page: Page) {
  await page.keyboard.press("ControlOrMeta+k");
  await expect(page.getByRole("combobox")).toBeFocused();
  await expect(page.getByRole("combobox")).toHaveValue("");
}
