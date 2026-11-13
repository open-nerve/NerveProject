// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter, useLocation, type Location } from "react-router";
import { SWRConfig } from "swr";
import { afterAll, afterEach, beforeAll, beforeEach, expect, it, vi } from "vitest";
import type { ModuleListItem as ModuleRow } from "@nerve/api-client";
import { renderFormattedPayloadDate } from "@nerve/utils";
import type { DateRangeDropdown } from "@/components/dropdowns/date-range";
import { userOf } from "@/store/member/fake-users";
import { moduleRowOf } from "@/store/planning/fake-modules";
import { ModuleFilterStore } from "@/store/planning/module_filter.store";
import { moduleOf, projectOf } from "@/store/project/fake-projects";
import { listedModules } from "./fake-module-lists";
import { ModuleListItem } from "./module-list-item";
import { ModulesListView } from "./modules-list-view";
import { useModuleDisplay } from "./use-module-display";

// A project's modules list (P6a spec 2.4, 2.6, R8, R13, R14): its modules not archived, filtered and searched by the
// project's own filters and search, as rows or as cards. A row: its name, a link to its page; its progress, the
// completed of its work items not cancelled (progressPercent), with a tick once all are done and a "!" for a module
// completed short of that; its preview, its days (calendar days), its status, its lead, its quick actions; no star to
// favorite it. A card: the same progress, as a count and a bar. Nothing matching, it says which to clear. The modules
// are a stand-in store's, the filters a session's real store; the days' picker, the avatars, the quick actions and the
// preview's sidebar stand-ins.

const held = vi.hoisted(() => ({
  modules: undefined as unknown as ReturnType<typeof import("./fake-module-lists").listedModules>,
  filters: undefined as unknown as import("@/store/planning/module_filter.store").ModuleFilterStore,
  /** The days' pickers shown, each by its button, with the props it was given. */
  pickers: new Map<Element, unknown>(),
  setLayout: undefined as ((layout: "list" | "board") => void) | undefined,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "x" }) }));
vi.mock("@/hooks/store/use-modules", () => ({ useModules: () => held.modules }));
vi.mock("@/hooks/store/use-module-filters", () => ({ useModuleFilters: () => held.filters }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => ({ canWriteModules: () => true }) }));
vi.mock("@/hooks/use-planning-reads", () => ({ usePlanningReads: () => ({ readable: true, listed: true }) }));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => ({ id: "id-acme" }) }));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({ getUserDetails: (id: string) => [ann].find((user) => user.id === id) }),
}));
vi.mock("@/components/assets/profile-image", () => ({
  ProfileAvatar: (props: { alt?: string; source: { memberID: string } }) => (
    <i data-avatar={props.source.memberID} title={props.alt} />
  ),
}));
vi.mock("@/components/dropdowns/date-range", () => ({
  DateRangeDropdown: (props: ComponentProps<typeof DateRangeDropdown>) => (
    <button
      type="button"
      aria-label="days"
      ref={(button) => {
        if (button) held.pickers.set(button, props);
      }}
    />
  ),
}));
vi.mock("./quick-actions", () => ({
  ModuleQuickActions: (props: { moduleId: string }) => (
    <button type="button" aria-label={`options of ${props.moduleId}`} />
  ),
}));
vi.mock("./progress-sidebar", () => ({
  ModuleDetailsSidebar: (props: { moduleId: string }) => <aside aria-label={`sidebar of ${props.moduleId}`} />,
}));

const web = projectOf("WEB", "id-acme");
const ops = projectOf("OPS", "id-acme");
const ann = userOf("Ann");
/** Counts of work items: of total, completed and cancelled, the rest in the backlog. */
const counts = (total: number, completed: number, cancelled = 0) => ({
  total_issues: total,
  completed_issues: completed,
  cancelled_issues: cancelled,
  backlog_issues: total - completed - cancelled,
});
// web's modules, in nerve's order
const rows: ModuleRow[] = [
  moduleRowOf(moduleOf(web, "Docs", { status: "planned" })),
  moduleRowOf(moduleOf(web, "Search", { status: "completed", lead_id: "u-gone" }), counts(5, 3, 1)),
  moduleRowOf(
    moduleOf(web, "Auth", {
      status: "in-progress",
      lead_id: ann.id,
      member_ids: [ann.id],
      start_date: "2026-11-23",
      target_date: "2026-11-29",
    }),
    counts(10, 4, 2)
  ),
  moduleRowOf(moduleOf(web, "Cache", { status: "completed" }), counts(3, 2, 1)),
];

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
let location: Location | undefined;
function Where() {
  location = useLocation();
  return null;
}
function Layout() {
  held.setLayout = useModuleDisplay(web.id).setLayout;
  return null;
}
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeAll(() => vi.stubEnv("TZ", "America/Los_Angeles"));
afterAll(() => vi.unstubAllEnvs());
beforeEach(async () => {
  vi.useFakeTimers({ now: new Date(2026, 10, 13, 21, 30) });
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  localStorage.clear();
  held.pickers.clear();
  held.modules = listedModules();
  held.modules.show("modules", web.id, rows);
  held.modules.show("modules", ops.id, [moduleRowOf(moduleOf(ops, "Ops"))]);
  held.filters = new ModuleFilterStore();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  await render(<ModulesListView workspaceSlug="acme" projectId={web.id} />);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

async function render(content: React.ReactNode, path = `/acme/projects/${web.id}/modules`) {
  await act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <MemoryRouter initialEntries={[path]}>
          {content}
          <Layout />
          <Where />
        </MemoryRouter>
      </SWRConfig>
    )
  );
}
const settle = () => act(async () => vi.advanceTimersByTimeAsync(100));
const text = () => host.textContent ?? "";
/** The links of the modules the list shows, in its order. */
const links = () => [...host.querySelectorAll<HTMLAnchorElement>("a[href]")];
/** The name a link shows: a row's title, after the ring of its progress, or a card's. */
const nameOf = (link: HTMLAnchorElement) => (link.querySelector(".truncate.text-13") ?? link).textContent;
const named = () => links().map(nameOf);
/** The row or the card of the module of name: the element that holds its link and its controls. */
const itemNamed = (name: string) =>
  links()
    .find((link) => nameOf(link) === name)
    ?.closest<HTMLElement>(".group, [data-prevent-progress]");
const ring = (name: string) => itemNamed(name)?.querySelector('[role="progressbar"]');
/** What a row's ring says over it: the percent, a tick once done, or "!". */
const mark = (name: string) => ring(name)?.parentElement?.nextElementSibling;
const buttonsOf = (name: string) =>
  [...(itemNamed(name)?.querySelectorAll("button") ?? [])].map(
    (one) => one.getAttribute("aria-label") ?? one.textContent
  );
const daysOf = (name: string) => {
  const picker = itemNamed(name)?.querySelector('button[aria-label="days"]');
  const props = (picker && held.pickers.get(picker)) as ComponentProps<typeof DateRangeDropdown> | undefined;
  return `${renderFormattedPayloadDate(props?.value.from) ?? "-"} → ${renderFormattedPayloadDate(props?.value.to) ?? "-"}`;
};

it("lists the project's modules, each linked to its page", () => {
  expect(named()).toEqual(["Auth", "Cache", "Docs", "Search"]);
  expect(itemNamed("Auth")?.querySelector("a")?.getAttribute("href")).toBe(
    `/acme/projects/${web.id}/modules/m-WEB-Auth`
  );
});

it("shows each row's progress, the completed of those not cancelled, a tick when done, a '!' when completed short", () => {
  // 4 of the 8 not cancelled; the old reading, (completed + cancelled) of the total, says 60
  expect(ring("Auth")?.getAttribute("aria-valuenow")).toBe("50");
  expect(mark("Auth")?.textContent).toBe("50%");
  // completed short of all: 3 of 4
  expect(ring("Search")?.getAttribute("aria-valuenow")).toBe("75");
  expect(mark("Search")?.textContent).toBe("!");
  // completed, all done: a tick
  expect(ring("Cache")?.getAttribute("aria-valuenow")).toBe("100");
  expect(mark("Cache")?.textContent).toBe("");
  expect(mark("Cache")?.querySelector("svg")).not.toBeNull();
  expect(ring("Docs")?.getAttribute("aria-valuenow")).toBe("0");
  expect(mark("Docs")?.textContent).toBe("0%");
});

it("shows a module's lead by the avatar, one who has left by the former member's name, else that it has none", () => {
  expect(itemNamed("Auth")?.querySelector("[data-avatar]")?.getAttribute("title")).toBe("Ann");
  expect(itemNamed("Search")?.querySelector("[data-avatar]")?.getAttribute("title")).toBe("distribution.former_member");
  expect(itemNamed("Docs")?.querySelector("[data-avatar]")).toBeNull();
  expect(itemNamed("Docs")?.querySelector('[aria-label="project_module.no_lead"]')).not.toBeNull();
});

it("gives a row its preview, its days, its status and its quick actions, with no star to favorite it", () => {
  expect(buttonsOf("Auth")).toEqual([
    "project_module.more_details",
    "days",
    "project_modules.status.in_progress",
    "options of m-WEB-Auth",
  ]);
});

it("shows a module's days as days of its project's calendar, not shifted by the browser's zone", () => {
  // the browser is in Los Angeles, where 2026-11-23 read as a time in UTC is the 22nd
  expect(new Date(2026, 10, 23).getTimezoneOffset()).toBe(480);
  expect(daysOf("Auth")).toBe("2026-11-23 → 2026-11-29");
  expect(daysOf("Docs")).toBe("- → -");
});

it("previews a module by its 'more details', and closes the preview by it again", async () => {
  const more = () =>
    [...(itemNamed("Cache")?.querySelectorAll("button") ?? [])].find(
      (one) => one.getAttribute("aria-label") === "project_module.more_details"
    );
  await act(async () => more()?.click());
  await settle();
  expect(location?.search).toBe("?peekModule=m-WEB-Cache");
  expect(host.querySelector('aside[aria-label="sidebar of m-WEB-Cache"]')).not.toBeNull();
  expect(held.modules.asked()).toEqual(["module of m-WEB-Cache", "progress of m-WEB-Cache", "links of m-WEB-Cache"]);
  await act(async () => more()?.click());
  expect(location?.search).toBe("");
  expect(host.querySelector("aside")).toBeNull();
});

it("opens the preview of an archived module's row, not its page", async () => {
  const old = moduleRowOf(moduleOf(web, "Old", { status: "completed", archived_at: "2026-11-02T09:00:00Z" }));
  held.modules.show("archive", web.id, [old]);
  await render(<ModuleListItem moduleId={old.id} workspaceSlug="acme" projectId={web.id} />);
  await act(async () => links()[0]?.click());
  expect(location?.pathname).toBe(`/acme/projects/${web.id}/modules`);
  expect(location?.search).toBe("?peekModule=m-WEB-Old");
});

it("shows each card's progress as a count and a bar, the completed of those not cancelled", async () => {
  await act(async () => held.setLayout?.("board"));
  expect(named()).toEqual(["Auth", "Cache", "Docs", "Search"]);
  expect(itemNamed("Auth")?.textContent).toContain("project_module.work_items_done 4 8");
  expect(ring("Auth")?.getAttribute("aria-valuenow")).toBe("50");
  expect(itemNamed("Search")?.textContent).toContain("project_module.work_items_done 3 4");
  expect(ring("Search")?.getAttribute("aria-valuenow")).toBe("75");
  expect(itemNamed("Docs")?.textContent).toContain("project_module.work_items_done 0 0");
  expect(buttonsOf("Auth")).toEqual([
    "project_modules.status.in_progress",
    "project_module.more_details",
    "days",
    "options of m-WEB-Auth",
  ]);
  expect(itemNamed("Auth")?.querySelector("[data-avatar]")?.getAttribute("title")).toBe("Ann");
  expect(daysOf("Auth")).toBe("2026-11-23 → 2026-11-29");
});

it("filters the project's modules by its own filters: another project's and its archive's are their own", async () => {
  await act(async () => held.filters.updateFilters(ops.id, "default", { status: ["completed"] }));
  await act(async () => held.filters.updateFilters(web.id, "archived", { status: ["completed"] }));
  expect(named()).toEqual(["Auth", "Cache", "Docs", "Search"]);
  await act(async () => held.filters.updateFilters(web.id, "default", { status: ["completed", "planned"] }));
  expect(named()).toEqual(["Cache", "Docs", "Search"]);
  await act(async () => held.filters.updateFilters(web.id, "default", { members: [ann.id] }));
  expect(named()).toEqual([]);
  await act(async () => held.filters.updateFilters(web.id, "default", { status: [] }));
  expect(named()).toEqual(["Auth"]);
});

it("searches the project's modules by name, whatever its case", async () => {
  await act(async () => held.filters.updateSearchQuery(ops.id, "default", "nothing"));
  await act(async () => held.filters.updateSearchQuery(web.id, "default", "CA"));
  expect(named()).toEqual(["Cache"]);
});

it.each([
  { what: "filters", by: () => held.filters.updateFilters(web.id, "default", { target_date: ["2025-01-01;before"] }) },
  { what: "search", by: () => held.filters.updateSearchQuery(web.id, "default", "zzz") },
])("says, nothing matching its $what, that clearing it shows them all", async ({ what, by }) => {
  await act(async () => by());
  expect(named()).toEqual([]);
  expect(text()).toContain("project_module.no_matching_modules");
  expect(text()).toContain(
    what === "search"
      ? "project_module.remove_search_criteria_to_see_all_modules"
      : "project_module.remove_filters_to_see_all_modules"
  );
});
