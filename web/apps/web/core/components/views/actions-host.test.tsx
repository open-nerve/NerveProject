// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { choose, dialog, dialogButton, row, settle, where } from "@/components/common/fake-quick-actions";
import { type FakeNerve, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { toasts } from "@/lib/fake-toast";
import { its } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import type { ViewStore } from "@/store/views/view.store";
import { ME, loadViews, viewOf, viewRequests, viewsTab } from "@/store/views/fake-views";
import { fill, submit } from "./fake-view-dialog";
import { ViewPages } from "./fake-view-pages";

// The host of a page of a project's views (P6a spec 2.3, 2.4, R17): a view's quick actions open its page's dialogs
// (useViewsPage), the one that changes the view and the one that deletes it, which follows nerve's answer though the
// row has gone first, and only while the page is mounted and the tab in the session it was sent in. The dialogs are
// real (the view's dialog, AlertModalCore); the views a views' store's over a fake nerve that answers when the test
// says; the caller web's admin (fake-permissions.ts).

const held = vi.hoisted(() => ({
  views: undefined as unknown as ViewStore,
  permissions: undefined as IUserPermissionStore | undefined,
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

/** web's admin at the page of web's views at page ("views", or "views/<id>", Bugs' own), which has Bugs and Mine. */
async function at(page: string) {
  const { acme, web, nerve, views, permissions } = await viewsTab(its.admin);
  const bugs = viewOf(web, "Bugs", "u-ann");
  const mine = viewOf(web, "Mine", ME);
  await loadViews(nerve, views, web, [bugs, mine]);
  Object.assign(held, { views, permissions });
  const list = `/${acme.slug}/projects/${web.id}/views`;
  await act(async () =>
    root.render(<ViewPages path={`${list}${page.replace("views", "").replace("<id>", bugs.id)}`} />)
  );
  await settle();
  return { nerve, views, web, bugs, mine, list };
}
const address = () => where.location?.pathname;
const title = () => document.getElementById(dialog()?.getAttribute("aria-labelledby") ?? "")?.textContent;
const press = async (text: string) => {
  await act(async () => dialogButton(text)?.click());
  await settle();
};
const escape = async () => {
  await act(async () =>
    (document.activeElement ?? document.body).dispatchEvent(
      new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true })
    )
  );
  await settle();
};
/** nerve answers the deletion of the view with response. */
const answerDeletion = async (nerve: FakeNerve, view: View, response: Response) => {
  await act(async () => nerve.to(viewRequests.at(view)).at(-1)?.answer(response));
  await settle();
};
/** The deletion's dialog of Bugs, opened by its menu and confirmed: nerve has the deletion. */
const deleting = async (nerve: FakeNerve, bugs: View) => {
  await choose(bugs.id, "delete");
  await press("delete");
  expect(nerve.to(viewRequests.at(bugs)).map(({ method }) => method)).toEqual(["DELETE"]);
};
const deleted = { type: "success", title: "toast.success", message: "project_views.action.delete.success" };

it.each([
  { page: "views", after: "stays on the list" },
  { page: "views/<id>", after: "goes to the project's views" },
])(
  "deletes the view from $page, closing and saying so once nerve has, though its row has gone first; then $after",
  async ({ page }) => {
    const { nerve, bugs, mine, list } = await at(page);
    await choose(bugs.id, "delete");
    expect(title()).toBe("project_views.delete_view.title");
    expect(dialog()?.textContent).toContain("project_views.delete_view.content");
    await press("delete");
    expect(nerve.to(viewRequests.at(bugs)).map(({ method }) => method)).toEqual(["DELETE"]);
    expect([dialog() === null, toasts]).toEqual([false, []]);
    await answerDeletion(nerve, bugs, noContent());
    expect(toasts).toEqual([deleted]);
    expect(dialog()).toBeNull();
    expect(address()).toBe(list);
    expect(row(bugs.id)).toBeNull();
    // the view it was on is gone from the list too, the other there
    expect(row(mine.id)).not.toBeNull();
  }
);

it("lets nothing close the dialog while its deletion is out", async () => {
  const { nerve, bugs } = await at("views");
  await deleting(nerve, bugs);
  await escape();
  expect(dialogButton("cancel")?.disabled).toBe(true);
  expect(dialog()).not.toBeNull();
});

it.each([
  { status: 409, code: "navigation.views_disabled", why: "errors.navigation_views_disabled" },
  { status: 404, code: "navigation.view_not_found", why: "errors.navigation_view_not_found" },
])("keeps the dialog when nerve refuses the deletion ($code), saying why, never success", async (refused) => {
  const { nerve, bugs, list } = await at("views/<id>");
  await deleting(nerve, bugs);
  await answerDeletion(nerve, bugs, problem(refused.status, refused.code));
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: refused.why }]);
  expect(dialog()).not.toBeNull();
  expect(row(bugs.id)).not.toBeNull();
  expect(address()).toBe(`${list}/${bugs.id}`);
});

it.each([
  { settles: "answered", response: noContent },
  { settles: "refused", response: () => problem(409, "navigation.views_disabled") },
])("follows nothing of a deletion nerve has $settles once a Back has left its page", async ({ response }) => {
  const { nerve, views, web, bugs } = await at("views/<id>");
  await deleting(nerve, bugs);
  await act(async () => where.navigate?.("/acme/projects/elsewhere/issues"));
  await settle();
  await answerDeletion(nerve, bugs, response());
  expect(toasts).toEqual([]);
  expect(address()).toBe("/acme/projects/elsewhere/issues");
  // nerve's answer reaches the store alone
  expect(views.getProjectViews(web.id)?.some(({ id }) => id === bugs.id)).toBe(response !== noContent);
});

it.each([
  { settles: "answered", response: noContent },
  { settles: "refused", response: () => problem(409, "navigation.views_disabled") },
])(
  "follows nothing of a deletion nerve has $settles after another tab moved this one to another account",
  async ({ response }) => {
    const { nerve, bugs, list } = await at("views/<id>");
    await deleting(nerve, bugs);
    switchAccount();
    await answerDeletion(nerve, bugs, response());
    expect(toasts).toEqual([]);
    expect(address()).toBe(`${list}/${bugs.id}`);
  }
);

it("opens the page's dialog that changes the view from its row's Edit, its name the view's; the row shows nerve's answer", async () => {
  const { nerve, mine } = await at("views");
  await choose(mine.id, "edit");
  expect([title(), dialog()?.querySelector<HTMLInputElement>('[name="name"]')?.value]).toEqual([
    "view.update.label",
    "Mine",
  ]);
  await fill("name", "Mine v2");
  await submit();
  expect(nerve.to(viewRequests.at(mine)).map(({ method, body }) => [method, body])).toEqual([
    ["PATCH", { name: "Mine v2" }],
  ]);
  await act(async () =>
    nerve
      .to(viewRequests.at(mine))
      .at(-1)
      ?.answer(json(200, { ...mine, name: "Mine v2" }))
  );
  await settle();
  expect(toasts).toEqual([{ type: "success", title: "toast.success", message: "project_views.action.update.success" }]);
  expect(dialog()).toBeNull();
  expect(row(mine.id)?.textContent).toContain("Mine v2");
});
