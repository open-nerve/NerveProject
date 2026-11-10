// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Burndown, Cycle, CycleProgress, Progress } from "@nerve/api-client";
import { userOf } from "@/store/member/fake-users";
import { progressOf, rowOf } from "@/store/planning/fake-cycles";
import { cycleOf, labelOf, projectOf } from "@/store/project/fake-projects";
import { live, mount, page, renderSidebar, sidebarCycles, unmount } from "./fake-sidebar";

// The details and the progress of a cycle's progress sidebar (P5a spec 2.3–2.5, R7, R11): its description, its lead,
// its members (its list's row's), its work items, its burndown and its breakdowns by state, assignee and label. The
// numbers are nerve's /progress, of one choice (cycleCounts): what nerve froze once the cycle has ended, else live;
// "work items" is the completed of those not cancelled (progressPercent's counts). The names of assignees and labels
// are the project's members' and labels' (distribution-rows.ts). The cycles are a stand-in store's; the project's
// members and labels, the avatars and the chart stand-ins that say what they were given.

const held = vi.hoisted(() => ({
  cycles: undefined as unknown as ReturnType<typeof sidebarCycles>,
  burndowns: [] as Burndown[],
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@/hooks/store/use-cycles", () => ({ useCycles: () => held.cycles }));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => ({ getProjectRoleByWorkspaceSlugAndProjectId: () => 15 }),
}));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => ({ id: "id-acme" }) }));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    project: {
      getProjectMemberDetails: (id: string, projectId: string) => {
        const member = projectId === "p-web" ? [ann, bob].find((user) => user.id === id) : undefined;
        return member ? { member } : null;
      },
    },
  }),
}));
vi.mock("@/hooks/store/use-label", () => ({
  useLabel: () => ({ getProjectLabels: (projectId: string) => (projectId === "p-web" ? [bug, ui] : undefined) }),
}));
vi.mock("@/components/assets/profile-image", () => ({
  ProfileAvatar: (props: { source: { memberID: string } }) => <i data-avatar={props.source.memberID} />,
}));
vi.mock("@/components/core/sidebar/burndown-chart", () => ({
  BurndownChart: (props: { burndown: Burndown }) => {
    held.burndowns.push(props.burndown);
    return <figure data-burndown={props.burndown.scope} />;
  },
}));
vi.mock("@/components/dropdowns/date-range", () => ({ DateRangeDropdown: () => null }));

const web = projectOf("WEB", "id-acme");
const ann = userOf("Ann");
const bob = userOf("bob");
const bug = labelOf(web, "Bug", 1, { color: "#EF4444" });
const ui = labelOf(web, "UI", 2, { color: "#3B82F6" });

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  localStorage.clear();
  held.burndowns = [];
  mount();
});
afterEach(() => unmount());

const sprint = cycleOf(web, "Sprint", {
  status: "current",
  start_date: "2026-11-09",
  end_date: "2026-11-15",
  description: "Ship the release",
  owned_by_id: ann.id,
});
const burndown = (scope: number): Burndown => ({
  start_date: "2026-11-09",
  end_date: "2026-11-15",
  scope,
  points: [{ date: "2026-11-09", remaining: scope }],
});
/** Ten work items: four completed, two cancelled; by Ann and bob, by Bug. */
const tenLive = live({
  total_issues: 10,
  backlog_issues: 1,
  unstarted_issues: 1,
  started_issues: 2,
  completed_issues: 4,
  cancelled_issues: 2,
  assignees: [
    { assignee_id: null, total_issues: 2, completed_issues: 0, pending_issues: 2 },
    { assignee_id: bob.id, total_issues: 3, completed_issues: 2, pending_issues: 1 },
    { assignee_id: "u-gone", total_issues: 1, completed_issues: 1, pending_issues: 0 },
    { assignee_id: ann.id, total_issues: 2, completed_issues: 1, pending_issues: 1 },
  ],
  labels: [
    { label_id: ui.id, total_issues: 2, completed_issues: 1, pending_issues: 1 },
    { label_id: null, total_issues: 4, completed_issues: 2, pending_issues: 2 },
    { label_id: bug.id, total_issues: 2, completed_issues: 1, pending_issues: 1 },
  ],
  burndown: burndown(8),
});
/** Six work items, all completed, by the member who has left, as nerve froze them. */
const sixFrozen: Progress = live({
  total_issues: 6,
  completed_issues: 6,
  assignees: [{ assignee_id: "u-gone", total_issues: 6, completed_issues: 6, pending_issues: 0 }],
  burndown: burndown(6),
});
const progress = (cycle: Cycle, now: Progress, frozen: Progress | null): CycleProgress => ({
  ...progressOf(cycle),
  live: now,
  progress_snapshot: frozen && { ...frozen, frozen_at: "2026-11-02T00:00:00Z" },
});

const text = () => page.host?.textContent ?? "";
/** The value of a row of the details, by its title. */
const detail = (title: string) =>
  [...(page.host?.querySelectorAll("dt") ?? [])].find((term) => term.textContent === title)?.nextElementSibling;
const tab = (name: string) =>
  [...(page.host?.querySelectorAll<HTMLButtonElement>('[role="tab"]') ?? [])].find((one) => one.textContent === name);
/** The rows of the open tab, each as it reads: its name, then its share. */
const rows = () => [...(page.host?.querySelectorAll('[role="tabpanel"] li') ?? [])].map((row) => row.textContent);

async function shown(cycle: Cycle, read: Partial<{ row: ReturnType<typeof rowOf>; progress: CycleProgress }>) {
  held.cycles = sidebarCycles(cycle, read);
  await renderSidebar({ cycleId: cycle.id, projectId: web.id, handleClose: () => undefined });
}

it("waits for the store to give the cycle", async () => {
  held.cycles = sidebarCycles(sprint);
  held.cycles.show({ cycle: undefined });
  await renderSidebar({ cycleId: sprint.id, projectId: web.id, handleClose: () => undefined });
  expect(text()).not.toContain("Sprint");
  expect(page.host?.querySelector("dt")).toBeNull();
});

it("shows its description as plain text, its lead and its members by the project's members", async () => {
  await shown(sprint, { row: rowOf(sprint, { assignee_ids: [ann.id, bob.id] }) });
  expect(page.host?.querySelector("textarea")).toBeNull();
  expect(text()).toContain("Ship the release");
  expect(detail("lead")?.textContent).toBe("Ann");
  expect(detail("lead")?.querySelector("[data-avatar]")?.getAttribute("data-avatar")).toBe(ann.id);
  const avatars = detail("members")?.querySelectorAll("[data-avatar]") ?? [];
  expect([...avatars].map((avatar) => avatar.getAttribute("data-avatar"))).toEqual([ann.id, bob.id]);
});

it.each([
  { lead: "one no longer the project's member", owned_by_id: "u-gone", shows: "distribution.former_member" },
  { lead: "none", owned_by_id: null, shows: "none" },
])("names a lead that is $lead", async ({ owned_by_id, shows }) => {
  await shown({ ...sprint, owned_by_id }, {});
  expect(detail("lead")?.textContent).toBe(shows);
});

it("reads its members from its list's row alone: none until a list has it", async () => {
  await shown(sprint, {});
  expect(detail("members")?.textContent).not.toContain("no_assignee");
  expect(detail("members")?.querySelector("[data-avatar]")).toBeNull();
  await act(async () => held.cycles.show({ row: rowOf(sprint) }));
  expect(detail("members")?.textContent).toBe("no_assignee");
});

it("counts its work items as completed of those not cancelled, the shares of its states alike", async () => {
  await shown(sprint, { progress: progress(sprint, tenLive, null) });
  expect(detail("work_items")?.textContent).toBe("4/8");
  await act(async () => tab("common.states")?.click());
  const group = "project_settings.states.groups";
  expect(rows()).toEqual([
    `${group}.backlog13%distribution.of_total 8`,
    `${group}.unstarted13%distribution.of_total 8`,
    `${group}.started25%distribution.of_total 8`,
    `${group}.completed50%distribution.of_total 8`,
  ]);
  expect(text()).toContain("project_cycles.active_cycle.cancelled_excluded 2");
});

const FROZEN_FIRST = "distribution.former_member100%distribution.of_total 6";
const LIVE_FIRST = "Ann50%distribution.of_total 2";
it.each([
  { state: "ended and frozen", status: "completed", frozen: sixFrozen, shows: "6/6", scope: 6, first: FROZEN_FIRST },
  { state: "ended, not frozen yet", status: "completed", frozen: null, shows: "4/8", scope: 8, first: LIVE_FIRST },
  // frozen once it ended, then current again on the project's day (its time zone changed): the live numbers count
  {
    state: "in progress, though frozen",
    status: "current",
    frozen: sixFrozen,
    shows: "4/8",
    scope: 8,
    first: LIVE_FIRST,
  },
] as const)("shows the numbers of a cycle $state: $shows", async ({ status, frozen, shows, scope, first }) => {
  const cycle = { ...sprint, status };
  await shown(cycle, { progress: progress(cycle, tenLive, frozen) });
  expect(detail("work_items")?.textContent).toBe(shows);
  expect(held.burndowns.at(-1)?.scope).toBe(scope);
  expect(rows()[0]).toBe(first);
});

it("has no data yet for a draft: neither chart nor breakdowns", async () => {
  const draft = cycleOf(web, "Ideas");
  await shown(draft, { progress: progress(draft, live({ total_issues: 2, backlog_issues: 2 }), null) });
  expect(text()).toContain("no_data_yet");
  expect(held.burndowns).toEqual([]);
  expect(page.host?.querySelector('[role="tab"]')).toBeNull();
});

it("draws no chart without a burndown, and waits for its progress", async () => {
  await shown(sprint, {});
  expect(page.host?.querySelector('[role="tab"]')).toBeNull();
  expect(detail("work_items")?.textContent).not.toContain("/");
  await act(async () => held.cycles.show({ progress: progress(sprint, { ...tenLive, burndown: null }, null) }));
  expect(held.burndowns).toEqual([]);
  expect(tab("common.assignees")).toBeDefined();
});

it("names its assignees and labels by the project's: by name, the fallbacks after, none last", async () => {
  await shown(sprint, { progress: progress(sprint, tenLive, null) });
  expect(tab("common.assignees")?.getAttribute("aria-selected")).toBe("true");
  expect(rows()).toEqual([
    "Ann50%distribution.of_total 2",
    "bob67%distribution.of_total 3",
    "distribution.former_member100%distribution.of_total 1",
    "no_assignee0%distribution.of_total 2",
  ]);
  await act(async () => tab("common.labels")?.click());
  expect(rows()).toEqual([
    "Bug50%distribution.of_total 2",
    "UI50%distribution.of_total 2",
    "distribution.no_labels50%distribution.of_total 4",
  ]);
  const dots = page.host?.querySelectorAll<HTMLElement>('[role="tabpanel"] li span[aria-hidden="true"]') ?? [];
  expect([...dots].map((dot) => dot.style.backgroundColor)).toEqual(["rgb(239, 68, 68)", "rgb(59, 130, 246)", ""]);
});

it("keeps the tab chosen for the cycle", async () => {
  await shown(sprint, { progress: progress(sprint, tenLive, null) });
  await act(async () => tab("common.labels")?.click());
  expect(localStorage.getItem(`cycle-progress-tab-${sprint.id}`)).toBe('"stat-labels"');
  await unmount();
  mount();
  await shown(sprint, { progress: progress(sprint, tenLive, null) });
  expect(tab("common.labels")?.getAttribute("aria-selected")).toBe("true");
});
