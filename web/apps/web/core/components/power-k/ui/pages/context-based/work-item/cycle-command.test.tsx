// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The work item's command that changes its cycle (P7c spec 2.5, R6; A5): offered, by Shift+C, to who may change the
// work item while its project's cycles are on. Its page reads the project's cycles as it opens and offers No cycle
// first, then the cycles the work item may join (cycleOptions: none ended, none archived), the one it is in marked; an
// ended or an archived cycle it is in shows by its name as its current value, not offered. A pick closes the page: the
// cycle it is in sends nothing, No cycle sends null. The write is the palette's guarded one (mutation-commands.ts), its
// body the change's in the work item's turn. The table of the commands' bodies is commands.test.tsx's; the stores and
// HTTP here are real (issueStoreTab), the page under the palette's host as wrapper.tsx runs one.
import { act, createContext, useContext, useMemo } from "react";
import { createRoot, type Root } from "react-dom/client";
import { Command } from "cmdk";
import { observer } from "mobx-react";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Cycle, IssueDetail, Project } from "@nerve/api-client";
import { PowerKCommandRegistry } from "@/components/power-k/core/registry";
import { ShortcutHandler } from "@/components/power-k/core/shortcut-handler";
import type { TPowerKCommandConfig, TPowerKContext, TPowerKPageType } from "@/components/power-k/core/types";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { StoreContext } from "@/lib/store-context";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { accountOf } from "@/store/member/fake-users";
import { cycleRequests, rowOf } from "@/store/planning/fake-cycles";
import { cycleOf, loadProjects } from "@/store/project/fake-projects";
import { seedAccount } from "@/store/user/fake-account";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { usePowerKWorkItemContextBasedCommands } from "./commands";
import { PowerKWorkItemContextBasedPages } from "./root";

const route = createContext({ workspaceSlug: "ws", workItem: "A-1" });
const auth = vi.hoisted(() => ({ state: { loginId: "cycle-login" } }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: auth, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({
  useSession: () => ({ status: "signed-in", loginId: auth.state.loginId }),
}));
vi.mock("@/lib/store-context", async () => {
  const react = await import("react");
  return { StoreContext: react.createContext(undefined), rootStore: undefined };
});
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", () => ({ useParams: () => useContext(route) }));

const PAGE = "update-work-item-cycle";
let tab: Awaited<ReturnType<typeof issueStoreTab>>, host: HTMLDivElement, view: Root;
/** One SWR cache for the test, as the app keeps one across its pages. */
let cache: Map<string, unknown>;
let commands: TPowerKCommandConfig[], issue: IssueDetail;
/** The pages the palette was asked to open (its context's setActivePage). */
const opened: (TPowerKPageType | null)[] = [];
const ctx: TPowerKContext = {
  params: { workspaceSlug: "ws", workItem: "A-1" },
  activeCommand: null,
  activeContext: "work-item",
  shouldShowContextBasedActions: true,
  setShouldShowContextBasedActions: () => {},
  navigate: () => {},
  closePalette: () => {},
  setActiveCommand: () => {},
  setActivePage: (page) => {
    opened.push(page);
  },
};
const cycleCommand = () => commands.find((item) => item.id === "add_work_item_to_cycle")!;
/** A pick of the page, handed to the command that opened it, as the palette's host does (wrapper.tsx). */
const select = (data: unknown) => {
  const command = cycleCommand();
  return command.type === "change-page" ? command.onSelect(data, ctx) : undefined;
};
const Palette = observer(function Palette({ page }: { page: TPowerKPageType | null }) {
  commands = usePowerKWorkItemContextBasedCommands();
  return (
    <Command shouldFilter={false}>
      <Command.List>
        <PowerKWorkItemContextBasedPages activePage={page} handleSelection={(data) => void select(data)} />
      </Command.List>
    </Command>
  );
});
type Address = { page: TPowerKPageType | null; workItem: string };
function View({ page, workItem }: Address) {
  const value = useMemo(() => ({ workspaceSlug: "ws", workItem }), [workItem]);
  return (
    <SWRConfig value={{ provider: () => cache as never }}>
      <StoreContext.Provider value={tab.app}>
        <route.Provider value={value}>
          <Palette page={page} />
        </route.Provider>
      </StoreContext.Provider>
    </SWRConfig>
  );
}
const render = (page: TPowerKPageType | null = PAGE, workItem = "A-1") =>
  act(async () => view.render(<View page={page} workItem={workItem} />));
const settle = () => act(async () => vi.advanceTimersByTimeAsync(100));
const patches = () => tab.nerve.calls.filter((call) => call.method === "PATCH");
const lists = (archived = false) =>
  tab.nerve.to(cycleRequests.list(tab.project)).filter((call) => (call.query.archived === "true") === archived);
const options = () => [...host.querySelectorAll<HTMLElement>('[role="option"]')];
/** A row of the page: its text, whether it is marked as the work item's cycle, whether it may be picked. */
const rows = () =>
  options().map((option) => ({
    text: option.textContent,
    current: option.lastElementChild?.querySelector("svg") != null,
    offered: option.getAttribute("aria-disabled") !== "true",
  }));
const press = (init: KeyboardEventInit) =>
  document.dispatchEvent(new KeyboardEvent("keydown", { bubbles: true, cancelable: true, ...init }));
const pick = (text: string) => {
  const option = options().find((item) => item.textContent === text)!;
  return act(async () => option.click());
};
/** Project A's cycles: ended, in progress, upcoming, a draft, and an ended one archived. */
const cyclesOfA = () => ({
  done: cycleOf(tab.project, "Done", { status: "completed" }),
  sprint: cycleOf(tab.project, "Sprint", { status: "current" }),
  next: cycleOf(tab.project, "Next", { status: "upcoming" }),
  draft: cycleOf(tab.project, "Draft"),
  old: cycleOf(tab.project, "Old", { status: "completed", archived_at: "2026-10-30T00:00:00Z" }),
});
/** nerve answers the reads of project A's cycles: its list of those not archived, and its archive. */
function listing(list: Cycle[], archive: Cycle[] = []) {
  tab.nerve.respond = (call) => {
    if (call.method === "GET" && call.path === cycleRequests.list(tab.project))
      call.answer(json(200, { data: (call.query.archived === "true" ? archive : list).map((cycle) => rowOf(cycle)) }));
  };
}
/** The caller of projectRole in project A (as project says), whose work item A-1 is in cycleId (null for none). */
async function setup(cycleId: string | null = null, projectRole: 5 | 15 = 15, project: Partial<Project> = {}) {
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 15 }]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    { ...tab.project, member_role: projectRole, guest_view_all_features: false, ...project },
  ]);
  issue = { ...issueDetailOf("x"), created_by_id: "u-actor", cycle_id: cycleId };
  tab.app.issue.issues.acceptRead(issue, tab.app.issue.captureRead());
  tab.nerve.calls.length = 0;
}
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  // oxlint-disable-next-line no-extend-native -- jsdom has no scrolling, which cmdk asks of the item it selects.
  HTMLElement.prototype.scrollIntoView = vi.fn();
  auth.state.loginId = "cycle-login";
  tab = await issueStoreTab();
  await seedAccount(tab.nerve, tab.app.user, accountOf("actor"));
  cache = new Map();
  opened.length = 0;
  toasts.length = 0;
  host = document.createElement("div");
  document.body.append(host);
  view = createRoot(host);
});
afterEach(async () => {
  await act(async () => view.unmount());
  host.remove();
  tab.app.dispose();
  Reflect.deleteProperty(HTMLElement.prototype, "scrollIntoView");
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

it.each([
  ["a member, its cycles on", 15, {}, false, true],
  ["its guest", 5, {}, false, false],
  ["a member, the work item archived", 15, {}, true, false],
  ["a member, its cycles off", 15, { cycle_view: false }, false, false],
] as const)("offers the cycle command to %s", async (_, role, project, archived, offered) => {
  await setup(null, role, project);
  if (archived)
    tab.app.issue.issues.acceptRead({ ...issue, archived_at: "2026-10-30T00:00:00Z" }, tab.app.issue.captureRead());
  await render(null);
  expect(cycleCommand().isVisible(ctx)).toBe(offered);
  expect(cycleCommand().isEnabled(ctx)).toBe(offered);
});

it("Shift+C opens its page through the palette's shortcuts, which C and Cmd+Shift+C do not", async () => {
  await setup();
  await render(null);
  expect(cycleCommand()).toMatchObject({ type: "change-page", page: PAGE, modifierShortcut: "shift+c" });
  expect(cycleCommand().shortcut).toBeUndefined();
  const keys = commands.flatMap((item) => [item.shortcut, item.keySequence, item.modifierShortcut]).filter(Boolean);
  expect(new Set(keys).size).toBe(keys.length);
  const registry = new PowerKCommandRegistry();
  registry.registerMultiple(commands);
  let palettes = 0;
  const handler = new ShortcutHandler(
    registry,
    () => ctx,
    () => {
      palettes++;
    }
  );
  document.addEventListener("keydown", handler.handleKeyDown);
  try {
    press({ key: "c" });
    press({ key: "C", shiftKey: true, metaKey: true });
    expect(opened).toEqual([]);
    press({ key: "C", shiftKey: true });
    expect(opened).toEqual([PAGE]);
    expect(palettes).toBe(1);
  } finally {
    document.removeEventListener("keydown", handler.handleKeyDown);
    handler.destroy();
  }
});

it("reads the project's cycles as its page opens, then offers No cycle first and those it may join", async () => {
  const { done, sprint, next, draft } = cyclesOfA();
  await setup(sprint.id);
  await render(null);
  expect(lists()).toEqual([]);
  await render();
  expect(lists()).toHaveLength(1);
  expect(host.querySelector('[role="status"]')).not.toBeNull();
  await act(async () => lists()[0]!.answer(json(200, { data: [done, sprint, next, draft].map((c) => rowOf(c)) })));
  await settle();
  expect(host.querySelector('[role="status"]')).toBeNull();
  expect(rows()).toEqual([
    { text: "cycle.no_cycle", current: false, offered: true },
    { text: "Sprint", current: true, offered: true },
    { text: "Next", current: false, offered: true },
    { text: "Draft", current: false, offered: true },
  ]);
  expect(lists(true)).toEqual([]);
});

it("says why nerve could not list the project's cycles, waiting no more", async () => {
  await setup();
  await render();
  await act(async () => lists()[0]!.answer(problem(403, "forbidden")));
  await settle();
  expect(host.querySelector('[role="status"]')).toBeNull();
  expect(options()).toEqual([]);
  expect(host.textContent).toBe("errors.forbidden");
});

it.each([
  ["an ended", "done", "Done"],
  ["an archived", "old", "Old, common.archived"],
] as const)(
  "shows %s cycle the work item is in by its name, as its current value, not offered",
  async (_, held, text) => {
    const cycles = cyclesOfA();
    listing([cycles.done, cycles.sprint], [cycles.old]);
    await setup(cycles[held].id);
    await render();
    await settle();
    expect(rows()).toEqual([
      { text, current: true, offered: false },
      { text: "cycle.no_cycle", current: false, offered: true },
      { text: "Sprint", current: false, offered: true },
    ]);
    await pick(text);
    await settle();
    expect(patches()).toEqual([]);
  }
);

it.each([
  ["another cycle", null, "Next", { cycle_id: "c-A-Next" }],
  ["No cycle", "c-A-Sprint", "cycle.no_cycle", { cycle_id: null }],
] as const)("a pick of %s sends exactly it, and closes the page", async (_, held, text, body) => {
  const { sprint, next } = cyclesOfA();
  listing([sprint, next]);
  await setup(held);
  await render();
  await settle();
  expect(cycleCommand().closeOnSelect).toBe(true);
  await pick(text);
  await act(async () => until(() => patches().length === 1, "the cycle's change"));
  expect(patches()[0]).toMatchObject({ path: "/api/v0/issues/x" });
  expect(patches()[0]!.body).toEqual(body);
  await act(async () => patches()[0]!.answer(json(200, { ...issue, ...body, updated_at: "2026-10-30T01:00:00Z" })));
  expect(tab.app.issue.issues.getIssueById("x")).toMatchObject(body);
});

it.each([
  ["its cycle", "c-A-Sprint", "Sprint"],
  ["No cycle, in none", null, "cycle.no_cycle"],
] as const)("a pick of the value it holds, %s, sends nothing", async (_, held, text) => {
  const { sprint, next } = cyclesOfA();
  listing([sprint, next]);
  await setup(held);
  await render();
  await settle();
  await pick(text);
  await settle();
  expect(patches()).toEqual([]);
});

it("says why once the project's cycles are turned off while its page is open, waiting no more", async () => {
  await setup();
  await render();
  expect(host.querySelector('[role="status"]')).not.toBeNull();
  await act(async () =>
    tab.app.projectRoot.project.confirmProject(tab.project, (held) => ({ ...held, cycle_view: false }))
  );
  await settle();
  expect(host.querySelector('[role="status"]')).toBeNull();
  expect(host.textContent).toBe("errors.planning_cycles_disabled");
});

it("an old work item's command cannot change the cycle of the one now at its address", async () => {
  await setup();
  await render(null);
  const old = cycleCommand();
  const other = { ...issueDetailOf("y"), sequence_id: 2 };
  await act(async () => tab.app.issue.issues.acceptRead(other, tab.app.issue.captureRead()));
  await render(null, "A-2");
  await render(null);
  await act(async () => {
    if (old.type === "change-page") void old.onSelect("c-A-Sprint", ctx);
  });
  await settle();
  expect(patches()).toEqual([]);
});

it.each(["session", "archive"])("a queued cycle change is dropped when its %s changes before its turn", async (why) => {
  const { sprint, next } = cyclesOfA();
  listing([sprint, next]);
  await setup();
  await render();
  await settle();
  await pick("Sprint");
  await act(async () => until(() => patches().length === 1, "the first change"));
  await pick("Next");
  if (why === "session") auth.state.loginId = "later-session";
  const archived_at = why === "archive" ? "2026-10-30T01:00:00Z" : null;
  const answer = { ...issue, cycle_id: sprint.id, archived_at, updated_at: "2026-10-30T01:00:00Z" };
  await act(async () => patches()[0]!.answer(json(200, answer)));
  await settle();
  expect(patches()).toHaveLength(1);
  expect(toasts).toEqual([]);
});

it("says nerve's refusal of a cycle change, the work item still in no cycle", async () => {
  const { sprint } = cyclesOfA();
  listing([sprint]);
  await setup();
  await render();
  await settle();
  await pick("Sprint");
  await act(async () => until(() => patches().length === 1, "the refused change"));
  await act(async () => patches()[0]!.answer(problem(409, "planning.cycle_completed")));
  expect(tab.app.issue.issues.getIssueById("x")?.cycle_id).toBeNull();
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.planning_cycle_completed" }]);
});
