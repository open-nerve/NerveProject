/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { useLocation, useNavigate, type Location, type NavigateFunction } from "react-router";
import { vi } from "vitest";

// What the tests of a cycle's or a module's quick actions do on its page, as a person would (fake-cycle-pages.tsx,
// fake-module-pages.tsx): find its row (data-row), open its menu by its button and choose an item, read what the menu
// offers and the dialog it opens, let the page's timers run, and see where the router's address is.

/** Lets the pages' timers run: a menu's or a dialog's transition, and what follows a change. */
export const settle = () => act(async () => vi.advanceTimersByTimeAsync(100));

/** The row of the cycle or the module of id, if its page shows it. */
export const row = (id: string) => document.querySelector<HTMLElement>(`[data-row="${id}"]`);

/** The button of the row's menu. */
export const menuButton = (id: string) =>
  row(id)?.querySelector<HTMLButtonElement>('button[aria-haspopup="menu"]') ?? null;

/** An item of a menu as a person reads it: its title, why it is disabled where it says so, and whether it is. */
export const offered = (items: HTMLButtonElement[]) =>
  items.map((item) => {
    const why = item.querySelector("p")?.textContent;
    return `${item.querySelector("h5")?.textContent}${why ? `: ${why}` : ""}${item.disabled ? " (disabled)" : ""}`;
  });

/** Opens the row's menu by its button: its items. */
export async function openMenu(id: string) {
  const button = menuButton(id);
  await act(async () => {
    button?.focus();
    button?.click();
  });
  await settle();
  const menu = document.getElementById(button?.getAttribute("aria-controls") ?? "");
  return [...(menu?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]') ?? [])];
}

/** Chooses the item of title in the row's menu. */
export async function choose(id: string, title: string) {
  const item = (await openMenu(id)).find((button) => button.querySelector("h5")?.textContent === title);
  if (!item) throw new Error(`The menu of ${id} offers no ${title}`);
  await act(async () => item.click());
  await settle();
}

/** The dialog open on the page, if one is. */
export const dialog = () => document.querySelector<HTMLElement>('[role="dialog"]');

/** The dialog's button whose text is text. */
export const dialogButton = (text: string) =>
  [...(dialog()?.querySelectorAll("button") ?? [])].find((button) => button.textContent?.trim() === text);

/** Where the pages are: the router's address, and its navigate, which moves it as a Back or a link would. */
export const where: { location?: Location; navigate?: NavigateFunction } = {};

/** Keeps where up to date, under the pages' router. */
export function Where() {
  where.location = useLocation();
  where.navigate = useNavigate();
  return null;
}
