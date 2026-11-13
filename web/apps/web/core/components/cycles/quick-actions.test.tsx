// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Cycle } from "@nerve/api-client";
import { choose, menuButton, offered, openMenu, row, settle } from "@/components/common/fake-quick-actions";
import { signedIn } from "@/lib/auth/fake-tab";
import { toasts } from "@/lib/fake-toast";
import { cycleOf } from "@/store/project/fake-projects";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { CyclePages, standInCycles } from "./fake-cycle-pages";

// A cycle's quick actions (P5a spec 2.5, R8): what its menu offers its caller, by his role in its project (the
// permission store's own, over acme and its project web as nerve lists them to him, fake-permissions.ts) and by the
// cycle's state; its button and its right-click menu; and the items that act at once (a new tab, its link). The pages
// and their cycles are fake-cycle-pages.tsx's, the clipboard's write a stand-in the test lets or refuses; the dialog
// that changes a cycle is a stand-in that keeps what its page gave it.

const held = vi.hoisted(() => ({
  cycles: undefined as unknown as ReturnType<typeof standInCycles>,
  permissions: undefined as IUserPermissionStore | undefined,
  copy: vi.fn(),
  edit: undefined as { isOpen: boolean; cycleId?: string; projectId: string } | undefined,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/hooks/store/use-cycles", () => ({ useCycles: () => held.cycles }));
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
  CycleCreateUpdateModal: (props: NonNullable<typeof held.edit>) => {
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

const states = {
  upcoming: { status: "upcoming", archived_at: null },
  "in progress": { status: "current", archived_at: null },
  ended: { status: "completed", archived_at: null },
  archived: { status: "completed", archived_at: "2026-11-01T09:00:00Z" },
} satisfies Record<string, Partial<Cycle>>;

/**
 * The caller of a role in acme and in web, at the page of web's cycles that shows its cycle Sprint, of state, created
 * by him or by another: an archived cycle in the archive, another in the list. Gives Sprint.
 */
async function shown(caller: Caller, state: keyof typeof states, created = false) {
  const { permissions, acme, web } = await callerInWeb(caller.workspaceRole, caller.projectRole, caller.guestViewAll);
  held.permissions = permissions;
  const sprint = cycleOf(web, "Sprint", { ...states[state], created_by_id: created ? "me" : "another" });
  held.cycles = standInCycles([sprint]);
  const page = state === "archived" ? "archives/cycles" : "cycles";
  await act(async () => root.render(<CyclePages path={`/${acme.slug}/projects/${web.id}/${page}`} />));
  await settle();
  return { sprint, link: `/${acme.slug}/projects/${web.id}/cycles/${sprint.id}` };
}

const NEW_TAB = "open_in_new_tab";
const COPY = "copy_link";
const NOT_YET = "archive: project_cycles.archive.only_completed (disabled)";
const creator = (caller: Caller) => ({ ...caller, who: `${caller.who} who created it`, created: true });
const other = (caller: Caller) => ({ ...caller, created: false });

it.each([
  { ...other(its.admin), state: "upcoming", items: ["edit", NEW_TAB, COPY, NOT_YET, "delete"] },
  { ...creator(its.member), state: "upcoming", items: ["edit", NEW_TAB, COPY, NOT_YET, "delete"] },
  { ...other(its.member), state: "upcoming", items: ["edit", NEW_TAB, COPY, NOT_YET] },
  { ...other(its.memberAdmin), state: "in progress", items: ["edit", NEW_TAB, COPY, NOT_YET, "delete"] },
  { ...creator(its.guest), state: "upcoming", items: [NEW_TAB, COPY] },
  { ...other(its.outsider), state: "upcoming", items: [NEW_TAB, COPY] },
  { ...creator(its.admin), state: "ended", items: [NEW_TAB, COPY, "archive"] },
  { ...creator(its.member), state: "ended", items: [NEW_TAB, COPY, "archive"] },
  { ...other(its.guest), state: "ended", items: [NEW_TAB, COPY] },
  { ...creator(its.member), state: "archived", items: [NEW_TAB, COPY, "restore"] },
  { ...other(its.guest), state: "archived", items: [NEW_TAB, COPY] },
  // nerve refuses every planning write of one it shows only his own work items, the workspace's admin too (OwnOnly)
  { ...creator(its.guestAdmin), state: "upcoming", items: [NEW_TAB, COPY] },
  { ...other(its.guestAdmin), state: "ended", items: [NEW_TAB, COPY] },
  { ...other(its.guestAdmin), state: "archived", items: [NEW_TAB, COPY] },
  { ...other(its.guestAdminSeesAll), state: "upcoming", items: ["edit", NEW_TAB, COPY, NOT_YET, "delete"] },
  { ...other(its.guestAdminSeesAll), state: "ended", items: [NEW_TAB, COPY, "archive"] },
  { ...other(its.guestAdminSeesAll), state: "archived", items: [NEW_TAB, COPY, "restore"] },
  // a creator deletes his cycle by his role in the project, not the workspace's
  { ...creator(its.guestSeesAll), state: "upcoming", items: [NEW_TAB, COPY] },
] as const)("offers $who, of a cycle $state, what nerve lets him do with it: $items", async (caller) => {
  const { sprint } = await shown(caller, caller.state, caller.created);
  expect(offered(await openMenu(sprint.id))).toEqual(caller.items);
});

it("names its button by the cycle, a button with none inside it, and keeps its right-click menu out of reach until opened", async () => {
  const { sprint } = await shown(its.admin, "ended");
  const button = menuButton(sprint.id);
  expect(button?.getAttribute("aria-label")).toBe("common.item_options Sprint");
  expect(button?.querySelector("button")).toBeNull();
  const panel = row(sprint.id)?.querySelector<HTMLElement>('[data-context-menu="true"]');
  expect(panel?.closest("[inert]")?.getAttribute("aria-hidden")).toBe("true");
  await act(async () => row(sprint.id)?.dispatchEvent(new MouseEvent("contextmenu", { bubbles: true })));
  await settle();
  expect(panel?.closest("[inert]")).toBeNull();
  // the right-click menu offers what the button's does
  expect(offered([...(panel?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]') ?? [])])).toEqual([
    NEW_TAB,
    COPY,
    "archive",
  ]);
});

it("opens the cycle in a new tab, which gets no hold of this one", async () => {
  const opened = vi.spyOn(window, "open").mockReturnValue(null);
  const { sprint, link } = await shown(its.guest, "upcoming");
  await choose(sprint.id, NEW_TAB);
  expect(opened.mock.calls).toEqual([[link, "_blank", "noopener,noreferrer"]]);
});

it("copies the cycle's link, saying so", async () => {
  const { sprint, link } = await shown(its.guest, "archived");
  await choose(sprint.id, COPY);
  expect(held.copy.mock.calls).toEqual([[link]]);
  expect(toasts).toEqual([
    { type: "success", title: "common.link_copied", message: "common.link_copied_to_clipboard" },
  ]);
});

it("says it could not copy the link when the browser does not let the page write the clipboard", async () => {
  held.copy.mockRejectedValueOnce(new DOMException("Write permission denied.", "NotAllowedError"));
  const { sprint } = await shown(its.member, "upcoming");
  await choose(sprint.id, COPY);
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "something_went_wrong_please_try_again" }]);
});

it("opens its page's dialog that changes the cycle", async () => {
  const { sprint } = await shown(its.member, "upcoming");
  expect(held.edit?.isOpen ?? false).toBe(false);
  await choose(sprint.id, "edit");
  expect(held.edit).toMatchObject({ isOpen: true, cycleId: sprint.id, projectId: sprint.project_id });
});
