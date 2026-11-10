// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Cycle } from "@nerve/api-client";
import type { DateRangeDropdown } from "@/components/dropdowns/date-range";
import { lateSettlingsAnswering, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { cycleOf, projectOf } from "@/store/project/fake-projects";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { renderFormattedPayloadDate } from "@nerve/utils";
import { mount, page, renderSidebar, sidebarCycles, unmount } from "./fake-sidebar";

// The head of a cycle's progress sidebar (P5a spec 2.5, R8, R12): its close button, its name and its status as nerve
// gives it, and its days, which the caller changes there while he writes the project and the cycle has neither ended
// nor been archived. A change of days is nerve's to make: the sidebar says so once nerve has made it, and shows nerve's
// days; refused, it says why and shows nerve's days again. The caller's role is the permission store's own, over acme
// and its project web as nerve lists them to him (fake-permissions.ts); the cycles are a stand-in store's.

const held = vi.hoisted(() => ({
  cycles: undefined as unknown as ReturnType<typeof sidebarCycles>,
  permissions: undefined as IUserPermissionStore | undefined,
  /** The props the dates' picker was last given. */
  dates: undefined as ComponentProps<typeof DateRangeDropdown> | undefined,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/hooks/store/use-cycles", () => ({ useCycles: () => held.cycles }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => held.permissions }));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({ project: { getProjectMemberDetails: () => null } }),
}));
vi.mock("@/hooks/store/use-label", () => ({ useLabel: () => ({ getProjectLabels: () => [] }) }));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => null }));
vi.mock("@/components/assets/profile-image", () => ({ ProfileAvatar: () => null }));
// the dates' picker keeps its props, so that a test reads the days it shows and picks a range as its calendar would
vi.mock("@/components/dropdowns/date-range", () => ({
  DateRangeDropdown: (props: ComponentProps<typeof DateRangeDropdown>) => {
    held.dates = props;
    return <button type="button" aria-label="dates" disabled={props.disabled} />;
  },
}));

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers({ now: new Date(2026, 10, 11, 12) });
  signedIn();
  toasts.length = 0;
  held.dates = undefined;
  mount();
});
afterEach(async () => {
  await unmount();
  vi.useRealTimers();
});

const states = {
  draft: { status: "draft", start_date: null, end_date: null },
  upcoming: { status: "upcoming", start_date: "2026-11-16", end_date: "2026-11-22" },
  "in progress": { status: "current", start_date: "2026-11-09", end_date: "2026-11-15" },
  ended: { status: "completed", start_date: "2026-10-26", end_date: "2026-11-01" },
  "archived, ended": { status: "completed", archived_at: "2026-11-02T09:00:00Z" },
  // archived while it had ended, then current again on the project's day (a change of the project's time zone)
  "archived, in progress": { status: "current", archived_at: "2026-11-02T09:00:00Z" },
} satisfies Record<string, Partial<Cycle>>;

/** The caller of a role in acme and in web, at the sidebar of web's cycle Sprint, of state: Sprint. */
async function shown(caller: Caller, state: keyof typeof states) {
  const { permissions, web } = await callerInWeb(caller.workspaceRole, caller.projectRole, caller.guestViewAll);
  held.permissions = permissions;
  const sprint = cycleOf(web, "Sprint", states[state]);
  held.cycles = sidebarCycles(sprint);
  await renderSidebar({ cycleId: sprint.id, projectId: web.id, handleClose: close });
  return sprint;
}
const close = vi.fn();
const datesEnabled = () => held.dates?.disabled !== true;
/** A day of November 2026, as the calendar gives it: the local midnight of the day. */
const november = (day: number) => new Date(2026, 10, day);
/** The person picks a range in the calendar. */
const pick = (from: Date | undefined, to?: Date) => act(async () => held.dates?.onSelect?.({ from, to }));
/** The days the picker shows, as nerve writes them ("-" for none). */
const shownDays = () => {
  const { from, to } = held.dates?.value ?? {};
  return `${renderFormattedPayloadDate(from) ?? "-"} → ${renderFormattedPayloadDate(to) ?? "-"}`;
};
const settle = () => act(async () => vi.advanceTimersByTimeAsync(100));
const success = {
  type: "success",
  title: "project_cycles.action.update.success.title",
  message: "project_cycles.action.update.success.description",
};

it("names its close button, and shows the cycle's name and its status as nerve gives it", async () => {
  close.mockReset();
  await shown(its.member, "in progress");
  const button = page.host?.querySelector<HTMLButtonElement>('button[aria-label="close"]');
  await act(async () => button?.click());
  expect(close).toHaveBeenCalledOnce();
  expect(page.host?.querySelector("h4")?.textContent).toBe("Sprint");
  expect(page.host?.textContent).toContain("project_cycles.status.in_progress");
});

it.each([
  { ...its.admin, state: "upcoming", editable: true },
  { ...its.member, state: "in progress", editable: true },
  { ...its.memberAdmin, state: "draft", editable: true },
  { ...its.guest, state: "upcoming", editable: false },
  { ...its.guestAdmin, state: "upcoming", editable: false },
  { ...its.guestAdminSeesAll, state: "upcoming", editable: true },
  { ...its.guestSeesAll, state: "upcoming", editable: false },
  { ...its.outsider, state: "upcoming", editable: false },
  { ...its.admin, state: "ended", editable: false },
  { ...its.admin, state: "archived, ended", editable: false },
  { ...its.admin, state: "archived, in progress", editable: false },
] as const)("lets $who change the days of a cycle $state: $editable", async ({ state, editable, ...caller }) => {
  await shown(caller, state);
  expect(datesEnabled()).toBe(editable);
});

it("changes the days once nerve has: no success until it answers, then nerve's days", async () => {
  const sprint = await shown(its.member, "upcoming");
  expect(shownDays()).toBe("2026-11-16 → 2026-11-22");
  await pick(november(16), november(25));
  expect(held.cycles.update).toHaveBeenCalledExactlyOnceWith(sprint.id, {
    start_date: "2026-11-16",
    end_date: "2026-11-25",
  });
  // out: the days picked show, no other may be picked, and nothing is said yet
  expect(shownDays()).toBe("2026-11-16 → 2026-11-25");
  expect(datesEnabled()).toBe(false);
  await settle();
  expect(toasts).toEqual([]);
  await act(async () => held.cycles.sent[0]?.change.answer({ ...sprint, end_date: "2026-11-25" }));
  await settle();
  expect(toasts).toEqual([success]);
  expect(shownDays()).toBe("2026-11-16 → 2026-11-25");
  expect(datesEnabled()).toBe(true);
});

it("says nerve's refusal of the days, never success, and shows nerve's days again", async () => {
  await shown(its.member, "upcoming");
  await pick(november(16), november(25));
  await act(async () => held.cycles.sent[0]?.change.refuse(refusal(409, "planning.cycle_dates_overlap")));
  await settle();
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.planning_cycle_dates_overlap" }]);
  expect(shownDays()).toBe("2026-11-16 → 2026-11-22");
  expect(datesEnabled()).toBe(true);
});

it("waits for the other day of a draft's range: nerve takes the days as a pair", async () => {
  const draft = await shown(its.member, "draft");
  await pick(november(16));
  expect(held.cycles.update).not.toHaveBeenCalled();
  expect(shownDays()).toBe("2026-11-16 → -");
  await pick(november(16), november(20));
  expect(held.cycles.update).toHaveBeenCalledExactlyOnceWith(draft.id, {
    start_date: "2026-11-16",
    end_date: "2026-11-20",
  });
});

it("sends nothing for the days the cycle already has", async () => {
  await shown(its.member, "upcoming");
  await pick(november(16), november(22));
  expect(held.cycles.update).not.toHaveBeenCalled();
  expect(shownDays()).toBe("2026-11-16 → 2026-11-22");
});

const leavings = [
  { leaves: "its page has gone", leave: () => unmount() },
  { leaves: "the tab has moved to another account", leave: async () => switchAccount() },
];
const settlings = lateSettlingsAnswering(cycleOf(projectOf("WEB", "id-acme"), "Sprint", states.upcoming));
it.each(
  leavings.flatMap(({ leaves, leave }) =>
    settlings.map(({ settles, settle: settleChange }) => ({ leaves, leave, settles, settleChange }))
  )
)("says nothing of a change of days that $settles once $leaves", async ({ leave, settleChange }) => {
  await shown(its.member, "upcoming");
  await pick(november(16), november(25));
  await leave();
  await act(async () => {
    const change = held.cycles.sent[0]?.change;
    if (change) settleChange(change);
  });
  await settle();
  expect(toasts).toEqual([]);
});
