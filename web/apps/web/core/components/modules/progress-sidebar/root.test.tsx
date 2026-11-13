// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Burndown, Module, Progress } from "@nerve/api-client";
import type { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { userOf } from "@/store/member/fake-users";
import { moduleProgressOf } from "@/store/planning/fake-modules";
import { labelOf, moduleOf, projectOf } from "@/store/project/fake-projects";
import { mount, page, renderSidebar, sidebarModules, unmount } from "./fake-sidebar";

// The details and the progress of a module's progress sidebar (P6a spec 2.5, 2.6, R8, R12): its name, its description,
// its lead and its members (nerve's, in the pickers of the route's project), its work items, how far it has come, its
// burndown and its breakdowns by state, assignee and label. The numbers are nerve's /progress: "work items" is the
// completed of those not cancelled, and how far it has come progressPercent's of the same counts; the breakdowns are the
// shared tabs, plain rows, the tab kept for the module. The modules are a stand-in store's; the project's members and
// labels, the avatars, the pickers and the chart stand-ins that say what they were given; the links (P6a T9) none.

type Picker = ComponentProps<typeof MemberDropdown>;
const held = vi.hoisted(() => ({
  modules: undefined as unknown as ReturnType<typeof sidebarModules>,
  burndowns: [] as Burndown[],
  /** The props each members' picker was last given: the lead's, the members'. */
  pickers: {} as Record<"lead" | "members", Picker | undefined>,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@/hooks/store/use-modules", () => ({ useModules: () => held.modules }));
// the caller writes the project's modules (its member)
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => ({ canWriteModules: () => true }) }));
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
vi.mock("@/components/dropdowns/member/dropdown", () => ({
  MemberDropdown: (props: Picker) => {
    held.pickers[props.multiple ? "members" : "lead"] = props;
    return null;
  },
}));
vi.mock("./sidebar-links", () => ({ ModuleSidebarLinks: () => null }));

const web = projectOf("WEB", "id-acme");
const ann = userOf("Ann");
const bob = userOf("bob");
const bug = labelOf(web, "Bug", 1, { color: "#EF4444" });
const ui = labelOf(web, "UI", 2, { color: "#3B82F6" });

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  localStorage.clear();
  held.burndowns = [];
  held.pickers = { lead: undefined, members: undefined };
  mount();
});
afterEach(() => unmount());

const auth = moduleOf(web, "Auth", {
  status: "in-progress",
  start_date: "2026-11-09",
  target_date: "2026-11-20",
  description: "Sign in\nand out",
  lead_id: ann.id,
  member_ids: [ann.id, bob.id],
});
const burndown = (scope: number): Burndown => ({
  start_date: "2026-11-09",
  end_date: "2026-11-20",
  scope,
  points: [{ date: "2026-11-09", remaining: scope }],
});
/** Ten work items: four completed, two cancelled; by Ann and bob, by Bug. */
const ten = moduleProgressOf({
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

const text = () => page.host?.textContent ?? "";
/** The value of a row of the details, by its title. */
const detail = (title: string) =>
  [...(page.host?.querySelectorAll("dt") ?? [])].find((term) => term.textContent === title)?.nextElementSibling;
const tab = (name: string) =>
  [...(page.host?.querySelectorAll<HTMLButtonElement>('[role="tab"]') ?? [])].find((one) => one.textContent === name);
/** The rows of the open tab, each as it reads: its name, then its share. */
const rows = () => [...(page.host?.querySelectorAll('[role="tabpanel"] li') ?? [])].map((row) => row.textContent);

async function shown(module: Module, progress?: Progress) {
  held.modules = sidebarModules(module, { progress });
  await renderSidebar({ moduleId: module.id, projectId: web.id, handleClose: () => undefined });
}

it("waits for the store to give the module", async () => {
  held.modules = sidebarModules(auth);
  held.modules.show({ module: undefined });
  await renderSidebar({ moduleId: auth.id, projectId: web.id, handleClose: () => undefined });
  expect(text()).not.toContain("Auth");
  expect(page.host?.querySelector("dt")).toBeNull();
});

it("shows its name, its description as plain text, and nerve's lead and members in the route project's pickers", async () => {
  await shown(auth);
  expect(page.host?.querySelector("h4")?.textContent).toBe("Auth");
  expect(page.host?.querySelector("textarea")).toBeNull();
  expect(page.host?.querySelector("p")?.textContent).toBe("Sign in\nand out");
  expect(held.pickers.lead).toMatchObject({ value: ann.id, projectId: web.id, multiple: false });
  expect(held.pickers.members).toMatchObject({ value: [ann.id, bob.id], projectId: web.id, multiple: true });
});

it("counts its work items as completed of those not cancelled; how far it has come and the states' shares alike", async () => {
  await shown(auth, ten);
  expect(detail("work_items")?.textContent).toBe("4/8");
  expect(text()).toContain("common.progress50%");
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

it("draws its burndown when nerve gives one", async () => {
  await shown(auth, ten);
  expect(held.burndowns.at(-1)?.scope).toBe(8);
  expect(text()).not.toContain("no_data_yet");
});

it("has no data yet without a burndown, and breaks its work items down all the same (R8)", async () => {
  await shown({ ...auth, start_date: null }, { ...ten, burndown: null });
  expect(held.burndowns).toEqual([]);
  expect(text()).toContain("no_data_yet");
  expect(tab("common.assignees")?.getAttribute("aria-selected")).toBe("true");
  expect(rows()).toHaveLength(4);
});

it("waits for its progress: neither its work items nor its breakdowns yet", async () => {
  await shown(auth);
  expect(detail("work_items")?.textContent).not.toContain("/");
  expect(page.host?.querySelector('[role="tab"]')).toBeNull();
  await act(async () => held.modules.show({ progress: ten }));
  expect(detail("work_items")?.textContent).toBe("4/8");
  expect(tab("common.assignees")).toBeDefined();
});

it("breaks its work items down by the project's assignees and labels, in plain rows that choose nothing", async () => {
  await shown(auth, ten);
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
  expect(page.host?.querySelector('[role="tabpanel"] li button')).toBeNull();
});

it("keeps the tab chosen for the module", async () => {
  await shown(auth, ten);
  await act(async () => tab("common.labels")?.click());
  expect(localStorage.getItem(`module-progress-tab-${auth.id}`)).toBe('"stat-labels"');
  await unmount();
  mount();
  await shown(auth, ten);
  expect(tab("common.labels")?.getAttribute("aria-selected")).toBe("true");
});
