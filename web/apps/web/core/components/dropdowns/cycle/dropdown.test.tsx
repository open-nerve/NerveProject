// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// M6 P7a spec 2.3, 2.4 (M2 closeout 4, M3 closeout 3): a work item's cycle dropdown, against the session's stores and a
// fake nerve. Its options are read as it opens and are the cycles the work item may join; its value is the cycle it is
// in, by name, an ended or an archived one too.
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Cycle, Project } from "@nerve/api-client";
import { CYCLE_GROUP_COLORS } from "@nerve/propel/icons";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueStoreTab } from "@/store/issue/fake-issues";
import { cycleRequests, loadArchivedCycles, rowOf } from "@/store/planning/fake-cycles";
import { cycleOf, loadProjects, projectOf } from "@/store/project/fake-projects";
import type { TButtonVariants } from "@/components/dropdowns/types";
import { CycleDropdown } from "./dropdown";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "cycles" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "ws" }) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-cycles", () => ({ useCycles: () => tab.app.planning.cycles }));

let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let renderer: ReturnType<typeof createRoot>;
let other: Project;
let now: Cycle, draft: Cycle, next: Cycle, ended: Cycle, shelved: Cycle, elsewhere: Cycle;
const changed = vi.fn();

beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await issueStoreTab();
  other = projectOf("B", tab.workspace.id, { id: "B" });
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project, other]);
  tab.nerve.calls.length = 0;
  now = cycleOf(tab.project, "Now", { status: "current" });
  draft = cycleOf(tab.project, "Draft");
  next = cycleOf(tab.project, "Next", { status: "upcoming" });
  ended = cycleOf(tab.project, "Ended", { status: "completed" });
  shelved = cycleOf(tab.project, "Shelved", { status: "upcoming", archived_at: "2026-10-20T00:00:00Z" });
  elsewhere = cycleOf(other, "Elsewhere", { status: "current" });
  changed.mockClear();
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

/** The cycles nerve lists for the project, not archived, in its order. */
const listed = () => [ended, now, draft, next].map((cycle) => rowOf(cycle));
const button = () => host.querySelector<HTMLButtonElement>("button")!;
const list = () => document.querySelector('[role="listbox"]');
const options = () => [...document.querySelectorAll('[role="option"]')].map((option) => option.textContent);
const alert = () => document.querySelector('[role="alert"]');

async function show(
  props: {
    value?: string | null;
    projectId?: string;
    disabled?: boolean;
    issueName?: string;
    buttonVariant?: TButtonVariants;
  } = {}
) {
  const { value = null, projectId = tab.project.id, disabled = false, issueName } = props;
  await act(async () =>
    renderer.render(
      <CycleDropdown
        projectId={projectId}
        value={value}
        onChange={changed}
        disabled={disabled}
        issueName={issueName}
        buttonVariant={props.buttonVariant ?? "border-with-text"}
      />
    )
  );
}
/** Opens the list, which reads the project's cycles: the read, which the caller answers, or not. */
async function open(project: Project = tab.project) {
  const at = tab.nerve.calls.length;
  await act(async () => {
    button().click();
    await until(() => tab.nerve.calls.length === at + 1, "the cycles' read");
  });
  const read = tab.nerve.calls[at]!;
  expect([read.method, read.path, read.query]).toEqual(["GET", cycleRequests.list(project), {}]);
  return read;
}
async function answer(read: { answer: (response: Response) => void }, response: Response) {
  await act(async () => {
    read.answer(response);
    await vi.advanceTimersByTimeAsync(20);
  });
}
async function openRead(rows = listed()) {
  await answer(await open(), json(200, { data: rows }));
}
/** Picks the option as a pointer does: Headless UI's option selects on the left button's mousedown. */
async function pick(text: string) {
  const option = [...document.querySelectorAll<HTMLElement>('[role="option"]')].find(
    (item) => item.textContent === text
  );
  expect(option, text).toBeDefined();
  await act(async () => {
    option!.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true, button: 0 }));
    await vi.advanceTimersByTimeAsync(20);
  });
}

it("reads the project's cycles each time it opens and lists No cycle first, then those a work item may join, in nerve's order", async () => {
  await show();
  const read = await open();
  expect(options(), "loading").toEqual([]);
  expect(list()?.textContent).toContain("common.loading");
  await answer(read, json(200, { data: listed() }));
  expect(options()).toEqual(["cycle.no_cycle", "Now", "Draft", "Next"]);
  await act(async () => button().click());
  expect(list()).toBeNull();
  await openRead([...listed(), rowOf(cycleOf(tab.project, "Later", { status: "upcoming" }))]);
  expect(options()).toEqual(["cycle.no_cycle", "Now", "Draft", "Next", "Later"]);
});

it("gives the cycle picked and closes; No cycle gives null; the cycle it holds gives nothing", async () => {
  await show({ value: now.id });
  await openRead();
  await pick("Draft");
  expect(changed.mock.calls).toEqual([[draft.id]]);
  expect(list()).toBeNull();
  await openRead();
  await pick("cycle.no_cycle");
  expect(changed.mock.calls).toEqual([[draft.id], [null]]);
  expect(list()).toBeNull();
  await openRead();
  await pick("Now");
  expect(changed.mock.calls, "the value it holds is no change").toEqual([[draft.id], [null]]);
  expect(list()).toBeNull();
  await show({ value: null });
  await openRead();
  await pick("cycle.no_cycle");
  expect(changed.mock.calls, "nor is No cycle while it is in none").toEqual([[draft.id], [null]]);
});

it("shows an ended or an archived cycle it is in on its button, by its name and status, the archived one captioned, and does not list it", async () => {
  await loadArchivedCycles(tab.nerve, tab.app.planning.cycles, [rowOf(shelved)], tab.project);
  tab.nerve.calls.length = 0;
  await show({ value: ended.id });
  await openRead();
  expect(button().textContent).toBe("common.cycle: Ended");
  expect(button().querySelector(`svg[color="${CYCLE_GROUP_COLORS.completed}"]`), "the ended status").not.toBeNull();
  expect(options()).toEqual(["cycle.no_cycle", "Now", "Draft", "Next"]);
  await act(async () => button().click());
  await show({ value: shelved.id });
  expect(button().textContent).toBe("common.cycle: Shelved, common.archived");
  await openRead();
  expect(options()).not.toContain("Shelved");
  await act(async () => button().click());
  await show({ value: shelved.id, buttonVariant: "border-without-text" });
  expect(button().textContent, "named the same without its text").toBe("common.cycle: Shelved, common.archived");
});

it("is named by its field and value, and by the row's work item when it has one", async () => {
  await show();
  expect(button().textContent).toBe("common.cycle: cycle.no_cycle");
  await openRead();
  await act(async () => button().click());
  await show({ value: next.id, issueName: "Fix the login" });
  expect(button().textContent).toBe("common.cycle: Next, Fix the login");
});

it("shows a refused read at the list's foot and reads again on retry", async () => {
  await show();
  await answer(await open(), problem(403, "forbidden"));
  expect(alert()?.textContent).toContain("errors.forbidden");
  expect(options()).toEqual([]);
  const at = tab.nerve.calls.length;
  await act(async () => {
    [...document.querySelectorAll("button")].find((item) => item.textContent === "common.retry")!.click();
    await until(() => tab.nerve.calls.length === at + 1, "the retry");
  });
  expect([tab.nerve.calls[at]!.method, tab.nerve.calls[at]!.path]).toEqual(["GET", cycleRequests.list(tab.project)]);
  await answer(tab.nerve.calls[at]!, json(200, { data: listed() }));
  expect(alert()).toBeNull();
  expect(options()).toEqual(["cycle.no_cycle", "Now", "Draft", "Next"]);
});

it("closes as its project changes, opens next on the new project's cycles, and the old project's late answer shows nothing there", async () => {
  await show();
  const late = await open();
  expect(button().querySelector('[role="status"]'), "the read in flight").not.toBeNull();
  await show({ projectId: other.id });
  expect(list(), "the old project's list closes").toBeNull();
  expect(button().querySelector('[role="status"]'), "the old project's read is not the new one's").toBeNull();
  await answer(late, problem(403, "forbidden"));
  const read = await open(other);
  expect(alert()).toBeNull();
  await answer(read, json(200, { data: [rowOf(elsewhere)] }));
  expect(options()).toEqual(["cycle.no_cycle", "Elsewhere"]);
  expect(alert()).toBeNull();
});

it("does not open while disabled", async () => {
  await show({ value: now.id, disabled: true });
  expect(button().disabled).toBe(true);
  await act(async () => {
    button().click();
    await vi.advanceTimersByTimeAsync(50);
  });
  expect(list()).toBeNull();
  expect(tab.nerve.calls).toEqual([]);
  expect(changed).not.toHaveBeenCalled();
});

it("has one native trigger, its search takes the focus, and Escape closes it back to the trigger", async () => {
  await show();
  expect(host.querySelectorAll("button button")).toHaveLength(0);
  await openRead();
  const search = document.querySelector<HTMLInputElement>('input[role="combobox"]')!;
  expect(document.activeElement).toBe(search);
  await act(async () => {
    search.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(list()).toBeNull();
  expect(document.activeElement).toBe(button());
});
