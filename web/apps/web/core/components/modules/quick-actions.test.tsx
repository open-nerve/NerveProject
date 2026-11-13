// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { ModuleStatus, Project } from "@nerve/api-client";
import { choose, menuButton, offered, openMenu, row, settle } from "@/components/common/fake-quick-actions";
import { signedIn } from "@/lib/auth/fake-tab";
import { toasts } from "@/lib/fake-toast";
import { moduleOf } from "@/store/project/fake-projects";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { ModulePages, standInModules } from "./fake-module-pages";

// A module's quick actions (P6a spec 2.6, R5): what its menu offers its caller, by his role in its project (the
// permission store's own, over acme and its project web as nerve lists them to him, fake-permissions.ts), by the
// project's modules being turned on and by the module's status and archive; its button and its right-click menu; and
// the items that act at once (a new tab, its link). The pages and their modules are fake-module-pages.tsx's, the
// clipboard's write a stand-in the test lets or refuses; the dialog that changes a module a stand-in that keeps what
// its page gave it.

const held = vi.hoisted(() => ({
  modules: undefined as unknown as ReturnType<typeof standInModules>,
  permissions: undefined as IUserPermissionStore | undefined,
  copy: vi.fn(),
  edit: undefined as { isOpen: boolean; moduleId?: string; projectId: string } | undefined,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/hooks/store/use-modules", () => ({ useModules: () => held.modules }));
// the caller is "me"
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => held.permissions,
  useUser: () => ({ data: { id: "me" } }),
}));
vi.mock("@nerve/utils", async (original) => ({
  ...(await original<typeof import("@nerve/utils")>()),
  copyUrlToClipboard: held.copy,
}));
vi.mock("./modal", () => ({
  CreateUpdateModuleModal: (props: NonNullable<typeof held.edit>) => {
    held.edit = props;
    return null;
  },
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
  held.edit = undefined;
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

const ARCHIVED = "2026-11-01T09:00:00Z";

/**
 * The caller of a role in acme and in web (of fields), at the page of web's modules that shows its module Auth, of
 * status, archived or not, created by him or by another: an archived module in the archive, another in the list.
 * Gives Auth and its link.
 */
async function shown(
  caller: Caller,
  status: ModuleStatus,
  {
    archived = false,
    created = false,
    project = {},
  }: { archived?: boolean; created?: boolean; project?: Partial<Project> }
) {
  const { permissions, acme, web } = await callerInWeb(
    caller.workspaceRole,
    caller.projectRole,
    caller.guestViewAll,
    project
  );
  held.permissions = permissions;
  const auth = moduleOf(web, "Auth", {
    status,
    archived_at: archived ? ARCHIVED : null,
    created_by_id: created ? "me" : "another",
  });
  held.modules = standInModules([auth]);
  const page = archived ? "archives/modules" : "modules";
  await act(async () => root.render(<ModulePages path={`/${acme.slug}/projects/${web.id}/${page}`} />));
  await settle();
  return { auth, link: `/${acme.slug}/projects/${web.id}/modules/${auth.id}` };
}

const NEW_TAB = "open_in_new_tab";
const COPY = "copy_link";
const NOT_CLOSED = "archive: project_module.archive.only_closed (disabled)";
const creator = (caller: Caller) => ({ ...caller, who: `${caller.who} who created it`, created: true });
const other = (caller: Caller) => ({ ...caller, created: false });
/** Where the module is: in the list or in the archive, its project's modules turned on or off. */
const states = {
  listed: {},
  archived: { archived: true },
  "listed, its project's modules turned off": { project: { module_view: false } },
  "archived, its project's modules turned off": { archived: true, project: { module_view: false } },
} satisfies Record<string, { archived?: boolean; project?: Partial<Project> }>;

it.each([
  // a module not archived, of each status: archived once completed or cancelled alone
  { ...other(its.admin), status: "backlog", state: "listed", items: ["edit", NEW_TAB, COPY, NOT_CLOSED, "delete"] },
  { ...other(its.admin), status: "planned", state: "listed", items: ["edit", NEW_TAB, COPY, NOT_CLOSED, "delete"] },
  { ...other(its.admin), status: "in-progress", state: "listed", items: ["edit", NEW_TAB, COPY, NOT_CLOSED, "delete"] },
  { ...other(its.admin), status: "paused", state: "listed", items: ["edit", NEW_TAB, COPY, NOT_CLOSED, "delete"] },
  { ...other(its.admin), status: "completed", state: "listed", items: ["edit", NEW_TAB, COPY, "archive", "delete"] },
  { ...other(its.admin), status: "cancelled", state: "listed", items: ["edit", NEW_TAB, COPY, "archive", "delete"] },
  // its admins delete it, and its creator while the project's admin or member; its other members do not
  { ...creator(its.member), status: "completed", state: "listed", items: ["edit", NEW_TAB, COPY, "archive", "delete"] },
  { ...other(its.member), status: "completed", state: "listed", items: ["edit", NEW_TAB, COPY, "archive"] },
  { ...other(its.member), status: "in-progress", state: "listed", items: ["edit", NEW_TAB, COPY, NOT_CLOSED] },
  {
    ...other(its.memberAdmin),
    status: "planned",
    state: "listed",
    items: ["edit", NEW_TAB, COPY, NOT_CLOSED, "delete"],
  },
  { ...creator(its.guest), status: "completed", state: "listed", items: [NEW_TAB, COPY] },
  { ...other(its.outsider), status: "completed", state: "listed", items: [NEW_TAB, COPY] },
  // archived: restored by one who writes, deleted by none (R5)
  { ...other(its.admin), status: "completed", state: "archived", items: [NEW_TAB, COPY, "restore"] },
  { ...creator(its.member), status: "cancelled", state: "archived", items: [NEW_TAB, COPY, "restore"] },
  { ...other(its.guest), status: "completed", state: "archived", items: [NEW_TAB, COPY] },
  // nerve refuses every planning write of one it shows only his own work items, the workspace's admin too (OwnOnly)
  { ...creator(its.guestAdmin), status: "completed", state: "listed", items: [NEW_TAB, COPY] },
  { ...other(its.guestAdmin), status: "completed", state: "archived", items: [NEW_TAB, COPY] },
  {
    ...other(its.guestAdminSeesAll),
    status: "completed",
    state: "listed",
    items: ["edit", NEW_TAB, COPY, "archive", "delete"],
  },
  { ...other(its.guestAdminSeesAll), status: "completed", state: "archived", items: [NEW_TAB, COPY, "restore"] },
  // a creator deletes his module by his role in the project, not the workspace's
  { ...creator(its.guestSeesAll), status: "completed", state: "listed", items: [NEW_TAB, COPY] },
  // and nerve refuses every write of the project's modules while they are turned off
  {
    ...creator(its.admin),
    status: "completed",
    state: "listed, its project's modules turned off",
    items: [NEW_TAB, COPY],
  },
  {
    ...creator(its.admin),
    status: "completed",
    state: "archived, its project's modules turned off",
    items: [NEW_TAB, COPY],
  },
] as const)(
  "offers $who, of a $status module $state, what nerve lets him do with it: $items",
  async ({ items, status, state, ...caller }) => {
    const { auth } = await shown(caller, status, { ...states[state], created: caller.created });
    expect(offered(await openMenu(auth.id))).toEqual(items);
  }
);

it("names its button by the module, a button with none inside it, and keeps its right-click menu out of reach until opened", async () => {
  const { auth } = await shown(its.admin, "completed", {});
  const button = menuButton(auth.id);
  expect(button?.getAttribute("aria-label")).toBe("common.item_options Auth");
  expect(button?.querySelector("button")).toBeNull();
  const panel = row(auth.id)?.querySelector<HTMLElement>('[data-context-menu="true"]');
  expect(panel?.closest("[inert]")?.getAttribute("aria-hidden")).toBe("true");
  await act(async () => row(auth.id)?.dispatchEvent(new MouseEvent("contextmenu", { bubbles: true })));
  await settle();
  expect(panel?.closest("[inert]")).toBeNull();
  // the right-click menu offers what the button's does
  expect(offered([...(panel?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]') ?? [])])).toEqual([
    "edit",
    NEW_TAB,
    COPY,
    "archive",
    "delete",
  ]);
});

it("opens the module in a new tab, which gets no hold of this one", async () => {
  const opened = vi.spyOn(window, "open").mockReturnValue(null);
  const { auth, link } = await shown(its.guest, "planned", {});
  await choose(auth.id, NEW_TAB);
  expect(opened.mock.calls).toEqual([[link, "_blank", "noopener,noreferrer"]]);
});

it("copies the module's link, saying so", async () => {
  const { auth, link } = await shown(its.guest, "completed", states.archived);
  await choose(auth.id, COPY);
  expect(held.copy.mock.calls).toEqual([[link]]);
  expect(toasts).toEqual([
    { type: "success", title: "common.link_copied", message: "common.link_copied_to_clipboard" },
  ]);
});

it("says it could not copy the link when the browser does not let the page write the clipboard", async () => {
  held.copy.mockRejectedValueOnce(new DOMException("Write permission denied.", "NotAllowedError"));
  const { auth } = await shown(its.member, "planned", {});
  await choose(auth.id, COPY);
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "something_went_wrong_please_try_again" }]);
});

it("opens its page's dialog that changes the module", async () => {
  const { auth } = await shown(its.member, "planned", {});
  expect(held.edit?.isOpen ?? false).toBe(false);
  await choose(auth.id, "edit");
  expect(held.edit).toMatchObject({ isOpen: true, moduleId: auth.id, projectId: auth.project_id });
});
