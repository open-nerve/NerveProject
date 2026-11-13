// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Progress } from "@nerve/api-client";
import { userOf } from "@/store/member/fake-users";
import { labelOf, projectOf } from "@/store/project/fake-projects";
import { ProgressTabs } from "./progress-tabs";

// The breakdowns of a cycle's or a module's progress, each a tab (P6a spec 2.5, R8): by state, each group's share of
// the work items not cancelled; by assignee and by label, named by the project's members and labels
// (distribution-rows.ts). Each tab shows its own breakdown, in the order of the tabs. The tab chosen is kept under the
// caller's key; the rows are plain unless the caller gives what choosing one does. The project's members and labels
// are stand-ins, as are the avatars.

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => ({ id: "id-acme" }) }));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    project: {
      getProjectMemberDetails: (id: string, projectId: string) =>
        projectId === "p-web" && id === ann.id ? { member: ann } : null,
    },
  }),
}));
vi.mock("@/hooks/store/use-label", () => ({
  useLabel: () => ({ getProjectLabels: (projectId: string) => (projectId === "p-web" ? [bug] : undefined) }),
}));
vi.mock("@/components/assets/profile-image", () => ({
  ProfileAvatar: (props: { source: { memberID: string } }) => <i data-avatar={props.source.memberID} />,
}));

const web = projectOf("WEB", "id-acme");
const ann = userOf("Ann");
const bug = labelOf(web, "Bug", 1, { color: "#EF4444" });
const KEY = "module-progress-tab-m-auth";

/** Ten work items: four completed, two cancelled; by Ann and none, by Bug and none. */
const progress: Progress = {
  total_issues: 10,
  backlog_issues: 1,
  unstarted_issues: 1,
  started_issues: 2,
  completed_issues: 4,
  cancelled_issues: 2,
  assignees: [
    { assignee_id: null, total_issues: 4, completed_issues: 1, pending_issues: 3 },
    { assignee_id: ann.id, total_issues: 6, completed_issues: 3, pending_issues: 3 },
  ],
  labels: [
    { label_id: null, total_issues: 6, completed_issues: 2, pending_issues: 4 },
    { label_id: bug.id, total_issues: 4, completed_issues: 2, pending_issues: 2 },
  ],
  burndown: null,
};

const group = "project_settings.states.groups";
const STATES = [
  `${group}.backlog13%distribution.of_total 8`,
  `${group}.unstarted13%distribution.of_total 8`,
  `${group}.started25%distribution.of_total 8`,
  `${group}.completed50%distribution.of_total 8`,
];
const ASSIGNEES = ["Ann50%distribution.of_total 6", "no_assignee25%distribution.of_total 4"];
const LABELS = ["Bug50%distribution.of_total 4", "distribution.no_labels33%distribution.of_total 6"];

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  localStorage.clear();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});

const tabs = () => [...host.querySelectorAll<HTMLButtonElement>('[role="tab"]')];
const tab = (name: string) => tabs().find((one) => one.textContent === name);
/** The panel the selected tab controls (aria-controls). */
const panel = () => {
  const id = tabs()
    .find((one) => one.getAttribute("aria-selected") === "true")
    ?.getAttribute("aria-controls");
  return id ? document.getElementById(id) : null;
};
/** The rows of the open tab, each as it reads: its name, then its share. */
const rows = () => [...(panel()?.querySelectorAll("li") ?? [])].map((row) => row.textContent);
const choose = async (name: string) => act(async () => tab(name)?.click());

const shown = (props: Partial<ComponentProps<typeof ProgressTabs>> = {}) =>
  act(async () => root.render(<ProgressTabs progress={progress} projectId={web.id} storageKey={KEY} {...props} />));

it("shows under each tab its own breakdown, the tabs by state, assignee and label, assignees first", async () => {
  await shown();
  expect(tabs().map((one) => one.textContent)).toEqual(["common.states", "common.assignees", "common.labels"]);
  expect(tab("common.assignees")?.getAttribute("aria-selected")).toBe("true");
  expect(rows()).toEqual(ASSIGNEES);
  await choose("common.states");
  expect(rows()).toEqual(STATES);
  expect(panel()?.textContent).toContain("project_cycles.active_cycle.cancelled_excluded 2");
  await choose("common.labels");
  expect(rows()).toEqual(LABELS);
  await choose("common.assignees");
  expect(rows()).toEqual(ASSIGNEES);
});

it("keeps the tab chosen under the caller's key", async () => {
  await shown();
  await choose("common.states");
  expect(localStorage.getItem(KEY)).toBe('"stat-states"');
  expect(Object.keys(localStorage)).toEqual([KEY]);
  await act(async () => root.unmount());
  root = createRoot(host);
  await shown();
  expect(tab("common.states")?.getAttribute("aria-selected")).toBe("true");
  expect(rows()).toEqual(STATES);
});

it.each(["common.states", "common.assignees", "common.labels"])(
  "shows plain rows under %s when the caller gives no choice of them: none is a button",
  async (name) => {
    await shown();
    await choose(name);
    expect(rows().length).toBeGreaterThan(0);
    expect(panel()?.querySelectorAll("button")).toHaveLength(0);
  }
);

it("makes a row of a value a button, which says whether its value is chosen and chooses it; the row of none stays plain", async () => {
  const choices = {
    states: { choose: vi.fn(), selected: ["started"] },
    assignees: { choose: vi.fn(), selected: [ann.id] },
    labels: { choose: vi.fn(), selected: [] },
  };
  await shown({ choices });
  const buttons = () => [...(panel()?.querySelectorAll("button") ?? [])];
  const pressed = () => buttons().map((one) => one.getAttribute("aria-pressed"));

  await choose("common.states");
  expect(pressed()).toEqual(["false", "false", "true", "false"]);
  await act(async () => buttons()[0]?.click());
  expect(choices.states.choose.mock.calls).toEqual([["backlog"]]);

  await choose("common.assignees");
  expect(buttons().map((one) => one.textContent)).toEqual([ASSIGNEES[0]]);
  expect(pressed()).toEqual(["true"]);
  await act(async () => buttons()[0]?.click());
  expect(choices.assignees.choose.mock.calls).toEqual([[ann.id]]);

  await choose("common.labels");
  expect(buttons().map((one) => one.textContent)).toEqual([LABELS[0]]);
  expect(pressed()).toEqual(["false"]);
  await act(async () => buttons()[0]?.click());
  expect(choices.labels.choose.mock.calls).toEqual([[bug.id]]);
});
