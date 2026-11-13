// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Module, Project } from "@nerve/api-client";
import { renderFormattedPayloadDate } from "@nerve/utils";
import type { DateRangeDropdown } from "@/components/dropdowns/date-range";
import type { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { lateSettlingsAnswering, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { moduleOf, projectOf } from "@/store/project/fake-projects";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { mount, page, renderSidebar, sidebarModules, unmount } from "./fake-sidebar";

// The changes of a module in its progress sidebar (P6a spec 2.6, R5, R6, R11, R13): its status, its days, its lead and
// its members, which the caller changes there while he writes the project's modules (canWriteModules) and the module is
// not archived. Each change is nerve's to make: the control is busy while it is out, says so once nerve has made it and
// shows nerve's value; refused, it says why and shows nerve's value still. A click of the members' list adds or removes
// one member, whom the store adds to or removes from nerve's set in its turn; the days go as a pair, once both are
// picked or both cleared, and not when they are the module's. The caller's role is the permission store's own, over
// acme and its project web (fake-permissions.ts); the modules a stand-in store's; the status's select the real one;
// the pickers of the days and the members stand-ins that keep their props, so that a test picks as their lists would.

type Picker = ComponentProps<typeof MemberDropdown>;
const held = vi.hoisted(() => ({
  modules: undefined as unknown as ReturnType<typeof sidebarModules>,
  permissions: undefined as IUserPermissionStore | undefined,
  /** The props the days' picker was last given. */
  dates: undefined as ComponentProps<typeof DateRangeDropdown> | undefined,
  /** The props each members' picker was last given: the lead's, the members'. */
  pickers: {} as Record<"lead" | "members", Picker | undefined>,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/hooks/store/use-modules", () => ({ useModules: () => held.modules }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => held.permissions }));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({ project: { getProjectMemberDetails: () => null } }),
}));
vi.mock("@/hooks/store/use-label", () => ({ useLabel: () => ({ getProjectLabels: () => [] }) }));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => null }));
vi.mock("@/components/assets/profile-image", () => ({ ProfileAvatar: () => null }));
vi.mock("@/components/dropdowns/date-range", () => ({
  DateRangeDropdown: (props: ComponentProps<typeof DateRangeDropdown>) => {
    held.dates = props;
    return null;
  },
}));
vi.mock("@/components/dropdowns/member/dropdown", () => ({
  MemberDropdown: (props: Picker) => {
    held.pickers[props.multiple ? "members" : "lead"] = props;
    return null;
  },
}));
vi.mock("./sidebar-links", () => ({ ModuleSidebarLinks: () => null }));

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
  held.pickers = { lead: undefined, members: undefined };
  mount();
});
afterEach(async () => {
  await unmount();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

const ARCHIVED = { status: "completed", archived_at: "2026-11-02T09:00:00Z" } as const;
const ann = "u-ann";
const bob = "u-bob";
/** The module Auth's fields: in progress, from the 16th to the 22nd of November, led by Ann, of Ann alone. */
const AUTH = {
  status: "in-progress",
  start_date: "2026-11-16",
  target_date: "2026-11-22",
  lead_id: ann,
  member_ids: [ann],
} satisfies Partial<Module>;

/** The caller of a role in acme and in web (of fields), at the sidebar of web's module Auth (of fields): Auth. */
async function shown(caller: Caller, fields: Partial<Module> = {}, project: Partial<Project> = {}) {
  const { permissions, web } = await callerInWeb(
    caller.workspaceRole,
    caller.projectRole,
    caller.guestViewAll,
    project
  );
  held.permissions = permissions;
  const auth = moduleOf(web, "Auth", { ...AUTH, ...fields });
  held.modules = sidebarModules(auth);
  await renderSidebar({ moduleId: auth.id, projectId: web.id, handleClose: () => undefined });
  return auth;
}

const settle = () => act(async () => vi.advanceTimersByTimeAsync(100));
/** nerve answers the change sent last with module. */
const answer = async (module: Module) => {
  await act(async () => held.modules.sent.at(-1)?.change.answer(module));
  await settle();
};
/** nerve refuses the change sent last. */
const refuse = async (error: unknown) => {
  await act(async () => held.modules.sent.at(-1)?.change.refuse(error));
  await settle();
};
const success = {
  type: "success",
  title: "project_module.action.update.success.title",
  message: "project_module.action.update.success.description",
};

const statusSelect = () => page.host?.querySelector<HTMLButtonElement>('button[aria-haspopup="listbox"]');
/** The status's select: opens its list and picks the status of its label. */
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
/** A day of November 2026, as the calendar gives it: the local midnight of the day. */
const november = (day: number) => new Date(2026, 10, day);
/** The person picks a range in the days' calendar (none: he unpicks the one day picked). */
const pick = (from?: Date, to?: Date) =>
  act(async () => held.dates?.onSelect?.(from === undefined && to === undefined ? undefined : { from, to }));
/** The days the picker shows, as nerve writes them ("-" for none). */
const shownDays = () => {
  const { from, to } = held.dates?.value ?? {};
  return `${renderFormattedPayloadDate(from) ?? "-"} → ${renderFormattedPayloadDate(to) ?? "-"}`;
};
/** The person picks in the members' picker of the lead or of the members, as its list would. */
const choose = (picker: "lead" | "members", value: string | string[] | null) =>
  act(async () => (held.pickers[picker]?.onChange as ((picked: typeof value) => void) | undefined)?.(value));
/** Whether each control lets the caller change the module: its status, its days, its lead, its members. */
const enabled = () => ({
  status: statusSelect()?.disabled === false,
  days: held.dates?.disabled === false,
  lead: held.pickers.lead?.disabled === false,
  members: held.pickers.members?.disabled === false,
});
const all = (on: boolean) => ({ status: on, days: on, lead: on, members: on });

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
])("lets $who change a module$state: $editable", async ({ fields, project, editable, ...caller }) => {
  await shown(caller, fields, project);
  expect(enabled()).toEqual(all(editable));
});

it("changes its status once nerve has: busy while out, no success before, then nerve's status", async () => {
  const auth = await shown(its.member);
  await pickStatus("project_modules.status.paused");
  expect(held.modules.update).toHaveBeenCalledExactlyOnceWith(auth.id, { status: "paused" });
  expect(statusSelect()?.textContent).toBe("project_modules.status.in_progress");
  expect(enabled()).toEqual({ ...all(true), status: false });
  await settle();
  expect(toasts).toEqual([]);
  await answer({ ...auth, status: "paused" });
  expect(toasts).toEqual([success]);
  expect(statusSelect()?.textContent).toBe("project_modules.status.paused");
  expect(enabled()).toEqual(all(true));
});

it("changes its lead once nerve has, and sends nothing for the lead it has", async () => {
  const auth = await shown(its.member);
  await choose("lead", ann);
  expect(held.modules.update).not.toHaveBeenCalled();
  await choose("lead", bob);
  expect(held.modules.update).toHaveBeenCalledExactlyOnceWith(auth.id, { lead_id: bob });
  expect(held.pickers.lead?.value).toBe(ann);
  expect(enabled()).toEqual({ ...all(true), lead: false });
  await settle();
  expect(toasts).toEqual([]);
  await answer({ ...auth, lead_id: bob });
  expect(toasts).toEqual([success]);
  expect(held.pickers.lead?.value).toBe(bob);
  expect(enabled()).toEqual(all(true));
});

it("adds or removes one member a click, whom the store adds to nerve's set or removes from it in its turn (R6)", async () => {
  const auth = await shown(its.member);
  await choose("members", [ann, bob]);
  expect(held.modules.changeMembers).toHaveBeenCalledExactlyOnceWith(auth.id, { add: [bob], remove: [] });
  expect(held.pickers.members?.value).toEqual([ann]);
  expect(enabled()).toEqual({ ...all(true), members: false });
  await settle();
  expect(toasts).toEqual([]);
  await answer({ ...auth, member_ids: [ann, bob] });
  expect(toasts).toEqual([success]);
  expect(held.pickers.members?.value).toEqual([ann, bob]);
  await choose("members", [bob]);
  expect(held.modules.changeMembers).toHaveBeenLastCalledWith(auth.id, { add: [], remove: [ann] });
  expect(held.modules.update).not.toHaveBeenCalled();
});

it("changes its days once both are picked, as a pair: the days picked show while out, then nerve's", async () => {
  const auth = await shown(its.member);
  expect(shownDays()).toBe("2026-11-16 → 2026-11-22");
  await pick(november(16), november(25));
  expect(held.modules.update).toHaveBeenCalledExactlyOnceWith(auth.id, {
    start_date: "2026-11-16",
    target_date: "2026-11-25",
  });
  expect(shownDays()).toBe("2026-11-16 → 2026-11-25");
  expect(enabled()).toEqual({ ...all(true), days: false });
  await settle();
  expect(toasts).toEqual([]);
  await answer({ ...auth, target_date: "2026-11-25" });
  expect(toasts).toEqual([success]);
  expect(shownDays()).toBe("2026-11-16 → 2026-11-25");
  expect(enabled()).toEqual(all(true));
});

it("waits for the other day of a range: one day alone sends nothing (R13)", async () => {
  const auth = await shown(its.member, { start_date: null, target_date: null });
  await pick(november(16));
  expect(held.modules.update).not.toHaveBeenCalled();
  expect(shownDays()).toBe("2026-11-16 → -");
  await pick(november(16), november(20));
  expect(held.modules.update).toHaveBeenCalledExactlyOnceWith(auth.id, {
    start_date: "2026-11-16",
    target_date: "2026-11-20",
  });
});

it("clears both its days as a pair", async () => {
  const auth = await shown(its.member);
  await pick();
  expect(held.modules.update).toHaveBeenCalledExactlyOnceWith(auth.id, { start_date: null, target_date: null });
  expect(shownDays()).toBe("- → -");
});

it("sends nothing for the days the module has", async () => {
  await shown(its.member);
  await pick(november(16), november(22));
  expect(held.modules.update).not.toHaveBeenCalled();
  expect(shownDays()).toBe("2026-11-16 → 2026-11-22");
});

const controls = [
  {
    control: "status",
    change: () => pickStatus("project_modules.status.paused"),
    shows: () => statusSelect()?.textContent,
    nerve: "project_modules.status.in_progress",
  },
  {
    control: "days",
    change: () => pick(november(16), november(25)),
    shows: shownDays,
    nerve: "2026-11-16 → 2026-11-22",
  },
  { control: "lead", change: () => choose("lead", bob), shows: () => held.pickers.lead?.value, nerve: ann },
  {
    control: "members",
    change: () => choose("members", [ann, bob]),
    shows: () => held.pickers.members?.value,
    nerve: [ann],
  },
];
it.each(controls)("says nerve's refusal of a change of its $control, never success, and shows nerve's", async (one) => {
  await shown(its.member);
  await one.change();
  await refuse(refusal(409, "planning.archived"));
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.planning_archived" }]);
  expect(one.shows()).toEqual(one.nerve);
  expect(enabled()).toEqual(all(true));
});

const leavings = [
  { leaves: "its sidebar has gone", leave: () => unmount() },
  { leaves: "the tab has moved to another account", leave: async () => switchAccount() },
];
const settlings = lateSettlingsAnswering(moduleOf(projectOf("WEB", "id-acme"), "Auth", AUTH));
it.each(
  controls.flatMap(({ control, change }) =>
    leavings.flatMap(({ leaves, leave }) =>
      settlings.map(({ settles, settle: settleChange }) => ({ control, change, leaves, leave, settles, settleChange }))
    )
  )
)("says nothing of a change of its $control that $settles once $leaves", async (one) => {
  await shown(its.member);
  await one.change();
  await one.leave();
  await act(async () => {
    const change = held.modules.sent.at(-1)?.change;
    if (change) one.settleChange(change);
  });
  await settle();
  expect(toasts).toEqual([]);
});
