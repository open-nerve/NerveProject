// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { WorkspaceRole } from "@nerve/api-client";
import { EUserWorkspaceRoles } from "@nerve/types";
import { choose, menuButton, offered, openMenu, row, settle } from "@/components/common/fake-quick-actions";
import { signedIn } from "@/lib/auth/fake-tab";
import { toasts } from "@/lib/fake-toast";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import type { ViewStore } from "@/store/views/view.store";
import { ME, workspaceViewOf, workspaceViewsTab } from "@/store/views/fake-views";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { WorkspaceViewPages } from "./fake-view-pages";

// A workspace's own view's row and its quick actions on the workspace's views list (M7/P6b spec 2.7, 2.11, R15): what
// its menu offers the caller by his role in the workspace (the permission store's, over acme as nerve lists it to him,
// fake-permissions.ts), never by a project's, and by whose the view is (nerve's View.UpdatableBy, View.DeletableBy):
// its owner, the workspace's admin or member, changes and deletes it; the workspace's admins delete any; its guests
// are shown none of its views (workspaceViewsOn). Its links are the workspace's view page's. The views are a views'
// store's over a fake nerve (fake-view-pages.tsx, at the app's routes), the clipboard's write a stand-in the test lets
// or refuses. A project's view's row is as ever (quick-actions.test.tsx).

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
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({ getUserDetails: (id: string) => ({ id, display_name: id.slice(2) }) }),
}));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => null }));
vi.mock("@/components/assets/profile-image", () => ({ ProfileAvatar: () => null }));
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

const { ADMIN, MEMBER, GUEST } = EUserWorkspaceRoles;

/**
 * The caller of role in acme, at acme's views list, which shows its own view Roadmap, owned by him or by Ann (as owned
 * says). Gives Roadmap and its link.
 */
async function shown(role: WorkspaceRole, owned: boolean) {
  const roadmap = workspaceViewOf(workspaceOf("acme"), "Roadmap", owned ? ME : "u-ann");
  const { acme, views, permissions } = await workspaceViewsTab(role, [roadmap]);
  Object.assign(held, { views, permissions });
  const list = `/${acme.slug}/workspace-views`;
  await act(async () => root.render(<WorkspaceViewPages path={list} workspace={acme} listed={[roadmap]} />));
  await settle();
  return { roadmap, link: `${list}/${roadmap.id}` };
}

const NEW_TAB = "open_in_new_tab";
const COPY = "copy_link";

it.each([
  // its owner changes it while the workspace's admin or member; the admins delete any view, a member his own
  { who: "its admin", role: ADMIN, owned: true, items: ["edit", NEW_TAB, COPY, "delete"] },
  { who: "its admin", role: ADMIN, owned: false, items: [NEW_TAB, COPY, "delete"] },
  { who: "its member", role: MEMBER, owned: true, items: ["edit", NEW_TAB, COPY, "delete"] },
  { who: "its member", role: MEMBER, owned: false, items: [NEW_TAB, COPY] },
])(
  "offers $who, of a workspace's view he owns ($owned), by his role in the workspace what nerve lets him do: $items",
  async ({ role, owned, items }) => {
    const { roadmap } = await shown(role, owned);
    expect(offered(await openMenu(roadmap.id))).toEqual(items);
  }
);

it.each([true, false])(
  "shows its guest no workspace's view, so no menu of one, though he owned it before (%s): nerve lets him read none",
  async (owned) => {
    const { roadmap } = await shown(GUEST, owned);
    expect([row(roadmap.id), menuButton(roadmap.id)]).toEqual([null, null]);
  }
);

it("links the row to the workspace's view page, and opens that page in a new tab, which gets no hold of this one", async () => {
  const opened = vi.spyOn(window, "open").mockReturnValue(null);
  const { roadmap, link } = await shown(MEMBER, false);
  expect(row(roadmap.id)?.querySelector("a")?.getAttribute("href")).toBe(link);
  await choose(roadmap.id, NEW_TAB);
  expect(opened.mock.calls).toEqual([[link, "_blank", "noopener,noreferrer"]]);
});

it("copies the workspace's view's link, saying so", async () => {
  const { roadmap, link } = await shown(MEMBER, false);
  await choose(roadmap.id, COPY);
  expect(held.copy.mock.calls).toEqual([[link]]);
  expect(toasts).toEqual([
    { type: "success", title: "common.link_copied", message: "common.link_copied_to_clipboard" },
  ]);
});

it("says it could not copy the link when the browser does not let the page write the clipboard", async () => {
  held.copy.mockRejectedValueOnce(new DOMException("Write permission denied.", "NotAllowedError"));
  const { roadmap } = await shown(ADMIN, true);
  await choose(roadmap.id, COPY);
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "something_went_wrong_please_try_again" }]);
});
