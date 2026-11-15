// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A command item gives cmdk its value and the keywords the palette's search finds it by (powerKFilter). cmdk takes an
// item's keywords anew only when its value changes: a row renamed while its list is open, its value the same (a cycle
// whose new name the page's read brings, valued by its id), is found by its new name, no longer by its old one (re-77's
// re-review, m1).
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { Command } from "cmdk";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { PowerKModalCommandItem } from "./command-item";
import { powerKFilter } from "./filter";

let host: HTMLDivElement, view: Root;
/** A list of one row, valued c-1 and named name, under the search, filtered as the palette's hosts filter it. */
const render = (name: string, search: string) =>
  act(async () =>
    view.render(
      <Command shouldFilter={search !== ""} filter={powerKFilter}>
        <Command.Input value={search} />
        <Command.List>
          <PowerKModalCommandItem value="c-1" keywords={[name]} label={name} onSelect={() => {}} />
        </Command.List>
      </Command>
    )
  );
const found = () => [...host.querySelectorAll('[role="option"]')].map((option) => option.textContent);
beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  // oxlint-disable-next-line no-extend-native -- jsdom has no scrolling, which cmdk asks of the item it selects.
  HTMLElement.prototype.scrollIntoView = vi.fn();
  host = document.createElement("div");
  document.body.append(host);
  view = createRoot(host);
});
afterEach(async () => {
  await act(async () => view.unmount());
  host.remove();
  Reflect.deleteProperty(HTMLElement.prototype, "scrollIntoView");
  vi.unstubAllGlobals();
});

it("a row renamed while its list is open is found by its new name, no longer by its old one", async () => {
  await render("Sprint", "spr");
  expect(found()).toEqual(["Sprint"]);
  await render("Launch", "lau");
  expect(found(), "the new name").toEqual(["Launch"]);
  await render("Launch", "spr");
  expect(found(), "the old name").toEqual([]);
});
