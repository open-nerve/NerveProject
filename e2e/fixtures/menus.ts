import { expect, type Locator, type Page } from "@playwright/test";

// A page's menus (Headless UI's, which CustomMenu renders), as the keyboard works them.

/**
 * Chooses item from the menu of button as a keyboard user does: Enter on button, which has the focus, opens the menu,
 * which shows item among its items and makes its first one active; ArrowDown moves the active item down to item, and
 * Enter chooses it.
 */
export async function chooseByKeyboard(page: Page, button: Locator, item: string): Promise<void> {
  await expect(button).toBeFocused();
  await page.keyboard.press("Enter");
  const menu = page.getByRole("menu");
  const items = menu.getByRole("menuitem");
  const choice = menu.getByRole("menuitem", { name: item, exact: true });
  await expect(choice).toBeVisible();
  // the active item is the one the menu's aria-activedescendant names
  await expect(menu).toHaveAttribute("aria-activedescendant", (await items.first().getAttribute("id")) ?? "");
  const id = (await choice.getAttribute("id")) ?? "";
  // a press after another, as many as items come before item
  const down = async (presses: number): Promise<void> => {
    if (presses === 0) return;
    await page.keyboard.press("ArrowDown");
    return down(presses - 1);
  };
  const index = await items.evaluateAll((all, wanted) => all.findIndex((each) => each.id === wanted), id);
  expect(index, `${item} is an item of the menu`).toBeGreaterThanOrEqual(0);
  await down(index);
  await expect(menu).toHaveAttribute("aria-activedescendant", id);
  await page.keyboard.press("Enter");
}
