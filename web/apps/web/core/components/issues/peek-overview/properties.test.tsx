// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// M6 P7a spec 2.3, 2.5, R3, R11, R14: the cycle and the modules a work item is in, in its preview's properties,
// against the session's stores and a fake nerve, as in its detail's sidebar: each shown while its project's domain is
// on, read-only while the preview is, one PATCH a pick (a module's as one addition to the set held), nerve's refusal
// said. An archived cycle or module it is in is named once the project's lists are read, with the caption Archived,
// the lists read once for the sidebar and the preview a narrow detail shows.
import { act } from "react";
import type { ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Cycle, IssueDetail, Module } from "@nerve/api-client";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { cycleRequests, loadCycles, rowOf } from "@/store/planning/fake-cycles";
import { loadArchivedModules, loadModules, moduleRequests, moduleRowOf } from "@/store/planning/fake-modules";
import { cycleOf, loadProjects, moduleOf } from "@/store/project/fake-projects";
import { IssueDetailsSidebar } from "../issue-detail/sidebar";
import { useIssueOperations } from "../issue-detail/use-issue-operations";
import { PeekOverviewProperties } from "./properties";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "preview" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "preview" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "ws" }) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-cycles", () => ({ useCycles: () => tab.app.planning.cycles }));
vi.mock("@/hooks/store/use-modules", () => ({ useModules: () => tab.app.planning.modules }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => tab.app.user.permission }));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => tab.app.state }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: () => undefined }) }));
// The preview's other properties have their own tests; here they are absent.
vi.mock("@/components/dropdowns/state/dropdown", () => ({ StateDropdown: () => null }));
vi.mock("@/components/dropdowns/member/dropdown", () => ({ MemberDropdown: () => null }));
vi.mock("@/components/dropdowns/priority", () => ({ PriorityDropdown: () => null }));
vi.mock("@/components/dropdowns/date", () => ({ DateDropdown: () => null }));
vi.mock("@/components/issues/parent-select-root", () => ({ IssueParentSelectRoot: () => null }));
vi.mock("../issue-detail/label", () => ({ IssueLabel: () => null }));

let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let renderer: ReturnType<typeof createRoot>;
let now: Cycle, next: Cycle, shelved: Cycle;
let alpha: Module, beta: Module, gamma: Module;
const workItem = (
  cycle_id: string | null,
  updated_at = "2026-10-29T00:00:00Z",
  module_ids: string[] = []
): IssueDetail => ({
  ...issueDetailOf("x"),
  cycle_id,
  module_ids,
  updated_at,
});

beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await issueStoreTab();
  now = cycleOf(tab.project, "Now", { status: "current" });
  next = cycleOf(tab.project, "Next", { status: "upcoming" });
  shelved = cycleOf(tab.project, "Shelved", { status: "completed", archived_at: "2026-10-20T00:00:00Z" });
  alpha = moduleOf(tab.project, "Alpha", { status: "in-progress" });
  beta = moduleOf(tab.project, "Beta");
  gamma = moduleOf(tab.project, "Gamma", { status: "completed", archived_at: "2026-10-20T00:00:00Z" });
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

function Preview({ disabled = false }: { disabled?: boolean }) {
  const issueOperations = useIssueOperations("ws", "A", "x");
  return (
    <PeekOverviewProperties
      workspaceSlug="ws"
      projectId="A"
      issueId="x"
      issueOperations={issueOperations}
      disabled={disabled}
    />
  );
}
function Sidebar() {
  const issueOperations = useIssueOperations("ws", "A", "x");
  return (
    <IssueDetailsSidebar workspaceSlug="ws" projectId="A" issueId="x" issueOperations={issueOperations} isEditable />
  );
}
const render = (node: ReactNode) =>
  act(async () => renderer.render(<SWRConfig value={{ provider: () => new Map() }}>{node}</SWRConfig>));
/** The work item in Now, of the project whose cycles the store has listed, in its preview. */
async function show(disabled = false) {
  await loadCycles(tab.nerve, tab.app.planning.cycles, [rowOf(now), rowOf(next)], tab.project);
  tab.app.issue.issues.acceptRead(workItem(now.id), tab.app.issue.captureRead());
  tab.nerve.calls.length = 0;
  await render(<Preview disabled={disabled} />);
}
const labels = () => [...host.querySelectorAll(".text-tertiary > span")].map((label) => label.textContent);
const cycleButtons = () =>
  [...host.querySelectorAll<HTMLButtonElement>("button")].filter((button) =>
    button.textContent?.startsWith("common.cycle: ")
  );
const cycleButton = () => cycleButtons()[0];
const patches = () => tab.nerve.calls.filter((call) => call.method === "PATCH");
/** The reads of the project's cycles so far, by their query: {} for its list, archived=true for its archive. */
const reads = () =>
  tab.nerve.calls
    .filter((call) => call.method === "GET" && call.path === cycleRequests.list(tab.project))
    .map((call) => call.query);
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
    "common.modules",
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

it("is read-only while the preview is", async () => {
  await show(true);
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

it("names an archived cycle it is in, with its caption, once the project's lists are read, once for the sidebar and the preview", async () => {
  tab.app.issue.issues.acceptRead(workItem(shelved.id), tab.app.issue.captureRead());
  tab.nerve.calls.length = 0;
  await render(
    <>
      <Sidebar />
      <Preview />
    </>
  );
  expect(
    cycleButtons().map((button) => button.textContent),
    "no name yet"
  ).toEqual(["common.cycle: ", "common.cycle: "]);
  await act(async () => until(() => reads().length === 1, "the project's list"));
  await act(async () => vi.advanceTimersByTimeAsync(50));
  expect(reads(), "its list first, once").toEqual([{}]);
  await act(async () => {
    tab.nerve.calls
      .find((call) => call.path === cycleRequests.list(tab.project))!
      .answer(json(200, { data: [rowOf(now), rowOf(next)] }));
    await until(() => reads().length === 2, "the project's archive");
  });
  await act(async () => vi.advanceTimersByTimeAsync(50));
  expect(reads(), "then its archive, once").toEqual([{}, { archived: "true" }]);
  await act(async () => {
    tab.nerve.calls
      .filter((call) => call.path === cycleRequests.list(tab.project))[1]!
      .answer(json(200, { data: [rowOf(shelved)] }));
    await vi.advanceTimersByTimeAsync(50);
  });
  expect(cycleButtons().map((button) => button.textContent)).toEqual([
    "common.cycle: Shelved, common.archived",
    "common.cycle: Shelved, common.archived",
  ]);
  expect(reads()).toHaveLength(2);
});

const moduleButtons = () =>
  [...host.querySelectorAll<HTMLButtonElement>("button")].filter((button) =>
    button.textContent?.startsWith("common.modules: ")
  );
const moduleButton = () => moduleButtons()[0];
/** The reads of the project's modules so far, by their query: {} for its list, archived=true for its archive. */
const moduleReads = () =>
  tab.nerve.calls.filter((call) => call.method === "GET" && call.path === moduleRequests.list(tab.project));
/** The work item in the modules, in its preview, the project's lists of them read, the archived Gamma among them. */
async function showInModules(moduleIds: string[], disabled = false) {
  await loadModules(
    tab.nerve,
    tab.app.planning.modules,
    [alpha, beta].map((item) => moduleRowOf(item)),
    tab.project
  );
  await loadArchivedModules(tab.nerve, tab.app.planning.modules, [moduleRowOf(gamma)], tab.project);
  tab.app.issue.issues.acceptRead(workItem(null, undefined, moduleIds), tab.app.issue.captureRead());
  tab.nerve.calls.length = 0;
  await render(<Preview disabled={disabled} />);
}

it("shows the modules before the cycle while the project's modules are on, and not while they are off", async () => {
  await showInModules([alpha.id]);
  expect(moduleButton()?.textContent).toBe("common.modules: Alpha");
  await act(async () => {
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, module_view: false }]);
  });
  expect(labels()).not.toContain("common.modules");
  expect(labels()).toContain("common.cycle");
  expect(moduleButton()).toBeUndefined();
});

it("keeps the modules read-only while the preview is", async () => {
  await showInModules([alpha.id, gamma.id], true);
  expect(moduleButton()?.disabled).toBe(true);
  await act(async () => {
    moduleButton()!.click();
    await vi.advanceTimersByTimeAsync(50);
  });
  expect(document.querySelector('[role="listbox"]')).toBeNull();
  expect(tab.nerve.calls).toEqual([]);
});

it("sends a module picked as one PATCH of the set the work item holds with it added", async () => {
  await showInModules([alpha.id, gamma.id]);
  await act(async () => {
    moduleButton()!.click();
    await until(() => moduleReads().length === 1, "the modules' read");
  });
  await act(async () => {
    moduleReads()[0]!.answer(json(200, { data: [alpha, beta].map((item) => moduleRowOf(item)) }));
    await vi.advanceTimersByTimeAsync(20);
  });
  const option = [...document.querySelectorAll<HTMLElement>('[role="option"]')].find(
    (item) => item.textContent === "Beta"
  );
  await act(async () => {
    option!.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true, button: 0 }));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(patches().map((call) => [call.path, call.body])).toEqual([
    ["/api/v0/issues/x", { module_ids: [alpha.id, gamma.id, beta.id] }],
  ]);
  await act(async () => {
    patches()[0]!.answer(json(200, workItem(null, "2026-10-30T00:00:00Z", [alpha.id, gamma.id, beta.id])));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(moduleButton()?.textContent).toBe("common.modules: module.count: Alpha; Gamma, common.archived; Beta");
  expect(patches()).toHaveLength(1);
  expect(toasts).toEqual([]);
});

it("names an archived module it is in, with its caption, once the project's lists are read, once for the sidebar and the preview", async () => {
  tab.app.issue.issues.acceptRead(workItem(null, undefined, [gamma.id]), tab.app.issue.captureRead());
  tab.nerve.calls.length = 0;
  await render(
    <>
      <Sidebar />
      <Preview />
    </>
  );
  expect(
    moduleButtons().map((button) => button.textContent),
    "no name yet"
  ).toEqual(["common.modules: ", "common.modules: "]);
  await act(async () => until(() => moduleReads().length === 1, "the project's list"));
  await act(async () => vi.advanceTimersByTimeAsync(50));
  expect(
    moduleReads().map((call) => call.query),
    "its list first, once"
  ).toEqual([{}]);
  await act(async () => {
    moduleReads()[0]!.answer(json(200, { data: [alpha, beta].map((item) => moduleRowOf(item)) }));
    await until(() => moduleReads().length === 2, "the project's archive");
  });
  await act(async () => vi.advanceTimersByTimeAsync(50));
  expect(
    moduleReads().map((call) => call.query),
    "then its archive, once"
  ).toEqual([{}, { archived: "true" }]);
  await act(async () => {
    moduleReads()[1]!.answer(json(200, { data: [moduleRowOf(gamma)] }));
    await vi.advanceTimersByTimeAsync(50);
  });
  expect(moduleButtons().map((button) => button.textContent)).toEqual([
    "common.modules: Gamma, common.archived",
    "common.modules: Gamma, common.archived",
  ]);
  expect(moduleReads()).toHaveLength(2);
});
