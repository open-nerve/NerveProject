/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { toasts } from "@/lib/fake-toast";
import type { TNavigationItem } from "./tab-navigation-root";
import { useProjectActions } from "./use-project-actions";

// What the project header's "Copy link" copies (M3 design 7.7): the link of the tab open, else the project's work
// items' (useCopyProjectLink), each said as a project's link. The hook runs in a component rendered on the server,
// with stand-ins for the clipboard's write and for the address's workspace, acme.

const browser = vi.hoisted(() => ({ copyUrlToClipboard: vi.fn() }));
vi.mock("@nerve/utils", () => ({ copyUrlToClipboard: browser.copyUrlToClipboard }));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "acme" }) }));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** web's cycles tab, open. */
const cycles: TNavigationItem = {
  name: "Cycles",
  href: "/acme/projects/p-web/cycles",
  icon: () => null,
  access: [],
  shouldRender: true,
  sortOrder: 2,
  i18n_key: "sidebar.cycles",
  key: "cycles",
};

/** Copies web's link from its header, with activeItem the tab open (none: no tab of web's is). */
function copiedFrom(activeItem?: TNavigationItem) {
  const header: { copy?: () => Promise<void> } = {};
  function Header() {
    header.copy = useProjectActions({ projectId: "p-web", activeItem }).handleCopyText;
    return null;
  }
  renderToStaticMarkup(<Header />);
  if (!header.copy) throw new Error("the header gave no copy");
  return header.copy();
}

beforeEach(() => {
  browser.copyUrlToClipboard.mockReset();
  browser.copyUrlToClipboard.mockResolvedValue(undefined);
  toasts.length = 0;
});

describe("the project header's copy of the link", () => {
  it.each([
    { open: "a tab", activeItem: cycles, link: "/acme/projects/p-web/cycles" },
    { open: "no tab", activeItem: undefined, link: "/acme/projects/p-web/issues" },
  ])("copies, with $open open, its link, and says a project's link was copied", async ({ activeItem, link }) => {
    await copiedFrom(activeItem);
    expect(browser.copyUrlToClipboard.mock.calls).toEqual([[link]]);
    expect(toasts).toEqual([
      { type: "success", title: "common.link_copied", message: "project_link_copied_to_clipboard" },
    ]);
  });
});
