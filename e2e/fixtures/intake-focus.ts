import { expect, type Locator, type Page } from "@playwright/test";

import { intakeDetail, intakeRow } from "./intake-pages";
import type { Issue } from "./issues";
import { chooseByKeyboard } from "./menus";

// The intake page's keys and focus (M7 P5a spec 2.8's R35, 2.12's H20), as its H20 (page) stories check them: what has
// the focus, that the screen shows where it landed, the header's menu by the keyboard, by the pointer and on a narrow
// screen, and the list's toggle there. The page itself, its rows and its regions are intake-pages.ts's.

/** A screen below the layout's lg (1024 px): the list lies over the detail, and a row pressed hides it. */
export const NARROW = { width: 800, height: 720 };

/**
 * What has page's focus, as a screen reader names it: "body" when nothing has (the document's body), else its role
 * (its own, or a link's, a button's, a named section's) and its name ("link WEB-3 Fix", "region Intake work item").
 */
export async function focusedName(page: Page): Promise<string> {
  return page.evaluate(() => {
    const element = document.activeElement;
    if (element === null || element === document.body) return "body";
    const implicit: Record<string, string> = { A: "link", BUTTON: "button", SECTION: "region", INPUT: "textbox" };
    const role = element.getAttribute("role") ?? implicit[element.tagName] ?? element.tagName.toLowerCase();
    return `${role} ${element.getAttribute("aria-label") ?? element.textContent?.trim() ?? ""}`;
  });
}

/** Checks that target has the focus, where the screen shows it: never on what the screen hides (P5a R35). */
export async function expectLanded(target: Locator): Promise<void> {
  await expect(target).toBeFocused();
  await expect(target).toBeInViewport();
}

/** How a story opens the detail header's menu: by the keyboard or the pointer, or on a narrow screen (narrowOn). */
export type MenuPath = "keyboard" | "pointer" | "narrow";

/** The button of the menu of the header the screen shows, the wide one's or the narrow one's: named Options. */
export function optionsOf(page: Page): Locator {
  return intakeDetail(page).getByRole("button", { name: "Options", exact: true });
}

/** The button that shows or hides the list on a narrow screen, saying whether it shows it (aria-expanded). */
export function listToggle(page: Page): Locator {
  return page.getByRole("button", { name: "Intake list", exact: true });
}

/**
 * Chooses item from the menu of the header the screen shows along path: by the keyboard, Enter on its button, then the
 * arrows and Enter (chooseByKeyboard); else by the pointer, its button, then the item, whose press focuses the item,
 * never the menu's button (P4 O4).
 */
export async function fromMenu(page: Page, path: MenuPath, item: string): Promise<void> {
  const options = optionsOf(page);
  if (path === "keyboard") {
    await options.focus();
    await chooseByKeyboard(page, options, item);
    return;
  }
  await options.click();
  await page.getByRole("menuitem", { name: item, exact: true }).click();
}

/**
 * Makes page's screen narrow and selects issue there as a person does: the list shown by its toggle if a row's press
 * hid it, the toggle saying so, then issue's row pressed, which selects it and hides the list over the detail again.
 * The narrow header's menu is then the one the screen shows; the wide header's identifier is not shown.
 */
export async function narrowOn(page: Page, issue: Issue): Promise<void> {
  await page.setViewportSize(NARROW);
  const toggle = listToggle(page);
  if ((await toggle.getAttribute("aria-expanded")) === "false") await toggle.click();
  await expect(toggle).toHaveAttribute("aria-expanded", "true");
  await intakeRow(page, issue).click();
  await expect(toggle).toHaveAttribute("aria-expanded", "false");
  await expect(page).toHaveURL(new RegExp(`[?&]inboxIssueId=${issue.id}(&|$)`));
}

/** Presses key (tabKey) on page until target has the focus, at most presses times: target, reached by the keyboard. */
export async function tabTo(page: Page, key: string, target: Locator, presses = 8): Promise<void> {
  await page.keyboard.press(key);
  const reached = await target.evaluate((element) => element === document.activeElement);
  if (reached) return;
  expect(presses, `${key} reaches ${target} within the presses given`).toBeGreaterThan(1);
  return tabTo(page, key, target, presses - 1);
}
