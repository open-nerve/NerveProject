/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { FiltersDropdown } from "./dropdown";

// The button of a filters' popover given no button of its own (M3 closeout spec 2.9): one button, Headless UI's
// Popover.Button itself, named by the title whatever it shows (a narrow header shows the mini icon alone), and saying
// whether the panel is open; closed, the panel is out of the page. The popover renders on the server, as a page first
// shows it: these tests run in Node, with no page to click in, so opening it is the stories' (S7, by the keyboard).

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
