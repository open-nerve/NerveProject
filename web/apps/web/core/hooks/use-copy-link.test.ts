/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { toasts } from "@/lib/fake-toast";
import { useCopyLink, useCopyProjectLink } from "./use-copy-link";

// Copying a link (M3 design 7.7): the hooks run as plain functions, with stand-ins for the clipboard's write, which
// the browser allows or refuses as the test says, and for the address's workspace.

const browser = vi.hoisted(() => ({ copyUrlToClipboard: vi.fn() }));
vi.mock("@nerve/utils", () => ({ copyUrlToClipboard: browser.copyUrlToClipboard }));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "acme" }) }));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

beforeEach(() => {
  browser.copyUrlToClipboard.mockReset();
  browser.copyUrlToClipboard.mockResolvedValue(undefined);
  toasts.length = 0;
});

describe("useCopyLink", () => {
  it("writes the link to the clipboard and says what was copied", async () => {
    await useCopyLink()("/acme/projects/p-web/issues", "the copied");
    expect(browser.copyUrlToClipboard.mock.calls).toEqual([["/acme/projects/p-web/issues"]]);
    expect(toasts).toEqual([{ type: "success", title: "common.link_copied", message: "the copied" }]);
  });

  it("says it could not when the browser does not let the page write the clipboard, and does not reject", async () => {
    browser.copyUrlToClipboard.mockRejectedValueOnce(new DOMException("Write permission denied.", "NotAllowedError"));
    await expect(useCopyLink()("/acme/projects/p-web/issues", "the copied")).resolves.toBeUndefined();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "something_went_wrong_please_try_again" }]);
  });
});

describe("useCopyProjectLink", () => {
  it("copies the link of the project's work items in the address's workspace", async () => {
    await useCopyProjectLink()("p-web");
    expect(browser.copyUrlToClipboard.mock.calls).toEqual([["/acme/projects/p-web/issues"]]);
    expect(toasts).toEqual([
      { type: "success", title: "common.link_copied", message: "project_link_copied_to_clipboard" },
    ]);
  });
});
