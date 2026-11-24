// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { choose, menuButton, offered, openMenu, row, settle } from "@/components/common/fake-quick-actions";
import { signedIn } from "@/lib/auth/fake-tab";
import { toasts } from "@/lib/fake-toast";
import { its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import type { ViewStore } from "@/store/views/view.store";
import { ME, loadViews, viewOf, viewsTab } from "@/store/views/fake-views";
import { ViewPages, viewRow } from "./fake-view-pages";

// A view's row and its quick actions in its project's list (P6a spec 2.2, 2.3, R17): what its menu offers the caller,
// by his role in its project (the permission store's own, over acme and its projects as nerve lists them to him,
// fake-permissions.ts) and by whose the view is (nerve's View.UpdatableBy, View.DeletableBy); its one menu, by its
// button and by a right-click on its row; and the items that act at once (a new tab, its link). The views are a views'
// store's over a fake nerve (fake-view-pages.tsx), the clipboard's write a stand-in the test lets or refuses.

const held = vi.hoisted(() => ({
  views: undefined as unknown as ViewStore,
  permissions: undefined as IUserPermissionStore | undefined,
  copy: vi.fn(),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/hooks/store/use-views", () => ({ useViews: () => held.views }));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => held.permissions,
  useUser: () => ({ data: { id: "u-me" } }),
}));
// the members by their account ids: u-ann is Ann
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({ getUserDetails: (id: string) => ({ id, display_name: id.slice(2) }) }),
}));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => null }));
vi.mock("@/components/assets/profile-image", () => ({
  ProfileAvatar: (props: { alt?: string }) => <span data-avatar="">{props.alt}</span>,
}));
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

/**
 * The caller of a role in acme and in web, at web's views list, which shows its view Bugs, owned by him or by Ann (as
 * owned says). Gives Bugs and its link.
 */
async function shown(caller: Caller, owned: boolean) {
  const { acme, web, nerve, views, permissions } = await viewsTab(caller);
  const bugs = viewOf(web, "Bugs", owned ? ME : "u-ann", { description: "Open bugs, the urgent first" });
  await loadViews(nerve, views, web, [bugs]);
  Object.assign(held, { views, permissions });
  await act(async () => root.render(<ViewPages path={`/${acme.slug}/projects/${web.id}/views`} />));
  await settle();
  return { bugs, link: `/${acme.slug}/projects/${web.id}/views/${bugs.id}` };
}

const NEW_TAB = "open_in_new_tab";
const COPY = "copy_link";

it.each([
  // its owner changes it while the project's admin or member; the admins delete any view, a member his own
  { caller: its.admin, owned: true, items: ["edit", NEW_TAB, COPY, "delete"] },
  { caller: its.admin, owned: false, items: [NEW_TAB, COPY, "delete"] },
  { caller: its.memberAdmin, owned: false, items: [NEW_TAB, COPY, "delete"] },
  { caller: its.guestAdmin, owned: false, items: [NEW_TAB, COPY, "delete"] },
  { caller: its.member, owned: true, items: ["edit", NEW_TAB, COPY, "delete"] },
  { caller: its.member, owned: false, items: [NEW_TAB, COPY] },
  // a guest reads the project's views alone: nerve refuses his writes, of a view he owned before he was made one too
  { caller: its.guest, owned: true, items: [NEW_TAB, COPY] },
  { caller: its.guest, owned: false, items: [NEW_TAB, COPY] },
  { caller: its.guestSeesAll, owned: false, items: [NEW_TAB, COPY] },
] as const)(
  "offers $caller.who, of a view he owns ($owned), what nerve lets him do with it: $items",
  async ({ caller, owned, items }) => {
    const { bugs } = await shown(caller, owned);
    expect(offered(await openMenu(bugs.id))).toEqual(items);
  }
);

it("shows the view's name, a link to its page, its description and its owner", async () => {
  const { bugs, link } = await shown(its.member, false);
  expect(row(bugs.id)?.querySelector("a")?.getAttribute("href")).toBe(link);
  expect(row(bugs.id)?.textContent).toContain("Bugs");
  expect(row(bugs.id)?.textContent).toContain("Open bugs, the urgent first");
  expect(row(bugs.id)?.querySelector("[data-avatar]")?.textContent).toBe("ann");
});

it("gives the row one menu, its button named by the view with none inside it, and its right-click menu out of reach until opened", async () => {
  const { bugs } = await shown(its.admin, true);
  expect(row(bugs.id)?.querySelectorAll('button[aria-haspopup="menu"]')).toHaveLength(1);
  const button = menuButton(bugs.id);
  expect(button?.getAttribute("aria-label")).toBe("common.item_options Bugs");
  expect(button?.querySelector("button")).toBeNull();
  const panels = row(bugs.id)?.querySelectorAll<HTMLElement>('[data-context-menu="true"]');
  expect(panels).toHaveLength(1);
  const panel = panels?.[0];
  expect(panel?.closest("[inert]")?.getAttribute("aria-hidden")).toBe("true");
  await act(async () => viewRow(bugs.id)?.dispatchEvent(new MouseEvent("contextmenu", { bubbles: true })));
  await settle();
  expect(panel?.closest("[inert]")).toBeNull();
  // the right-click menu offers what the button's does
  expect(offered([...(panel?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]') ?? [])])).toEqual([
    "edit",
    NEW_TAB,
    COPY,
    "delete",
  ]);
});

it("opens the view in a new tab, which gets no hold of this one", async () => {
  const opened = vi.spyOn(window, "open").mockReturnValue(null);
  const { bugs, link } = await shown(its.guest, false);
  await choose(bugs.id, NEW_TAB);
  expect(opened.mock.calls).toEqual([[link, "_blank", "noopener,noreferrer"]]);
});

it("copies the view's link, saying so", async () => {
  const { bugs, link } = await shown(its.guest, false);
  await choose(bugs.id, COPY);
  expect(held.copy.mock.calls).toEqual([[link]]);
  expect(toasts).toEqual([
    { type: "success", title: "common.link_copied", message: "common.link_copied_to_clipboard" },
  ]);
});

it("says it could not copy the link when the browser does not let the page write the clipboard", async () => {
  held.copy.mockRejectedValueOnce(new DOMException("Write permission denied.", "NotAllowedError"));
  const { bugs } = await shown(its.member, true);
  await choose(bugs.id, COPY);
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "something_went_wrong_please_try_again" }]);
});
