// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Burndown, CycleProgress, MemberUser, Progress } from "@nerve/api-client";
import { refusal } from "@/lib/fake-refusal";
import { userOf } from "@/store/member/fake-users";
import { progressOf, rowOf } from "@/store/planning/fake-cycles";
import { cycleOf, labelOf, projectOf } from "@/store/project/fake-projects";
import { CyclePeekOverview } from "../cycle-peek-overview";
import { listedCycles } from "../fake-cycle-lists";
import { ActiveCycleRoot } from "./root";

// The card of a project's cycle in progress on its cycles list (P5a spec 2.3–2.5, R7, R10): none, it says so; else
// the cycle's row, its progress (the work items of each state group, the cancelled said apart, the completed of those
// not cancelled), its burndown (nerve's, the work items pending) and its breakdowns by assignee and by label, named by
// the project's (distribution-rows.ts), the tab chosen kept. Its numbers are nerve's /progress of one choice
// (cycleCounts): a cycle in progress has its live ones. It reads the progress by the key the cycle's page and its
// preview read it by. The cycles are a stand-in store's; the project's members and labels, the avatars, the cycle's row
// and the chart stand-ins that say what they were given.

const held = vi.hoisted(() => ({
  cycles: undefined as unknown as ReturnType<typeof listedCycles>,
  burndowns: [] as Burndown[],
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "x" }) }));
vi.mock("@/hooks/store/use-cycles", () => ({ useCycles: () => held.cycles }));
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
vi.mock("@/components/cycles/list/cycles-list-item", () => ({
  CyclesListItem: (props: { cycleId: string }) => <a href={`#${props.cycleId}`}>row of {props.cycleId}</a>,
}));
vi.mock("@/components/cycles/progress-sidebar", () => ({ CycleDetailsSidebar: () => null }));

const web = projectOf("WEB", "id-acme");
// members and labels whose ids are not in the order of their names: nerve gives its rows by id
const ann: MemberUser = { ...userOf("Ann"), id: "u-2" };
const bob: MemberUser = { ...userOf("bob"), id: "u-1" };
const bug = labelOf(web, "Bug", 1, { id: "l-2", color: "#EF4444" });
const ui = labelOf(web, "UI", 2, { id: "l-1", color: "#3B82F6" });
const sprint = cycleOf(web, "S2", { status: "current", start_date: "2026-11-09", end_date: "2026-11-15" });
const burndown = (scope: number): Burndown => ({
  start_date: "2026-11-09",
  end_date: "2026-11-15",
  scope,
  points: [{ date: "2026-11-09", remaining: scope }],
});
/** Ten work items: four completed, two cancelled; by assignee and by label as nerve gives them, by id, none last. */
const tenLive: Partial<Progress> = {
  total_issues: 10,
  backlog_issues: 1,
  unstarted_issues: 1,
  started_issues: 2,
  completed_issues: 4,
  cancelled_issues: 2,
  assignees: [
    { assignee_id: "u-0", total_issues: 1, completed_issues: 1, pending_issues: 0 },
    { assignee_id: bob.id, total_issues: 3, completed_issues: 2, pending_issues: 1 },
    { assignee_id: ann.id, total_issues: 2, completed_issues: 1, pending_issues: 1 },
    { assignee_id: null, total_issues: 2, completed_issues: 0, pending_issues: 2 },
  ],
  labels: [
    { label_id: ui.id, total_issues: 2, completed_issues: 1, pending_issues: 1 },
    { label_id: bug.id, total_issues: 2, completed_issues: 1, pending_issues: 1 },
    { label_id: null, total_issues: 4, completed_issues: 2, pending_issues: 2 },
  ],
  burndown: burndown(8),
};
/** Six work items, all completed, as nerve froze them once. */
const sixFrozen: CycleProgress["progress_snapshot"] = {
  ...progressOf(sprint, { total_issues: 6, completed_issues: 6, burndown: burndown(6) }).live,
  frozen_at: "2026-11-02T00:00:00Z",
};

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  localStorage.clear();
  held.burndowns = [];
  held.cycles = listedCycles();
  mount();
});
afterEach(() => unmount());

function mount() {
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
}
async function unmount() {
  await act(async () => root.unmount());
  host.remove();
}

/** Renders the card of web's cycle in progress, at the list's address and its query, with what else the page has. */
async function render(search = "", beside: ReactNode = null) {
  await act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <MemoryRouter initialEntries={[`/acme/projects/${web.id}/cycles${search}`]}>
          <ActiveCycleRoot workspaceSlug="acme" projectId={web.id} />
          {beside}
        </MemoryRouter>
      </SWRConfig>
    )
  );
}
/** web's cycles are listed, sprint in progress among them, and the card shows; nerve answers its progress. */
async function inProgress(progress: CycleProgress = progressOf(sprint, tenLive)) {
  held.cycles.show("cycles", web.id, [rowOf(cycleOf(web, "S3", { status: "upcoming" })), rowOf(sprint)]);
  await render();
  await held.cycles.answer("progress", progress);
}

const text = () => host.textContent ?? "";
/** A card, by its title. */
const card = (title: string) => host.querySelector<HTMLElement>(`section[aria-label="${title}"]`);
const PROGRESS = "project_cycles.active_cycle.progress";
const BURNDOWN = "project_cycles.active_cycle.issue_burndown";
const tabs = () => [...host.querySelectorAll<HTMLButtonElement>('[role="tab"]')];
const tab = (name: string) => tabs().find((one) => one.textContent === name);
/** The rows of the open tab, each as it reads: its name, then its share. */
const rows = () => [...host.querySelectorAll('[role="tabpanel"] li')].map((row) => row.textContent);
const ASSIGNEES = "project_cycles.active_cycle.assignees";
const LABELS = "project_cycles.active_cycle.labels";

it("waits for the project's cycles, reading nothing yet", async () => {
  await render();
  expect(host.querySelector('[role="status"]')).not.toBeNull();
  expect(text()).not.toContain("project_cycles.empty_state.active.title");
  expect(held.cycles.asked()).toEqual([]);
});

it("says that the project has no cycle in progress, reading no progress", async () => {
  held.cycles.show("cycles", web.id, [rowOf(cycleOf(web, "S3", { status: "upcoming" })), rowOf(cycleOf(web, "D"))]);
  await render();
  expect(text()).toContain("project_cycles.empty_state.active.title");
  expect(host.querySelector('[role="status"]')).toBeNull();
  expect(held.cycles.asked()).toEqual([]);
});

it("shows the cycle in progress by its row, and reads its progress by the key its page and its preview share", async () => {
  held.cycles.show("cycles", web.id, [rowOf(sprint)]);
  await render(`?peekCycle=${sprint.id}`, <CyclePeekOverview workspaceSlug="acme" projectId={web.id} />);
  expect(host.querySelector(`a[href="#${sprint.id}"]`)?.textContent).toBe(`row of ${sprint.id}`);
  // one read of the progress, the card's and the preview's
  expect(held.cycles.asked().toSorted()).toEqual([`cycle of ${sprint.id}`, `progress of ${sprint.id}`]);
  await held.cycles.answer("progress", progressOf(sprint, tenLive));
  expect(card(PROGRESS)?.querySelector('[role="progressbar"]')?.getAttribute("aria-valuenow")).toBe("50");
});

it("waits for its progress", async () => {
  held.cycles.show("cycles", web.id, [rowOf(sprint)]);
  await render();
  expect(held.cycles.asked()).toEqual([`progress of ${sprint.id}`]);
  expect(host.querySelector('[role="status"]')).not.toBeNull();
  expect(card(PROGRESS)).toBeNull();
  expect(tabs()).toEqual([]);
  await held.cycles.answer("progress", progressOf(sprint, tenLive));
  expect(host.querySelector('[role="status"]')).toBeNull();
  expect(card(PROGRESS)).not.toBeNull();
});

it("shows its progress: each state group's work items, the cancelled said apart, the completed of the rest", async () => {
  await inProgress();
  const progress = card(PROGRESS);
  expect(progress?.querySelector('[role="progressbar"]')?.getAttribute("aria-valuenow")).toBe("50");
  expect(progress?.textContent).toContain("project_cycles.active_cycle.work_items_completed 4 8");
  const group = "project_settings.states.groups";
  expect([...(progress?.querySelectorAll("li") ?? [])].map((row) => row.textContent)).toEqual([
    `${group}.completed4 issue.label 4`,
    `${group}.started2 issue.label 2`,
    `${group}.unstarted1 issue.label 1`,
    `${group}.backlog1 issue.label 1`,
  ]);
  expect(progress?.textContent).toContain("project_cycles.active_cycle.cancelled_excluded 2");
});

it("counts a cycle in progress by its live progress, though nerve keeps what it froze of it (R7)", async () => {
  // frozen once it ended, then in progress again on the project's day (its days changed): the live numbers count
  await inProgress({ ...progressOf(sprint, tenLive), progress_snapshot: sixFrozen });
  expect(card(PROGRESS)?.querySelector('[role="progressbar"]')?.getAttribute("aria-valuenow")).toBe("50");
  expect(card(BURNDOWN)?.textContent).toContain("project_cycles.active_cycle.pending_work_items 4");
  expect(held.burndowns.at(-1)?.scope).toBe(8);
  expect(rows()[0]).toBe("Ann50%distribution.of_total 2");
});

it("shows its burndown: nerve's, and its work items pending, linked to the cycle's page", async () => {
  await inProgress();
  const burndownCard = card(BURNDOWN);
  // backlog, unstarted and started
  expect(burndownCard?.textContent).toContain("project_cycles.active_cycle.pending_work_items 4");
  expect(burndownCard?.querySelector("figure")?.getAttribute("data-burndown")).toBe("8");
  expect(held.burndowns.at(-1)).toEqual(burndown(8));
  const links = [...(burndownCard?.querySelectorAll("a") ?? [])].map((link) => link.getAttribute("href"));
  expect(links).toContain(`/acme/projects/${web.id}/cycles/${sprint.id}`);
});

it("has neither progress nor burndown to show of a cycle without work items", async () => {
  await inProgress(progressOf(sprint, { burndown: burndown(0) }));
  expect(card(PROGRESS)?.textContent).toContain("active_cycle.empty_state.progress.title");
  expect(card(PROGRESS)?.querySelector('[role="progressbar"]')).toBeNull();
  expect(card(BURNDOWN)?.textContent).toContain("active_cycle.empty_state.chart.title");
  expect(held.burndowns).toEqual([]);
  expect(text()).toContain("active_cycle.empty_state.assignee.title");
});

it("breaks its work items down by assignee and by label alone, named by the project's, by name, none last", async () => {
  await inProgress();
  // the tab of priority work items first (P5b spec 2.7), the assignees' shown
  expect(tabs().map((one) => one.textContent)).toEqual([
    "project_cycles.active_cycle.priority_issue",
    ASSIGNEES,
    LABELS,
  ]);
  expect(tab(ASSIGNEES)?.getAttribute("aria-selected")).toBe("true");
  expect(rows()).toEqual([
    "Ann50%distribution.of_total 2",
    "bob67%distribution.of_total 3",
    "distribution.former_member100%distribution.of_total 1",
    "no_assignee0%distribution.of_total 2",
  ]);
  const avatars = host.querySelectorAll('[role="tabpanel"] [data-avatar]');
  expect([...avatars].map((avatar) => avatar.getAttribute("data-avatar"))).toEqual([ann.id, bob.id]);
  await act(async () => tab(LABELS)?.click());
  expect(rows()).toEqual([
    "Bug50%distribution.of_total 2",
    "UI50%distribution.of_total 2",
    "distribution.no_labels50%distribution.of_total 4",
  ]);
});

it("keeps the tab chosen (activeCycleTab)", async () => {
  await inProgress();
  await act(async () => tab(LABELS)?.click());
  expect(localStorage.getItem("activeCycleTab")).toBe('"Labels"');
  await unmount();
  mount();
  held.cycles = listedCycles();
  await inProgress();
  expect(tab(LABELS)?.getAttribute("aria-selected")).toBe("true");
  expect(rows()[0]).toBe("Bug50%distribution.of_total 2");
});

it("says why nerve could not read its progress, and reads it again on retry", async () => {
  held.cycles.show("cycles", web.id, [rowOf(sprint)]);
  await render();
  await held.cycles.refuse("progress", refusal(503, "server_busy"));
  expect(host.querySelector('[role="alert"]')?.textContent).toContain("errors.server_busy");
  expect(host.querySelector('[role="status"]')).toBeNull();
  const retry = [...host.querySelectorAll("button")].find((one) => one.textContent === "common.retry");
  await act(async () => retry?.click());
  expect(held.cycles.asked()).toEqual([`progress of ${sprint.id}`, `progress of ${sprint.id}`]);
  await held.cycles.answer("progress", progressOf(sprint, tenLive));
  expect(host.querySelector('[role="alert"]')).toBeNull();
  expect(card(PROGRESS)).not.toBeNull();
});
