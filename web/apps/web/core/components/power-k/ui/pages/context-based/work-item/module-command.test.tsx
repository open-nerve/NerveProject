// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The work item's command that changes its modules (P7c spec 2.5, R6; A6): offered, by Shift+M, to who may change the
// work item while its project's modules are on. Its page reads the project's modules as it opens and offers those the
// work item may join (moduleOptions: whatever their status, none archived, in nerve's order), then the archived ones
// it is in, captioned, only to be taken out; those it is in ticked as nerve confirmed them. Each pick adds the work
// item to one module, or takes it out of one, as the page showed it, the page staying open; the write is the palette's
// guarded one (writes.ts), its body the set the change makes of the set held in the work item's turn, and its type
// takes no whole set. The table of the commands' bodies is commands.test.tsx's; the stores and HTTP here are real
// (issueStoreTab), the page under the palette's host as wrapper.tsx runs one.
import { act, createContext, useContext, useMemo } from "react";
import { createRoot, type Root } from "react-dom/client";
import { Command } from "cmdk";
import { observer } from "mobx-react";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Issue, IssueDetail, Module, Project } from "@nerve/api-client";
import { PowerKCommandRegistry } from "@/components/power-k/core/registry";
import { ShortcutHandler } from "@/components/power-k/core/shortcut-handler";
import type { TPowerKCommandConfig, TPowerKContext, TPowerKPageType } from "@/components/power-k/core/types";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { StoreContext } from "@/lib/store-context";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { accountOf } from "@/store/member/fake-users";
import { moduleRequests, moduleRowOf } from "@/store/planning/fake-modules";
import { loadProjects, moduleOf } from "@/store/project/fake-projects";
import { seedAccount } from "@/store/user/fake-account";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { usePowerKWorkItemContextBasedCommands } from "./commands";
import { PowerKWorkItemContextBasedPages } from "./root";
import type { useWorkItemWrites } from "./writes";

const route = createContext({ workspaceSlug: "ws", workItem: "A-1" });
const auth = vi.hoisted(() => ({ state: { loginId: "module-login" } }));
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

const PAGE = "update-work-item-module";
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
const moduleCommand = () => commands.find((item) => item.id === "add_work_item_to_modules")!;
/** A pick of the page, handed to the command that opened it, as the palette's host does (wrapper.tsx). */
const select = (data: unknown) => {
  const command = moduleCommand();
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
  tab.nerve.to(moduleRequests.list(tab.project)).filter((call) => (call.query.archived === "true") === archived);
const options = () => [...host.querySelectorAll<HTMLElement>('[role="option"]')];
/** The page's rows: each one's text, and whether it is ticked, as a module the work item is in. */
const rows = () =>
  options().map((option) => ({
    text: option.textContent,
    ticked: option.lastElementChild?.querySelector("svg") != null,
  }));
/** Whether the page's row of that text is ticked. */
const tickedOf = (text: string) => rows().find((row) => row.text === text)?.ticked;
const press = (init: KeyboardEventInit) =>
  document.dispatchEvent(new KeyboardEvent("keydown", { bubbles: true, cancelable: true, ...init }));
const pick = (text: string) => {
  const option = options().find((item) => item.textContent === text)!;
  return act(async () => option.click());
};
/** nerve answers a write of the work item, its modules now these. */
const confirm = (index: number, module_ids: string[], fields: Partial<Issue> = {}) =>
  act(async () =>
    patches()[index]!.answer(
      json(200, { ...issue, module_ids, updated_at: `2026-10-30T0${index + 1}:00:00Z`, ...fields })
    )
  );
/** Project A's modules: one of each status, and two archived. */
const modulesOfA = () => ({
  alpha: moduleOf(tab.project, "Alpha"),
  beta: moduleOf(tab.project, "Beta", { status: "in-progress" }),
  gamma: moduleOf(tab.project, "Gamma", { status: "completed" }),
  delta: moduleOf(tab.project, "Delta", { status: "cancelled" }),
  old: moduleOf(tab.project, "Old", { status: "completed", archived_at: "2026-10-30T00:00:00Z" }),
  gone: moduleOf(tab.project, "Gone", { archived_at: "2026-10-30T00:00:00Z" }),
});
/** nerve answers the reads of project A's modules: its list of those not archived, and its archive. */
function listing(list: Module[], archive: Module[] = []) {
  tab.nerve.respond = (call) => {
    if (call.method === "GET" && call.path === moduleRequests.list(tab.project))
      call.answer(json(200, { data: (call.query.archived === "true" ? archive : list).map((m) => moduleRowOf(m)) }));
  };
}
/** The caller of projectRole in project A (as project says), whose work item A-1 is in moduleIds. */
async function setup(moduleIds: string[] = [], projectRole: 5 | 15 = 15, project: Partial<Project> = {}) {
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 15 }]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    { ...tab.project, member_role: projectRole, guest_view_all_features: false, ...project },
  ]);
  issue = { ...issueDetailOf("x"), created_by_id: "u-actor", module_ids: moduleIds };
  tab.app.issue.issues.acceptRead(issue, tab.app.issue.captureRead());
  tab.nerve.calls.length = 0;
}
/** The page open on the work item in moduleIds, nerve answering its reads with project A's modules. */
async function opening(moduleIds: string[]) {
  const all = modulesOfA();
  listing([all.alpha, all.beta, all.gamma, all.delta], [all.old, all.gone]);
  await setup(moduleIds);
  await render();
  await settle();
  return all;
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
  auth.state.loginId = "module-login";
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
  ["a member, its modules on", 15, {}, false, true],
  ["its guest", 5, {}, false, false],
  ["a member, the work item archived", 15, {}, true, false],
  ["a member, its modules off", 15, { module_view: false }, false, false],
  ["a member, its cycles off", 15, { cycle_view: false }, false, true],
] as const)("offers the modules command to %s", async (_, role, project, archived, offered) => {
  await setup([], role, project);
  if (archived)
    tab.app.issue.issues.acceptRead({ ...issue, archived_at: "2026-10-30T00:00:00Z" }, tab.app.issue.captureRead());
  await render(null);
  expect(moduleCommand().isVisible(ctx)).toBe(offered);
  expect(moduleCommand().isEnabled(ctx)).toBe(offered);
});

it("Shift+M opens its page through the palette's shortcuts, which M and Cmd+Shift+M do not", async () => {
  await setup();
  await render(null);
  expect(moduleCommand()).toMatchObject({ type: "change-page", page: PAGE, modifierShortcut: "shift+m" });
  expect(moduleCommand().shortcut).toBeUndefined();
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
    press({ key: "m" });
    press({ key: "M", shiftKey: true, metaKey: true });
    expect(opened).toEqual([]);
    press({ key: "M", shiftKey: true });
    expect(opened).toEqual([PAGE]);
    expect(palettes).toBe(1);
  } finally {
    document.removeEventListener("keydown", handler.handleKeyDown);
    handler.destroy();
  }
});

it("reads the project's modules as its page opens, then offers each it may join, whatever its status", async () => {
  const { alpha, beta, gamma, delta } = modulesOfA();
  await setup([beta.id]);
  await render(null);
  expect(lists()).toEqual([]);
  await render();
  expect(lists()).toHaveLength(1);
  expect(host.querySelector('[role="status"]')).not.toBeNull();
  await act(async () =>
    lists()[0]!.answer(json(200, { data: [alpha, beta, gamma, delta].map((m) => moduleRowOf(m)) }))
  );
  await settle();
  expect(host.querySelector('[role="status"]')).toBeNull();
  expect(rows()).toEqual([
    { text: "Alpha", ticked: false },
    { text: "Beta", ticked: true },
    { text: "Gamma", ticked: false },
    { text: "Delta", ticked: false },
  ]);
  expect(lists(true)).toEqual([]);
});

it("says why nerve could not list the project's modules, waiting no more", async () => {
  await setup();
  await render();
  await act(async () => lists()[0]!.answer(problem(403, "forbidden")));
  await settle();
  expect(host.querySelector('[role="status"]')).toBeNull();
  expect(options()).toEqual([]);
  expect(host.textContent).toBe("errors.forbidden");
});

it("lists an archived module the work item is in after the options, by its name, ticked, to be taken out", async () => {
  const { beta } = await opening(["m-A-Old", "m-A-Beta"]);
  expect(lists(true)).toHaveLength(1);
  expect(rows()).toEqual([
    { text: "Alpha", ticked: false },
    { text: "Beta", ticked: true },
    { text: "Gamma", ticked: false },
    { text: "Delta", ticked: false },
    { text: "Old, common.archived", ticked: true },
  ]);
  await pick("Old, common.archived");
  await act(async () => until(() => patches().length === 1, "the module's removal"));
  expect(patches()[0]).toMatchObject({ path: "/api/v0/issues/x" });
  expect(patches()[0]!.body).toEqual({ module_ids: [beta.id] });
  await confirm(0, [beta.id]);
  expect(rows().map((row) => row.text)).toEqual(["Alpha", "Beta", "Gamma", "Delta"]);
});

it.each([
  ["adds it to a module it is not in, after those it is in", ["m-A-Gamma"], false, ["m-A-Gamma", "m-A-Alpha"]],
  ["takes it out of a module it is in", ["m-A-Alpha", "m-A-Gamma"], true, ["m-A-Gamma"]],
] as const)("a pick of Alpha %s, the page staying open, its tick nerve's", async (_, held, ticked, after) => {
  await opening([...held]);
  expect(moduleCommand().closeOnSelect).toBe(false);
  expect(tickedOf("Alpha")).toBe(ticked);
  await pick("Alpha");
  await act(async () => until(() => patches().length === 1, "the module's change"));
  expect(patches()[0]!.body).toEqual({ module_ids: after });
  expect(tickedOf("Alpha")).toBe(ticked);
  await confirm(0, [...after]);
  expect(tab.app.issue.issues.getIssueById("x")?.module_ids).toEqual(after);
  expect(tickedOf("Alpha")).toBe(!ticked);
  expect(options()).toHaveLength(4);
});

it("a pick queued behind another is made in its turn into the set the first one left", async () => {
  await opening(["m-A-Gamma"]);
  await pick("Alpha");
  await act(async () => until(() => patches().length === 1, "the first change"));
  await pick("Beta");
  expect(patches()).toHaveLength(1);
  await confirm(0, ["m-A-Gamma", "m-A-Alpha"]);
  await act(async () => until(() => patches().length === 2, "the queued change"));
  expect(patches()[1]!.body).toEqual({ module_ids: ["m-A-Gamma", "m-A-Alpha", "m-A-Beta"] });
});

it("a pick is decided by the ticks nerve confirmed: one not ticked yet is added again, not taken out", async () => {
  await opening([]);
  await pick("Alpha");
  await act(async () => until(() => patches().length === 1, "the first addition"));
  await pick("Alpha");
  await confirm(0, ["m-A-Alpha"]);
  await act(async () => until(() => patches().length === 2, "the second addition"));
  expect(patches()[1]!.body).toEqual({ module_ids: ["m-A-Alpha"] });
});

it("says why once the project's modules are turned off while its page is open, waiting no more", async () => {
  await setup();
  await render();
  expect(host.querySelector('[role="status"]')).not.toBeNull();
  await act(async () =>
    tab.app.projectRoot.project.confirmProject(tab.project, (held) => ({ ...held, module_view: false }))
  );
  await settle();
  expect(host.querySelector('[role="status"]')).toBeNull();
  expect(host.textContent).toBe("errors.planning_modules_disabled");
});

it("an old work item's command cannot change the modules of the one now at its address", async () => {
  await setup();
  await render(null);
  const old = moduleCommand();
  const other = { ...issueDetailOf("y"), sequence_id: 2 };
  await act(async () => tab.app.issue.issues.acceptRead(other, tab.app.issue.captureRead()));
  await render(null, "A-2");
  await render(null);
  await act(async () => {
    if (old.type === "change-page") void old.onSelect("m-A-Alpha", ctx);
  });
  await settle();
  expect(patches()).toEqual([]);
});

it.each(["session", "archive"])(
  "a queued module change is dropped when its %s changes before its turn",
  async (why) => {
    await opening([]);
    await pick("Alpha");
    await act(async () => until(() => patches().length === 1, "the first change"));
    await pick("Beta");
    if (why === "session") auth.state.loginId = "later-session";
    await confirm(0, ["m-A-Alpha"], { archived_at: why === "archive" ? "2026-10-30T01:00:00Z" : null });
    await settle();
    expect(patches()).toHaveLength(1);
    expect(toasts).toEqual([]);
  }
);

it("says nerve's refusal of a module change, the work item still in none", async () => {
  await opening([]);
  await pick("Alpha");
  await act(async () => until(() => patches().length === 1, "the refused change"));
  await act(async () => patches()[0]!.answer(problem(409, "planning.archived")));
  expect(tab.app.issue.issues.getIssueById("x")?.module_ids).toEqual([]);
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.planning_archived" }]);
});

/** Never called: the command's write (writes.ts), given the modules as a whole set and as a change. */
function changes(write: ReturnType<typeof useWorkItemWrites>, held: Issue) {
  return [
    // @ts-expect-error the command's change cannot give the modules as a whole set
    write.change({ module_ids: [...held.module_ids, "m-A-Alpha"] }),
    // @ts-expect-error nor as the work item's fields, which carry it
    write.change(held),
    write.change({ modules: { add: ["m-A-Alpha"] } }),
    write.change({ modules: { remove: held.module_ids.slice(0, 1) } }),
  ];
}

it("the command's write takes the modules as one addition or removal, never as a whole set", () => {
  expect(changes).toBeTypeOf("function");
});
