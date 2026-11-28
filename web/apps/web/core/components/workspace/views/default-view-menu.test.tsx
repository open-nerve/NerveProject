// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A workspace's default view's quick actions (M7/P6b spec 2.8, 2.10, R12; re-97 I4): its menu, by a button named by the
// view, opens the view's page in a new tab, which gets no hold of this one, and copies its link, saying so, or saying
// that the browser did not let the page write the clipboard; nothing else (a default view is no view nerve holds: no
// change, no deletion), for anyone of the workspace. The clipboard's write is a stand-in the test lets or refuses.
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { DEFAULT_GLOBAL_VIEWS_LIST } from "@nerve/constants";
import { choose, menuButton, offered, openMenu } from "@/components/common/fake-quick-actions";
import { signedIn } from "@/lib/auth/fake-tab";
import { toasts } from "@/lib/fake-toast";
import { DefaultViewMenu } from "./default-view-menu";

const held = vi.hoisted(() => ({ copy: vi.fn() }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/utils", async (original) => ({
  ...(await original<typeof import("@nerve/utils")>()),
  copyUrlToClipboard: held.copy,
}));

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers();
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  signedIn();
  toasts.length = 0;
  held.copy.mockReset().mockResolvedValue(undefined);
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

const ASSIGNED = DEFAULT_GLOBAL_VIEWS_LIST.find((view) => view.key === "assigned")!;
const LINK = "/ws/workspace-views/assigned";
/** The menu of ws's default view Assigned, in a row of its own. */
const shown = () =>
  act(async () =>
    root.render(
      <div data-row="assigned">
        <DefaultViewMenu workspaceSlug="ws" view={ASSIGNED} />
      </div>
    )
  );

it("offers to open its page in a new tab and to copy its link, nothing else, by a button named by the view", async () => {
  await shown();
  expect(menuButton("assigned")?.getAttribute("aria-label")).toBe("common.item_options default_global_view.assigned");
  expect(offered(await openMenu("assigned"))).toEqual(["open_in_new_tab", "copy_link"]);
});

it("opens the view's page in a new tab, which gets no hold of this one", async () => {
  const opened = vi.spyOn(window, "open").mockReturnValue(null);
  await shown();
  await choose("assigned", "open_in_new_tab");
  expect(opened.mock.calls).toEqual([[LINK, "_blank", "noopener,noreferrer"]]);
});

it("copies the view's link, saying so", async () => {
  await shown();
  await choose("assigned", "copy_link");
  expect(held.copy.mock.calls).toEqual([[LINK]]);
  expect(toasts).toEqual([
    { type: "success", title: "common.link_copied", message: "common.link_copied_to_clipboard" },
  ]);
});

it("says it could not copy the link when the browser does not let the page write the clipboard", async () => {
  held.copy.mockRejectedValueOnce(new DOMException("Write permission denied.", "NotAllowedError"));
  await shown();
  await choose("assigned", "copy_link");
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "something_went_wrong_please_try_again" }]);
});
