// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The archive's words of a project's planning (M6 P8 spec 2.4), through its real header, filters and list against the
// session's stores and a fake nerve. The display panel and "add filter" offer the cycle and the module as on the
// project's list (the shared grouping panel and planning filters), while the route's project has the domain on and the
// store holds its list (read by the project's wrapper, as here); the list is the project's, archived=true, grouped and
// filtered by them; its groups are named through usePlanningNames, an archived value by name and caption once the
// project's archive is read; a read-only list adds nothing to a group. While a domain is off neither word is offered,
// and a saved grouping and filter by it stay, to change and remove: the group unnamed, the value unavailable (P7b R8).
import { act, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { IWorkItemFilterInstance } from "@nerve/shared-state";
import { EIssuesStoreType } from "@nerve/types";
import type { RootStore } from "@/store/root.store";
import { seedAccount } from "@/store/user/fake-account";
import { accountOf } from "@/store/member/fake-users";
import { issuePageOf, listIssue } from "@/store/issue/fake-issues";
import { cycleRequests, loadCycles, rowOf } from "@/store/planning/fake-cycles";
import { loadModules, moduleRequests, moduleRowOf } from "@/store/planning/fake-modules";
import { cycleOf, loadProjects, moduleOf } from "@/store/project/fake-projects";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { ArchivedIssuesHeader } from "../../archived-issues-header";
import { ArchivedIssueLayoutRoot } from "./archived-issue-layout-root";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "archive-words" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "archive-words" }) }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws", projectId: "A" }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/use-profile-image", () => import("@/lib/fake-profile-image"));
// Only the browser viewport transport is replaced; the header, filters, list and rows are real.
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children }: { children: ReactNode }) => children,
}));

let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
beforeEach(async () => {
  vi.stubGlobal(
    "IntersectionObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  localStorage.clear();
  tab = await issueSelectionTab();
  bound.app = tab.app;
  await seedAccount(tab.nerve, tab.app.user, accountOf("alice"));
  await tab.app.state.fetchProjectStates("A");
  tab.nerve.calls.length = 0;
});
afterEach(async () => {
  await tab.close();
  localStorage.clear();
});

const ARCHIVED_AT = "2026-10-20T09:00:00Z";
/** A domain's values: one its list gives, one archived (a filter keeps ids alone). */
const SPRINT = { id: "019a0000-0000-7000-8000-0000000000c1", name: "Sprint" };
const OLD = { id: "019a0000-0000-7000-8000-0000000000c2", name: "Old" };
const ALPHA = { id: "019a0000-0000-7000-8000-0000000000d1", name: "Alpha" };
const GAMMA = { id: "019a0000-0000-7000-8000-0000000000d2", name: "Gamma" };
/** Each domain's words, its values and the rows of its project's archive. */
const words = [
  {
    group: "cycle",
    title: "common.cycle",
    property: "cycle_id",
    other: "module_id",
    off: { cycle_view: false },
    path: cycleRequests.list,
    listed: SPRINT,
    archived: OLD,
    of: (id: string) => ({ cycle_id: id }),
    archive: () => [rowOf(cycleOf(tab.project, OLD.name, { id: OLD.id, archived_at: ARCHIVED_AT }))],
  },
  {
    group: "module",
    title: "common.modules",
    property: "module_id",
    other: "cycle_id",
    off: { module_view: false },
    path: moduleRequests.list,
    listed: ALPHA,
    archived: GAMMA,
    of: (id: string) => ({ module_ids: [id] }),
    archive: () => [moduleRowOf(moduleOf(tab.project, GAMMA.name, { id: GAMMA.id, archived_at: ARCHIVED_AT }))],
  },
] as const;

/** The project's lists of what is not archived, in the store as its wrapper reads them. */
async function directory() {
  await loadCycles(tab.nerve, tab.app.planning.cycles, [rowOf(cycleOf(tab.project, SPRINT.name, SPRINT))], tab.project);
  await loadModules(
    tab.nerve,
    tab.app.planning.modules,
    [moduleRowOf(moduleOf(tab.project, ALPHA.name, ALPHA))],
    tab.project
  );
  tab.nerve.calls.length = 0;
}
/** The archive's preferences the browser holds for alice. */
const saved = (filters: object) =>
  localStorage.setItem(
    "issue_local_filters",
    JSON.stringify([{ key: "ARCHIVED", workspaceSlug: "ws", viewId: "A", userId: "u-alice", filters }])
  );
const mount = () =>
  tab.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter>
        <ArchivedIssuesHeader />
        <ArchivedIssueLayoutRoot />
      </MemoryRouter>
    </SWRConfig>
  );
const lists = () => tab.nerve.to("/api/v0/projects/A/issues");
async function answer(index: number, page: object) {
  await act(async () => until(() => lists().length === index + 1, `archive read ${index}`));
  await act(async () => lists()[index]!.answer(json(200, page)));
}
const grouped = (groups: [string, object][]) => ({
  total: groups.length,
  groups: groups.map(([key, fields], at) => ({
    key,
    total: 1,
    data: [{ ...listIssue(`x${at}`, at + 10, fields), archived_at: ARCHIVED_AT }],
    next_cursor: null,
  })),
});
/** The project's reads of a domain's lists so far, by their query: {} for its list, archived=true for its archive. */
const reads = (path: string) =>
  tab.nerve.calls.filter((call) => call.method === "GET" && call.path === path).map((call) => call.query);
const settle = () => act(async () => vi.advanceTimersByTimeAsync(50));
const filter = (): IWorkItemFilterInstance => tab.app.workItemFilters.getFilter(EIssuesStoreType.ARCHIVED, "A")!;
/** What "add filter" lists. */
const offered = () => filter().configManager.allAvailableConfigs.map((config) => config.id);
/** The display panel's groupings, a checked one marked, and one of them by its title. */
const groupingButtons = () => [
  ...(document
    .querySelector('button[aria-label="common.group_by"]')
    ?.closest(".py-2")
    ?.querySelectorAll<HTMLButtonElement>("button[aria-pressed]") ?? []),
];
const groupings = () =>
  groupingButtons().map((b) => `${b.textContent}${b.getAttribute("aria-pressed") === "true" ? " (checked)" : ""}`);
const grouping = (title: string) => groupingButtons().find((b) => b.textContent === title);
/** The list's group headers: each one's name, and its other buttons (a new work item, an existing one). */
const headers = () => [...document.querySelectorAll('[class~="group/list-header"]')];
const groupNames = () => headers().map((h) => h.querySelector("button[aria-expanded]")?.getAttribute("aria-label"));
const adders = () => [...document.querySelectorAll('[class~="group/list-header"] button:not([aria-expanded])')];
const click = (element: HTMLElement | null | undefined) =>
  act(async () => {
    expect(element).toBeTruthy();
    element!.click();
  });
const display = () =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].find(
    (b) => b.getAttribute("aria-label") === "common.display" || b.textContent?.trim() === "common.display"
  );

it.each(words)(
  "offers the $group in the archive's panel and filters, and groups by it as the archive's list, naming its groups and adding nothing to them",
  async ({ title, property, path, listed, archived, of, archive }) => {
    await directory();
    await mount();
    await answer(0, issuePageOf([]));
    expect(offered()).toEqual(expect.arrayContaining(["cycle_id", "module_id"]));
    await click(display());
    expect(groupings()).toEqual(expect.arrayContaining(["common.cycle", "common.modules"]));
    await click(grouping(title));
    await answer(
      1,
      grouped([
        [archived.id, of(archived.id)],
        [listed.id, of(listed.id)],
      ])
    );
    expect(lists()[1]!.query).toMatchObject({ archived: "true", group_by: property });
    // The archived value's group is unnamed until one read of the project's archive names it, with its caption.
    await act(async () => until(() => reads(path(tab.project)).length > 0, "the archive's read"));
    await settle();
    expect(reads(path(tab.project))).toEqual([{ archived: "true" }]);
    expect(groupNames()).toEqual([listed.name, "—"]);
    await act(async () => {
      tab.nerve.calls.find((call) => call.path === path(tab.project))!.answer(json(200, { data: archive() }));
      await vi.advanceTimersByTimeAsync(20);
    });
    expect(groupNames()).toEqual([listed.name, `${archived.name}, common.archived`]);
    expect(adders()).toEqual([]);
    expect(tab.nerve.calls.every((call) => call.method === "GET")).toBe(true);
  }
);

it.each(words)(
  "offers no $group word while the project has its domain off, and keeps a saved grouping and filter by it to change and remove",
  async ({ group, title, property, other, off, path, archived, of }) => {
    await directory();
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, ...off }]);
    tab.nerve.calls.length = 0;
    saved({ display_filters: { group_by: group }, rich_filters: { [`${property}__in`]: archived.id } });
    await mount();
    await answer(0, grouped([[archived.id, of(archived.id)]]));
    expect(lists()[0]!.query).toMatchObject({ archived: "true", group_by: property, [property]: archived.id });
    await settle();
    expect(groupNames(), "the group reads no name").toEqual(["—"]);
    expect(reads(path(tab.project)), "nor either of the domain's lists").toEqual([]);
    expect(filter().configManager.getConfigByProperty(property)?.isEnabled, "add filter does not list it").toBe(false);
    expect(offered()).toContain(other);
    expect(document.body.textContent).toContain("common.filter_options.unavailable");
    await click(display());
    expect(groupings().filter((name) => name.startsWith(title))).toEqual([`${title} (checked)`]);
    await click(grouping("common.labels"));
    await answer(1, grouped([]));
    expect(lists()[1]!.query).toMatchObject({ archived: "true", group_by: "label_id" });
    expect(grouping("common.labels")?.getAttribute("aria-pressed"), "the panel open, labels picked").toBe("true");
    expect(grouping(title), "once changed, no longer offered").toBeUndefined();
    await click(document.querySelector<HTMLButtonElement>('button[aria-label="Remove filter"]'));
    await answer(2, grouped([]));
    expect(lists()[2]!.query).not.toHaveProperty(property);
    expect(offered()).not.toContain(property);
    expect(tab.nerve.calls.every((call) => call.method === "GET")).toBe(true);
  }
);
