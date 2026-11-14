// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// M6 P7a spec 2.5, R3, R11: the cycle a work item is in, in its detail's sidebar, against the session's stores and a
// fake nerve: shown while its project's cycles are on, changed through the detail's operations, one PATCH a pick.
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Cycle, IssueDetail } from "@nerve/api-client";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { cycleRequests, loadCycles, rowOf } from "@/store/planning/fake-cycles";
import { cycleOf, loadProjects } from "@/store/project/fake-projects";
import { IssueDetailsSidebar } from "./sidebar";
import { useIssueOperations } from "./use-issue-operations";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "sidebar" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "ws" }) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-cycles", () => ({ useCycles: () => tab.app.planning.cycles }));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => tab.app.state }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: () => undefined }) }));
// The sidebar's other properties have their own tests; here they are absent.
vi.mock("@/components/dropdowns/state/dropdown", () => ({ StateDropdown: () => null }));
vi.mock("@/components/dropdowns/member/dropdown", () => ({ MemberDropdown: () => null }));
vi.mock("@/components/dropdowns/priority", () => ({ PriorityDropdown: () => null }));
vi.mock("@/components/dropdowns/date", () => ({ DateDropdown: () => null }));
vi.mock("@/components/issues/parent-select-root", () => ({ IssueParentSelectRoot: () => null }));
vi.mock("./label", () => ({ IssueLabel: () => null }));

let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let renderer: ReturnType<typeof createRoot>;
let now: Cycle, next: Cycle;
const workItem = (cycle_id: string | null, updated_at = "2026-10-29T00:00:00Z"): IssueDetail => ({
  ...issueDetailOf("x"),
  cycle_id,
  updated_at,
});

beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await issueStoreTab();
  now = cycleOf(tab.project, "Now", { status: "current" });
  next = cycleOf(tab.project, "Next", { status: "upcoming" });
  await loadCycles(tab.nerve, tab.app.planning.cycles, [rowOf(now), rowOf(next)], tab.project);
  tab.app.issue.issues.acceptRead(workItem(now.id), tab.app.issue.captureRead());
  tab.nerve.calls.length = 0;
  toasts.length = 0;
  host = document.createElement("div");
  document.body.append(host);
  renderer = createRoot(host);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  host.remove();
  tab.app.dispose();
  vi.useRealTimers();
});

function Sidebar({ isEditable }: { isEditable: boolean }) {
  const issueOperations = useIssueOperations("ws", "A", "x");
  return (
    <IssueDetailsSidebar
      workspaceSlug="ws"
      projectId="A"
      issueId="x"
      issueOperations={issueOperations}
      isEditable={isEditable}
    />
  );
}
const show = (isEditable = true) => act(async () => renderer.render(<Sidebar isEditable={isEditable} />));
const labels = () => [...host.querySelectorAll(".text-tertiary > span")].map((label) => label.textContent);
const cycleButton = () =>
  [...host.querySelectorAll<HTMLButtonElement>("button")].find((button) =>
    button.textContent?.startsWith("common.cycle: ")
  );
const patches = () => tab.nerve.calls.filter((call) => call.method === "PATCH");
/** Opens the cycle's list, answers its read of the project's cycles, and picks the option. */
async function pick(text: string) {
  const at = tab.nerve.calls.length;
  await act(async () => {
    cycleButton()!.click();
    await until(() => tab.nerve.calls.length === at + 1, "the cycles' read");
  });
  const read = tab.nerve.calls[at]!;
  expect([read.method, read.path]).toEqual(["GET", cycleRequests.list(tab.project)]);
  await act(async () => {
    read.answer(json(200, { data: [rowOf(now), rowOf(next)] }));
    await vi.advanceTimersByTimeAsync(20);
  });
  const option = [...document.querySelectorAll<HTMLElement>('[role="option"]')].find(
    (item) => item.textContent === text
  );
  await act(async () => {
    option!.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true, button: 0 }));
    await vi.advanceTimersByTimeAsync(20);
  });
}

it("shows the cycle after the due date while the project's cycles are on, and not while they are off", async () => {
  await show();
  expect(labels()).toEqual([
    "common.state",
    "common.assignees",
    "common.priority",
    "common.order_by.start_date",
    "common.order_by.due_date",
    "common.cycle",
    "common.parent",
    "common.labels",
  ]);
  expect(cycleButton()?.textContent).toBe("common.cycle: Now");
  await act(async () => {
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, cycle_view: false }]);
  });
  expect(labels()).not.toContain("common.cycle");
  expect(cycleButton()).toBeUndefined();
});

it("is read-only while the work item cannot be edited", async () => {
  await show(false);
  expect(cycleButton()?.disabled).toBe(true);
  await act(async () => {
    cycleButton()!.click();
    await vi.advanceTimersByTimeAsync(50);
  });
  expect(document.querySelector('[role="listbox"]')).toBeNull();
  expect(tab.nerve.calls).toEqual([]);
});

it("sends a pick as one PATCH of the cycle alone, and shows the cycle once nerve has it", async () => {
  await show();
  await pick("Next");
  expect(patches().map((call) => [call.path, call.body])).toEqual([["/api/v0/issues/x", { cycle_id: next.id }]]);
  expect(cycleButton()?.textContent, "nerve's value until it answers").toBe("common.cycle: Now");
  await act(async () => {
    patches()[0]!.answer(json(200, workItem(next.id, "2026-10-30T00:00:00Z")));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(cycleButton()?.textContent).toBe("common.cycle: Next");
  expect(patches()).toHaveLength(1);
  expect(toasts).toEqual([]);
});

it("says why nerve refused a pick and keeps showing nerve's cycle", async () => {
  await show();
  await pick("Next");
  await act(async () => {
    patches()[0]!.answer(problem(409, "planning.cycle_completed"));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(toasts.map((toast) => toast.message)).toEqual(["errors.planning_cycle_completed"]);
  expect(cycleButton()?.textContent).toBe("common.cycle: Now");
  expect(patches()).toHaveLength(1);
});

it("sends two picks in a row as two PATCHes, each of the cycle picked", async () => {
  await show();
  await pick("Next");
  await pick("cycle.no_cycle");
  expect(patches(), "the second waits for the first").toHaveLength(1);
  await act(async () => {
    patches()[0]!.answer(json(200, workItem(next.id, "2026-10-30T00:00:00Z")));
    await until(() => patches().length === 2, "the second pick");
  });
  expect(patches().map((call) => call.body)).toEqual([{ cycle_id: next.id }, { cycle_id: null }]);
  await act(async () => {
    patches()[1]!.answer(json(200, workItem(null, "2026-10-31T00:00:00Z")));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(cycleButton()?.textContent).toBe("common.cycle: cycle.no_cycle");
});
