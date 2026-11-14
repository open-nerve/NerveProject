// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The filters by a work item's cycle and its modules (P7b spec 2.6, R7, R8), through the filters' wrapper and its real
// config, against the session's stores and a fake nerve: the project's cycles and modules that are not archived, ended
// ones too, after None; a value an applied condition holds that the list does not give (an archived one) by its name
// and caption once usePlanningNames reads the project's archive; never the page's own; none offered while the project
// has the domain turned off, an applied condition kept and removable; the lists those the project's wrapper reads, the
// filters reading none of their own.
import { act } from "react";
import type { ReactElement, ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { observer } from "mobx-react";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@nerve/constants";
import type { IWorkItemFilterInstance } from "@nerve/shared-state";
import {
  COLLECTION_OPERATOR,
  EIssuesStoreType,
  FILTER_FIELD_TYPE,
  LOGICAL_OPERATOR,
  type TWorkItemFilterExpression,
  type TWorkItemFilterProperty,
} from "@nerve/types";
import { FilterItem } from "@/components/rich-filters/filter-item/root";
import { WorkItemFiltersHOC } from "@/components/work-item-filters/filters-hoc/base";
import { answered, json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toProjectFilters } from "@/lib/issue-filters";
import { StoreContext } from "@/lib/store-context";
import { useSessionSWR } from "@/lib/use-session-swr";
import { issueStoreTab, requestedIssue } from "@/store/issue/fake-issues";
import { cycleRequests, loadCycles, rowOf } from "@/store/planning/fake-cycles";
import { loadModules, moduleRequests, moduleRowOf } from "@/store/planning/fake-modules";
import { cycleOf, loadProjects, moduleOf, preferencesOf } from "@/store/project/fake-projects";

const bound = vi.hoisted(() => ({ app: undefined as unknown }));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext(undefined),
  get rootStore() {
    return bound.app;
  },
}));
vi.mock("@/lib/auth/api-client", () => ({
  tokenManager: { state: { loginId: "planning-filters" } },
  publicClient: {},
}));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "planning-filters" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/hooks/use-profile-image", () => import("@/lib/fake-profile-image"));

class NoObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let view: ReturnType<typeof createRoot>;
let changes: TWorkItemFilterExpression[];
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  vi.stubGlobal("ResizeObserver", NoObserver);
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  tab = await issueStoreTab();
  bound.app = tab.app;
  changes = [];
  host = document.createElement("div");
  document.body.append(host);
  view = createRoot(host);
});
afterEach(async () => {
  await act(async () => view.unmount());
  host.remove();
  tab.app.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

const ARCHIVED = "2026-10-20T09:00:00Z";
/**
 * The project's cycles and modules: those its lists give (Sprint, Next and Ended; Alpha and Beta), loaded into the
 * store as its wrapper reads them, and archived ones its lists do not give (Old and Older, Gamma and Delta).
 */
async function directory() {
  const cycles = {
    sprint: cycleOf(tab.project, "Sprint", { status: "current" }),
    next: cycleOf(tab.project, "Next", { status: "upcoming" }),
    ended: cycleOf(tab.project, "Ended", { status: "completed" }),
    old: cycleOf(tab.project, "Old", { status: "completed", archived_at: ARCHIVED }),
    older: cycleOf(tab.project, "Older", { status: "completed", archived_at: ARCHIVED }),
  };
  const modules = {
    alpha: moduleOf(tab.project, "Alpha"),
    beta: moduleOf(tab.project, "Beta", { status: "in-progress" }),
    gamma: moduleOf(tab.project, "Gamma", { archived_at: ARCHIVED }),
    delta: moduleOf(tab.project, "Delta", { archived_at: ARCHIVED }),
  };
  const listed = [cycles.sprint, cycles.next, cycles.ended].map((cycle) => rowOf(cycle));
  await loadCycles(tab.nerve, tab.app.planning.cycles, listed, tab.project);
  const modulesListed = [modules.alpha, modules.beta].map((item) => moduleRowOf(item));
  await loadModules(tab.nerve, tab.app.planning.modules, modulesListed, tab.project);
  tab.nerve.calls.length = 0;
  return { ...cycles, ...modules };
}

const Chips = observer(function Chips({ filter }: { filter: IWorkItemFilterInstance }) {
  return filter.allConditions.map((condition) => (
    <FilterItem key={condition.id} condition={condition} filter={filter} showTransition={false} />
  ));
});
type Page = "issues" | "cycle_issues" | "module_issues";
/** The read another page makes of the project's list of a domain: its wrapper's (use-project-fetch.ts). */
function WrapperRead() {
  useSessionSWR(["PROJECT_CYCLES", tab.project.id], (id) => tab.app.planning.cycles.fetchCycles(id));
  useSessionSWR(["PROJECT_MODULES", tab.project.id], (id) => tab.app.planning.modules.fetchModules(id));
  return null;
}
/** The page's filters of project A, under its options, with the conditions saved as expression; their row of chips. */
async function render(
  page: Page,
  expression: TWorkItemFilterExpression = {},
  { update = (value: TWorkItemFilterExpression) => void changes.push(value), beside = null as ReactNode } = {}
) {
  await act(async () =>
    view.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <StoreContext.Provider value={tab.app as never}>
          {beside}
          <WorkItemFiltersHOC
            entityId="A"
            entityType={EIssuesStoreType.PROJECT}
            workspaceSlug="ws"
            projectId="A"
            memberIds={[]}
            stateIds={[]}
            labelIds={[]}
            filtersToShowByLayout={ISSUE_DISPLAY_FILTERS_BY_PAGE[page]!.filters}
            initialWorkItemFilters={{
              richFilters: expression,
              displayFilters: undefined,
              displayProperties: undefined,
              kanbanFilters: undefined,
            }}
            updateFilters={update}
          >
            {({ filter }) => filter && <Chips filter={filter} />}
          </WorkItemFiltersHOC>
        </StoreContext.Provider>
      </SWRConfig>
    )
  );
  return tab.app.workItemFilters.getFilter(EIssuesStoreType.PROJECT, "A")!;
}
const settle = () => act(async () => vi.advanceTimersByTimeAsync(50));
/** What "add filter" lists. */
const offered = (filter: IWorkItemFilterInstance) =>
  filter.configManager.allAvailableConfigs.map((config) => config.id).toSorted();
async function options(filter: IWorkItemFilterInstance, property: TWorkItemFilterProperty) {
  const config = filter.configManager.getConfigByProperty(property)?.getOperatorConfig(COLLECTION_OPERATOR.IN);
  if (config?.type !== FILTER_FIELD_TYPE.MULTI_SELECT) throw new Error(`Missing multi-select for ${property}`);
  return typeof config.getOptions === "function" ? await config.getOptions() : config.getOptions;
}
/** The project's reads of a domain's list so far, by their query: {} for its list, archived=true for its archive. */
const reads = (path: string) =>
  tab.nerve.calls.filter((call) => call.method === "GET" && call.path === path).map((call) => call.query);
const remove = () =>
  act(async () => host.querySelector<HTMLButtonElement>('button[aria-label="Remove filter"]')!.click());

it("offers the project's cycles that are not archived, ended ones too, and its modules, each after None", async () => {
  const { sprint, next, ended, alpha, beta } = await directory();
  const filter = await render("issues");
  expect(offered(filter)).toEqual(expect.arrayContaining(["cycle_id", "module_id"]));
  expect(filter.configManager.getConfigByProperty("cycle_id")?.label).toBe("common.filter_options.fields.cycle_id");
  expect(filter.configManager.getConfigByProperty("module_id")?.label).toBe("common.filter_options.fields.module_id");
  const cycles = await options(filter, "cycle_id");
  expect(cycles.map(({ value, label }) => [value, label])).toEqual([
    ["none", "cycle.no_cycle"],
    [sprint.id, "Sprint"],
    [next.id, "Next"],
    [ended.id, "Ended"],
  ]);
  const statuses = cycles.slice(1).map((option) => (option.icon as ReactElement<{ cycleGroup: string }>).props);
  expect(statuses.map((props) => props.cycleGroup)).toEqual(["current", "upcoming", "completed"]);
  const modules = await options(filter, "module_id");
  expect(modules.map(({ value, label }) => [value, label])).toEqual([
    ["none", "module.no_module"],
    [alpha.id, "Alpha"],
    [beta.id, "Beta"],
  ]);
});

it("saves No cycle as the page's filter none, which the list's query sends", async () => {
  const store = tab.app.issue.projectIssuesFilter;
  await answered(
    tab.nerve,
    () => store.fetchFilters("ws", "A"),
    ["GET", "/api/v0/me/projects/A/preferences"],
    preferencesOf(),
    "the page's preferences"
  );
  await directory();
  const filter = await render("issues", {}, { update: (value) => void store.updateFilterExpression("ws", "A", value) });
  await act(async () =>
    filter.addCondition(LOGICAL_OPERATOR.AND, {
      property: "cycle_id",
      operator: COLLECTION_OPERATOR.IN,
      value: undefined,
    })
  );
  // An empty condition opens its options; No cycle is picked as a person picks it.
  await act(async () => until(() => !!document.querySelector('[role="option"]'), "the cycle's options"));
  const none = [...document.querySelectorAll<HTMLElement>('[role="option"]')].find((option) =>
    option.textContent?.includes("cycle.no_cycle")
  );
  const patches = () => tab.nerve.calls.filter((call) => call.method === "PATCH");
  await act(async () => {
    none!.dispatchEvent(new MouseEvent("mousedown", { button: 0, bubbles: true, cancelable: true }));
  });
  await act(async () => until(() => patches().length === 1, "the preferences' change"));
  expect(patches()[0]!.path).toBe("/api/v0/me/projects/A/preferences");
  expect(patches()[0]!.body).toEqual({ filters: { cycle_id: ["none"] } });
  await act(async () => {
    patches()[0]!.answer(json(200, preferencesOf({ filters: { cycle_id: ["none"] } })));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(toProjectFilters(store.getIssueFilters("A")!.richFilters)).toEqual({ cycle_id: ["none"] });
  const list = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 20 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  expect(list.call.query).toMatchObject({ cycle_id: "none" });
});

it.each([
  ["cycle_id", "old", "older", cycleRequests.list],
  ["module_id", "gamma", "delta", moduleRequests.list],
] as const)(
  "shows a held archived %s by its name and caption, once its project's archive is read, and lets it go",
  async (property, held, another, path) => {
    const values = await directory();
    const name = values[held].name;
    await render("issues", { [`${property}__in`]: values[held].id });
    await act(async () => until(() => reads(path(tab.project)).some((query) => query.archived), "the archive's read"));
    await settle();
    expect(host.textContent).toContain(`common.filter_options.fields.${property}`);
    expect(host.textContent).toContain("common.filter_options.unavailable");
    expect(reads(path(tab.project)).filter((query) => query.archived)).toEqual([{ archived: "true" }]);
    const archive = tab.nerve.calls.find((call) => call.path === path(tab.project) && call.query.archived)!;
    const archived =
      property === "cycle_id"
        ? [rowOf(values.old), rowOf(values.older)]
        : [moduleRowOf(values.gamma), moduleRowOf(values.delta)];
    await act(async () => {
      archive.answer(json(200, { data: archived }));
      await vi.advanceTimersByTimeAsync(20);
    });
    expect(host.textContent).toContain(`${name}, common.archived`);
    expect(host.textContent).not.toContain("common.filter_options.unavailable");
    expect(reads(path(tab.project)).filter((query) => query.archived)).toHaveLength(1);
    // The held one is an option, picked; another archived one never is.
    const filter = tab.app.workItemFilters.getFilter(EIssuesStoreType.PROJECT, "A")!;
    const labels = (await options(filter, property)).map((option) => option.label);
    expect(labels).toContain(`${name}, common.archived`);
    expect(labels.join()).not.toContain(values[another].name);
    await remove();
    expect(toProjectFilters(changes.at(-1)!)).toEqual({});
  }
);

it.each([
  ["cycle_issues", "module_id", "cycle_id"],
  ["module_issues", "cycle_id", "module_id"],
] as const)("never offers the %s page its own domain's filter, the other's only", async (page, other, own) => {
  await directory();
  const filter = await render(page);
  await settle();
  expect(offered(filter)).toContain(other);
  expect(offered(filter)).not.toContain(own);
});

it.each([
  ["cycle_id", { cycle_view: false }, "old", "module_id", cycleRequests.list],
  ["module_id", { module_view: false }, "gamma", "cycle_id", moduleRequests.list],
] as const)(
  "offers no %s filter while the project has its domain off, and keeps a saved one, unnamed, to remove",
  async (property, off, held, other, path) => {
    const values = await directory();
    await act(async () => {
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, ...off }]);
    });
    tab.nerve.calls.length = 0;
    const filter = await render("issues", { [`${property}__in`]: values[held].id });
    await settle();
    expect(filter.configManager.getConfigByProperty(property)?.isEnabled, "add filter does not list it").toBe(false);
    expect(offered(filter)).toContain(other);
    expect(host.textContent).toContain(`common.filter_options.fields.${property}`);
    expect(host.textContent).toContain("common.filter_options.unavailable");
    expect(reads(path(tab.project)), "neither of its lists").toEqual([]);
    await remove();
    expect(toProjectFilters(changes.at(-1)!)).toEqual({});
    expect(offered(filter), "nor once the saved one is gone").not.toContain(property);
    expect(offered(filter)).toContain(other);
  }
);

it("reads neither of the project's lists as a page mounts, nor offers either filter until the store holds them", async () => {
  const filter = await render("issues");
  await settle();
  expect([cycleRequests.list(tab.project), moduleRequests.list(tab.project)].map(reads)).toEqual([[], []]);
  expect(offered(filter)).not.toContain("cycle_id");
  expect(offered(filter)).not.toContain("module_id");
});

it("offers the two filters once the project's wrapper has read its lists, which the filters do not read again", async () => {
  const filter = await render("issues", {}, { beside: <WrapperRead /> });
  await settle();
  const lists = [cycleRequests.list(tab.project), moduleRequests.list(tab.project)];
  expect(lists.map(reads), "the wrapper's reads alone").toEqual([[{}], [{}]]);
  expect(offered(filter), "nothing to offer before they answer").not.toContain("cycle_id");
  expect(offered(filter)).not.toContain("module_id");
  await act(async () => {
    const answer = (path: string, rows: unknown[]) =>
      tab.nerve.calls.find((call) => call.path === path)!.answer(json(200, { data: rows }));
    answer(lists[0]!, [rowOf(cycleOf(tab.project, "Sprint"))]);
    answer(lists[1]!, [moduleRowOf(moduleOf(tab.project, "Alpha"))]);
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(offered(filter)).toEqual(expect.arrayContaining(["cycle_id", "module_id"]));
  await settle();
  expect(lists.map(reads)).toEqual([[{}], [{}]]);
});
