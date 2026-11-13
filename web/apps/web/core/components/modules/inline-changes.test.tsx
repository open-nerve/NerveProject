// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Module, Project } from "@nerve/api-client";
import { renderFormattedPayloadDate } from "@nerve/utils";
import type { DateRangeDropdown } from "@/components/dropdowns/date-range";
import { signedIn } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { moduleRowOf } from "@/store/planning/fake-modules";
import { moduleOf } from "@/store/project/fake-projects";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { listedModules } from "./fake-module-lists";
import { ModuleCardItem } from "./module-card-item";
import { ModuleListItem } from "./module-list-item";

// The changes of a module in its row and its card (P6a spec 2.6, R5, R11, R13): its status and its days, which the
// caller changes there while he writes the project's modules (canWriteModules) and the module is not archived. Each
// change is nerve's to make, as in the module's sidebar (useModuleChange): the control is busy while it is out, says so
// once nerve has made it and shows nerve's value; refused, it says why and shows nerve's value still; the days go as a
// pair. The caller's role is the permission store's own, over acme and its project web (fake-permissions.ts); the
// modules a stand-in store's; the status's select the real one; the days' picker a stand-in that keeps its props.

const held = vi.hoisted(() => ({
  modules: undefined as unknown as ReturnType<typeof import("./fake-module-lists").listedModules>,
  permissions: undefined as IUserPermissionStore | undefined,
  dates: undefined as ComponentProps<typeof DateRangeDropdown> | undefined,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/hooks/store/use-modules", () => ({ useModules: () => held.modules }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => held.permissions }));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => null }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: () => undefined }) }));
vi.mock("@/components/assets/profile-image", () => ({ ProfileAvatar: () => null }));
vi.mock("@/components/dropdowns/date-range", () => ({
  DateRangeDropdown: (props: ComponentProps<typeof DateRangeDropdown>) => {
    held.dates = props;
    return null;
  },
}));
vi.mock("./quick-actions", () => ({ ModuleQuickActions: () => null }));
vi.mock("./progress-sidebar", () => ({ ModuleDetailsSidebar: () => null }));
vi.mock("@/hooks/use-planning-reads", () => ({ usePlanningReads: () => ({ readable: true, listed: true }) }));

const KINDS = [
  { kind: "row", Item: ModuleListItem },
  { kind: "card", Item: ModuleCardItem },
] as const;
const ARCHIVED = { status: "completed", archived_at: "2026-11-02T09:00:00Z" } as const;
/** The module Auth's fields: in progress, from the 16th to the 22nd of November. */
const AUTH = { status: "in-progress", start_date: "2026-11-16", target_date: "2026-11-22" } satisfies Partial<Module>;

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
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
  held.dates = undefined;
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

/** The caller of a role in acme and in web (of project), at web's module Auth (of fields) as its kind shows it. */
async function shown(
  caller: Caller,
  Item: (typeof KINDS)[number]["Item"],
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
  const auth = moduleOf(web, "Auth", { ...AUTH, ...fields });
  held.modules = listedModules();
  held.modules.show(auth.archived_at === null ? "modules" : "archive", web.id, [moduleRowOf(auth)]);
  await act(async () =>
    root.render(
      <MemoryRouter initialEntries={[`/acme/projects/${web.id}/modules`]}>
        <Item moduleId={auth.id} workspaceSlug="acme" projectId={web.id} />
      </MemoryRouter>
    )
  );
  return auth;
}

const settle = () => act(async () => vi.advanceTimersByTimeAsync(100));
const answer = async (module: Module) => {
  await act(async () => held.modules.sent.at(-1)?.answer(module));
  await settle();
};
const refuse = async (error: unknown) => {
  await act(async () => held.modules.sent.at(-1)?.refuse(error));
  await settle();
};
const success = {
  type: "success",
  title: "project_module.action.update.success.title",
  message: "project_module.action.update.success.description",
};
const statusSelect = () => host.querySelector<HTMLButtonElement>('button[aria-haspopup="listbox"]');
async function pickStatus(label: string) {
  const select = statusSelect();
  await act(async () => {
    select?.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true }));
    select?.click();
  });
  await settle();
  const option = [...document.querySelectorAll<HTMLElement>('[role="option"]')].find(
    (item) => item.textContent === label
  );
  await act(async () => option?.click());
  await settle();
}
const november = (day: number) => new Date(2026, 10, day);
const pick = (from?: Date, to?: Date) => act(async () => held.dates?.onSelect?.(from && { from, to }));
const shownDays = () => {
  const { from, to } = held.dates?.value ?? {};
  return `${renderFormattedPayloadDate(from) ?? "-"} → ${renderFormattedPayloadDate(to) ?? "-"}`;
};
const enabled = () => ({ status: statusSelect()?.disabled === false, days: held.dates?.disabled === false });
const both = (on: boolean) => ({ status: on, days: on });

const callers = [
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
    state: " while modules are turned off",
    fields: {},
    project: { module_view: false },
    editable: false,
  },
];
const changes = [
  {
    control: "status",
    change: () => pickStatus("project_modules.status.paused"),
    shows: () => statusSelect()?.textContent,
    nerve: "project_modules.status.in_progress",
  },
  {
    control: "days",
    change: () => pick(november(17), november(25)),
    shows: shownDays,
    nerve: "2026-11-16 → 2026-11-22",
  },
];

describe.each(KINDS)("in its $kind", ({ Item }) => {
  it.each(callers)("lets $who change a module$state: $editable", async ({ fields, project, editable, ...caller }) => {
    await shown(caller, Item, fields, project);
    expect(enabled()).toEqual(both(editable));
  });

  it("changes its status once nerve has: busy while out, then nerve's", async () => {
    const auth = await shown(its.member, Item);
    await pickStatus("project_modules.status.paused");
    expect(held.modules.update).toHaveBeenCalledExactlyOnceWith(auth.id, { status: "paused" });
    expect(statusSelect()?.textContent).toBe("project_modules.status.in_progress");
    expect(enabled()).toEqual({ status: false, days: true });
    await settle();
    expect(toasts).toEqual([]);
    await answer({ ...auth, status: "paused" });
    expect(toasts).toEqual([success]);
    expect(statusSelect()?.textContent).toBe("project_modules.status.paused");
    expect(enabled()).toEqual(both(true));
  });

  it("changes its days as a pair, once both are picked and nerve has", async () => {
    const auth = await shown(its.member, Item);
    await pick(november(17));
    expect(held.modules.update).not.toHaveBeenCalled();
    await pick(november(17), november(25));
    expect(held.modules.update).toHaveBeenCalledExactlyOnceWith(auth.id, {
      start_date: "2026-11-17",
      target_date: "2026-11-25",
    });
    expect(enabled()).toEqual({ status: true, days: false });
    await settle();
    expect(toasts).toEqual([]);
    await answer({ ...auth, start_date: "2026-11-17", target_date: "2026-11-25" });
    expect(toasts).toEqual([success]);
    expect(shownDays()).toBe("2026-11-17 → 2026-11-25");
    expect(enabled()).toEqual(both(true));
  });

  it.each(changes)(
    "says nerve's refusal of a change of its $control, never success, and shows nerve's",
    async ({ change, shows, nerve }) => {
      await shown(its.member, Item);
      await change();
      await refuse(refusal(409, "planning.archived"));
      expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.planning_archived" }]);
      expect(shows()).toEqual(nerve);
      expect(enabled()).toEqual(both(true));
    }
  );

  it.each(["is answered", "is refused"])("says nothing of a change that %s once the list has gone", async (settles) => {
    const auth = await shown(its.member, Item);
    await pickStatus("project_modules.status.paused");
    await act(async () => root.render(<p>elsewhere</p>));
    if (settles === "is answered") await answer({ ...auth, status: "paused" });
    else await refuse(new Error("refused"));
    expect(toasts).toEqual([]);
  });
});
