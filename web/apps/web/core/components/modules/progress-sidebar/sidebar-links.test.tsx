// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Module, ModuleLink, Project } from "@nerve/api-client";
import { calculateTimeAgo } from "@nerve/utils";
import { signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { userOf } from "@/store/member/fake-users";
import { moduleLinkOf } from "@/store/planning/fake-modules";
import { moduleOf } from "@/store/project/fake-projects";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { mount, page, renderSidebar, sidebarModules, unmount } from "./fake-sidebar";

// The links of a module's progress sidebar (P6a spec 2.8, R5, R10, R11): every caller who reads the module sees them,
// a guest too, and copies their addresses; the caller who writes the project's modules (canWriteModules) adds, changes
// and removes them while the module is not archived. A row shows the link's inline icon alone, never its crawl
// source's, and its title, else its page's, else its address; who added it while that account is there. A link is
// removed once nerve has removed it. The caller's role is the permission store's own, over acme and its project web
// (fake-permissions.ts); the modules a stand-in store's, whose changes nerve settles when the test says; the clipboard's
// write a stand-in the test lets or refuses; the rest of the sidebar, and the bounded reads of the links' metadata
// (use-metadata-refresh.test.tsx), none.

const held = vi.hoisted(() => ({
  modules: undefined as unknown as ReturnType<typeof sidebarModules>,
  permissions: undefined as IUserPermissionStore | undefined,
  copy: vi.fn(),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/hooks/store/use-modules", () => ({ useModules: () => held.modules }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => held.permissions }));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({ getUserDetails: (id: string) => (id === ann.id ? ann : undefined) }),
}));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@nerve/utils", async (original) => ({
  ...(await original<typeof import("@nerve/utils")>()),
  copyTextToClipboard: held.copy,
}));
vi.mock("./sidebar-header", () => ({ ModuleSidebarHeader: () => null }));
vi.mock("./sidebar-details", () => ({ ModuleSidebarDetails: () => null }));
vi.mock("./issue-progress", () => ({ ModuleProgress: () => null }));
vi.mock("../links/use-metadata-refresh", () => ({ useModuleLinkMetadataRefresh: () => undefined }));

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers({ now: new Date(2026, 10, 13, 12) });
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
  mount();
});
afterEach(async () => {
  await unmount();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

const ARCHIVED = { status: "completed", archived_at: "2026-11-02T09:00:00Z" } as const;
const ann = userOf("Ann");
const ICON = "data:image/png;base64,iVBORw0KGgo=";

/** The caller of a role in acme and in web (of fields), at the sidebar of web's module Auth (of fields) and its links. */
async function shown(
  caller: Caller,
  links: (auth: Module) => ModuleLink[],
  fields: Partial<Module> = {},
  project: Partial<Project> = {}
) {
  const { permissions, web } = await callerInWeb(
    caller.workspaceRole,
    caller.projectRole,
    caller.guestViewAll,
    project
  );
  held.permissions = permissions;
  const auth = moduleOf(web, "Auth", { status: "in-progress", ...fields });
  const listed = links(auth);
  held.modules = sidebarModules(auth, { links: listed });
  await renderSidebar({ moduleId: auth.id, projectId: web.id, handleClose: () => undefined });
  return { auth, links: listed };
}
/** A link of Auth titled Docs, added by Ann. */
const docsOf = (auth: Module) =>
  moduleLinkOf(auth, "https://docs.example.com", { title: "Docs", created_by_id: ann.id, metadata: { title: "Docs" } });

const settle = () => act(async () => vi.advanceTimersByTimeAsync(100));
const text = () => page.host?.textContent ?? "";
const button = (name: string) =>
  [...(page.host?.querySelectorAll<HTMLButtonElement>("button") ?? [])].find(
    (one) => (one.getAttribute("aria-label") ?? one.textContent?.trim()) === name
  );
const dialog = () => document.querySelector<HTMLElement>('[role="dialog"]');
const dialogTitle = () => document.getElementById(dialog()?.getAttribute("aria-labelledby") ?? "")?.textContent;
const anchors = () =>
  [...(page.host?.querySelectorAll("a") ?? [])].map((a) => ({
    text: a.textContent,
    href: a.getAttribute("href"),
    target: a.getAttribute("target"),
    rel: a.getAttribute("rel"),
  }));
/** What the caller may do with the link Docs: add one, copy it, change it, remove it. */
const offered = () => ({
  add: button("common.add_link") !== undefined,
  copy: button("common.actions.copy_link: Docs") !== undefined,
  edit: button("common.edit_item Docs") !== undefined,
  remove: button("common.delete_item Docs") !== undefined,
});
const click = (name: string) => act(async () => button(name)?.click());
/** nerve answers the change of a link sent last with answer, or refuses it. */
const answer = async (link?: ModuleLink) => {
  await act(async () => held.modules.linkSent.at(-1)?.change.answer(link));
  await settle();
};
const refuse = async (error: unknown) => {
  await act(async () => held.modules.linkSent.at(-1)?.change.refuse(error));
  await settle();
};

it.each([
  { ...its.admin, state: "", fields: {}, project: {}, editable: true },
  { ...its.member, state: "", fields: {}, project: {}, editable: true },
  { ...its.memberAdmin, state: "", fields: {}, project: {}, editable: true },
  { ...its.guestAdminSeesAll, state: "", fields: {}, project: {}, editable: true },
  { ...its.guest, state: "", fields: {}, project: {}, editable: false },
  { ...its.guestAdmin, state: "", fields: {}, project: {}, editable: false },
  { ...its.guestSeesAll, state: "", fields: {}, project: {}, editable: false },
  { ...its.outsider, state: "", fields: {}, project: {}, editable: false },
  { ...its.admin, state: " that is archived", fields: ARCHIVED, project: {}, editable: false },
  { ...its.member, state: " that is archived", fields: ARCHIVED, project: {}, editable: false },
  {
    ...its.admin,
    state: " while the project's modules are turned off",
    fields: {},
    project: { module_view: false },
    editable: false,
  },
])("shows $who the links of a module$state, to copy; to add, change and remove: $editable", async (one) => {
  const { fields, project, editable, ...caller } = one;
  await shown(caller, (auth) => [docsOf(auth)], fields, project);
  expect(anchors().map((a) => a.text)).toEqual(["Docs"]);
  expect(offered()).toEqual({ add: editable, copy: true, edit: editable, remove: editable });
});

it("tells a guest the module has no links yet, offering none to add", async () => {
  await shown(its.guest, () => []);
  await click("common.links");
  expect(text()).toContain("common.no_links_added_yet");
  expect(button("common.add_link")).toBeUndefined();
});

it("names a link by its title, else its page's, else its address; its inline icon alone, never fetched from elsewhere", async () => {
  await shown(its.guest, (auth) => [
    moduleLinkOf(auth, "https://a.example.com", {
      title: "Handbook",
      metadata: { title: "Page A", favicon: ICON, favicon_url: "https://a.example.com/favicon.ico" },
    }),
    moduleLinkOf(auth, "https://b.example.com", {
      metadata: { title: "Page B", favicon: "https://b.example.com/b.ico", favicon_url: "https://b.example.com/x.ico" },
    }),
    moduleLinkOf(auth, "https://c.example.com", { title: "", metadata: { error: "failed" } }),
  ]);
  const opens = { target: "_blank", rel: "noopener noreferrer" };
  expect(anchors()).toEqual([
    { text: "Handbook", href: "https://a.example.com", ...opens },
    { text: "Page B", href: "https://b.example.com", ...opens },
    { text: "https://c.example.com", href: "https://c.example.com", ...opens },
  ]);
  expect([...(page.host?.querySelectorAll("img") ?? [])].map((img) => img.getAttribute("src"))).toEqual([ICON]);
  expect(page.host?.innerHTML).not.toContain("favicon.ico");
  expect(page.host?.innerHTML).not.toContain("b.ico");
});

it("says who added a link while that account is there, and only when otherwise", async () => {
  const at = "2026-11-12T09:00:00Z";
  await shown(its.member, (auth) => [
    moduleLinkOf(auth, "https://a.example.com", { created_by_id: ann.id, created_at: at }),
    moduleLinkOf(auth, "https://b.example.com", { created_by_id: "u-gone", created_at: at }),
    moduleLinkOf(auth, "https://c.example.com", { created_by_id: null, created_at: at }),
  ]);
  const ago = calculateTimeAgo(at);
  expect(text()).toContain(`project_module.link.added_by ${ago} Ann`);
  expect(text().split(`project_module.link.added_at ${ago}`)).toHaveLength(3);
  expect(text()).not.toContain("u-gone");
});

it("copies a link's address as it is, saying so", async () => {
  await shown(its.guest, (auth) => [docsOf(auth)]);
  await click("common.actions.copy_link: Docs");
  expect(held.copy.mock.calls).toEqual([["https://docs.example.com"]]);
  expect(toasts).toEqual([{ type: "success", title: "toast.success", message: "common.link_copied_to_clipboard" }]);
});

it("says it could not copy a link's address when the browser does not let the page write the clipboard", async () => {
  held.copy.mockRejectedValueOnce(new DOMException("Write permission denied.", "NotAllowedError"));
  await shown(its.guest, (auth) => [docsOf(auth)]);
  await click("common.actions.copy_link: Docs");
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "something_went_wrong_please_try_again" }]);
});

it("removes a link once nerve has: the row stays, busy, until then; then it goes and says so", async () => {
  const { links } = await shown(its.member, (auth) => [docsOf(auth)]);
  await click("common.delete_item Docs");
  await click("common.delete_item Docs");
  expect(held.modules.removeLink.mock.calls).toEqual([[links[0]?.id]]);
  expect(button("common.delete_item Docs")?.disabled).toBe(true);
  expect(anchors().map((a) => a.text)).toEqual(["Docs"]);
  expect(toasts).toEqual([]);
  await answer();
  expect(anchors()).toEqual([]);
  expect(toasts).toEqual([
    { type: "success", title: "links.toasts.removed.title", message: "links.toasts.removed.message" },
  ]);
});

it.each([
  ["of an archived module", refusal(409, "planning.archived"), "errors.planning_archived"],
  ["it may not remove", refusal(403, "forbidden"), "errors.forbidden"],
  ["that is gone", refusal(404, "planning.module_link_not_found"), "errors.planning_module_link_not_found"],
  ["the store no longer holds", new Error("Link not found"), "errors.unknown"],
])("keeps a link nerve refuses to remove, %s, saying why", async (_, error, why) => {
  await shown(its.member, (auth) => [docsOf(auth)]);
  await click("common.delete_item Docs");
  await refuse(error);
  expect(anchors().map((a) => a.text)).toEqual(["Docs"]);
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: why }]);
  expect(button("common.delete_item Docs")?.disabled).toBe(false);
});

it.each([
  ["is answered", () => answer()],
  ["is refused", () => refuse(refusal(409, "planning.archived"))],
])("follows nothing of a removal that %s once its sidebar has gone", async (_, settleIt) => {
  await shown(its.member, (auth) => [docsOf(auth)]);
  await click("common.delete_item Docs");
  await act(async () => page.root?.render(null));
  await settleIt();
  expect(toasts).toEqual([]);
});

it("follows nothing of a removal answered after another tab moved this one to another account", async () => {
  await shown(its.member, (auth) => [docsOf(auth)]);
  await click("common.delete_item Docs");
  switchAccount();
  await answer();
  expect(toasts).toEqual([]);
});

it("opens the dialog that adds a link, and the one that changes a link, its address and title in it", async () => {
  const { auth, links } = await shown(its.member, (module) => [docsOf(module)]);
  await click("common.add_link");
  expect(dialogTitle()).toBe("common.add_link");
  expect(document.querySelector<HTMLInputElement>("#url")?.value).toBe("");
  const cancel = [...(dialog()?.querySelectorAll("button") ?? [])].find((one) => one.textContent === "common.cancel");
  await act(async () => cancel?.click());
  await settle();
  expect(dialog()).toBeNull();
  await click("common.edit_item Docs");
  expect(dialogTitle()).toBe("common.update_link");
  expect(document.querySelector<HTMLInputElement>("#url")?.value).toBe("https://docs.example.com");
  expect(document.querySelector<HTMLInputElement>("#title")?.value).toBe("Docs");
  await act(async () =>
    dialog()
      ?.querySelector("form")
      ?.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  );
  await settle();
  expect(held.modules.changeLink.mock.calls).toEqual([
    [links[0]?.id, { url: "https://docs.example.com", title: "Docs" }],
  ]);
  await answer({ ...docsOf(auth), title: "Guide" });
  expect(dialog()).toBeNull();
  expect(anchors().map((a) => a.text)).toEqual(["Guide"]);
});
