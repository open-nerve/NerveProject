// @vitest-environment jsdom
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FiltersDropdown } from "./dropdown";

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

// The button of a filters' popover given no button of its own (M3 closeout spec 2.9): one button, Headless UI's
// Popover.Button itself, named by the title whatever it shows (a narrow header shows the mini icon alone), and saying
// whether the panel is open; closed, the panel is out of the page. The first case preserves the server-rendered
// contract; the DOM cases below exercise both trigger variants, opening, closing and focus return.

describe("FiltersDropdown's own button", () => {
  it("is the popover's one button, named by the title beside a mini icon, saying the panel is closed and out of the page", () => {
    const markup = renderToStaticMarkup(
      <FiltersDropdown title="Display" miniIcon={<svg data-mini-icon="" />}>
        <p>the panel</p>
      </FiltersDropdown>
    );
    const buttons = markup.match(/<button[^>]*>/g) ?? [];
    expect(buttons).toHaveLength(1);
    expect([buttons[0]?.includes('aria-label="Display"'), buttons[0]?.includes('aria-expanded="false"')]).toEqual([
      true,
      true,
    ]);
    expect([markup.includes("data-mini-icon"), markup.includes("the panel")]).toEqual([true, false]);
  });
});

describe("FiltersDropdown's actual button and panel", () => {
  let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
  beforeEach(() => {
    globalThis.IS_REACT_ACT_ENVIRONMENT = true;
    vi.useFakeTimers();
    vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
    host = document.createElement("div");
    document.body.append(host);
    root = createRoot(host);
  });
  afterEach(async () => {
    await act(async () => root.unmount());
    host.remove();
    vi.restoreAllMocks();
    vi.useRealTimers();
  });
  for (const custom of [false, true]) {
    it(`${custom ? "custom" : "default"} names one trigger, locates the real panel, and returns focus after Escape`, async () => {
      await act(async () =>
        root.render(
          <FiltersDropdown title="Display" placement="bottom-end" menuButton={custom ? <svg /> : undefined}>
            <button>Panel option</button>
          </FiltersDropdown>
        )
      );
      const button = host.querySelector<HTMLButtonElement>("button")!;
      expect(host.querySelectorAll("button")).toHaveLength(1);
      expect(button.getAttribute("aria-label")).toBe("Display");
      expect(button.getAttribute("aria-expanded")).toBe("false");
      await act(async () => {
        button.focus();
        button.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }));
        await vi.advanceTimersByTimeAsync(100);
      });
      expect(button.getAttribute("aria-expanded")).toBe("true");
      const panel = document.getElementById(button.getAttribute("aria-controls")!)!;
      expect(panel.style.position).toBe("fixed");
      expect(panel.getAttribute("data-popper-placement")).toBe("bottom-end");
      await act(async () => {
        panel.querySelector<HTMLButtonElement>("button")!.focus();
        panel.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));
        await vi.advanceTimersByTimeAsync(400);
      });
      expect(button.getAttribute("aria-expanded")).toBe("false");
      expect(document.activeElement).toBe(button);
      expect(host.textContent).not.toContain("Panel option");
    });
    it(`${custom ? "custom" : "default"} honors disabled and tabIndex`, async () => {
      await act(async () =>
        root.render(
          <FiltersDropdown title="Display" disabled tabIndex={-1} menuButton={custom ? <svg /> : undefined}>
            <p>Panel</p>
          </FiltersDropdown>
        )
      );
      const button = host.querySelector<HTMLButtonElement>("button")!;
      expect(button.disabled).toBe(true);
      expect(button.tabIndex).toBe(-1);
      await act(async () => button.click());
      expect(button.getAttribute("aria-expanded")).toBe("false");
      expect(host.textContent).not.toContain("Panel");
    });
  }
});
