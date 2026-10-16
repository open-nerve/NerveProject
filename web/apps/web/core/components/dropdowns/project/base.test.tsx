/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { ProjectDropdownBase } from "./base";

// Where the project dropdown takes the keyboard (M3 closeout, P11's keyboard list): it renders on the server, closed,
// with the tabIndex its caller gives it. As in every dropdown of dropdowns/, its Tab stop is the DropdownButton in
// it, where the caller's tabIndex puts it (the combobox's own button, round it, Headless UI keeps out of the Tab
// order); its root, a box round the button and the list, is no Tab stop and no control (role presentation).

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

describe("ProjectDropdownBase", () => {
  it("puts its DropdownButton where its caller's tabIndex says, its root no Tab stop", () => {
    const markup = renderToStaticMarkup(
      <ProjectDropdownBase
        buttonVariant="border-with-text"
        getProjectById={() => undefined}
        projectIds={[]}
        multiple={false}
        value={null}
        onChange={() => {}}
        tabIndex={0}
      />
    );
    const tags = markup.match(/<(div|button)[^>]*>/g) ?? [];
    expect(tags.map((tag) => /tabindex="(-?\d+)"/.exec(tag)?.[1] ?? "none")).toEqual(["none", "-1", "0"]);
  });
});
