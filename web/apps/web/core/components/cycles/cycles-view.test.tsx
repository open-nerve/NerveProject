// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter, useLocation, type Location } from "react-router";
import { SWRConfig } from "swr";
import { afterAll, afterEach, beforeAll, beforeEach, expect, it, vi } from "vitest";
import type { CycleListItem } from "@nerve/api-client";
import { userOf } from "@/store/member/fake-users";
import { rowOf } from "@/store/planning/fake-cycles";
import { CycleFilterStore } from "@/store/planning/cycle_filter.store";
import { cycleOf, projectOf } from "@/store/project/fake-projects";
import { listedCycles } from "./fake-cycle-lists";
import { CyclesView } from "./cycles-view";

// A project's cycles list (P5a spec 2.3, 2.5, R5–R7): the cycle in progress first (its card is the active cycle's
// own, a stand-in here), then the upcoming cycles and the drafts in the list's order, then the ended ones in nerve's
// order; each row its progress, its work items while not yet started, its days, its creator and its members; filtered
// and searched by the project's own filters and search. The cycles are a stand-in store's, the filters the real store
// of a session's; the avatars, the cycle's quick actions and the preview's sidebar stand-ins.

const held = vi.hoisted(() => ({
  cycles: undefined as unknown as ReturnType<typeof listedCycles>,
  filters: undefined as unknown as CycleFilterStore,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "x" }) }));
vi.mock("@/hooks/store/use-cycles", () => ({ useCycles: () => held.cycles }));
vi.mock("@/hooks/store/use-cycle-filters", () => ({ useCycleFilters: () => held.filters }));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => ({ id: "id-acme" }) }));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    project: {
      getProjectMemberDetails: (id: string, projectId: string) => {
        const member = projectId === web.id ? [ann, bob].find((user) => user.id === id) : undefined;
        return member ? { member } : null;
      },
    },
  }),
}));
vi.mock("@/components/assets/profile-image", () => ({
  ProfileAvatar: (props: { alt?: string; source: { memberID: string } }) => (
    <i data-avatar={props.source.memberID} title={props.alt} />
  ),
}));
vi.mock("@/components/cycles/active-cycle/root", () => ({
  ActiveCycleRoot: (props: { projectId: string }) => <section aria-label={`in progress of ${props.projectId}`} />,
}));
vi.mock("@/components/cycles/quick-actions", () => ({
  CycleQuickActions: (props: { cycleId: string }) => (
    <button type="button" aria-label={`options of ${props.cycleId}`} />
  ),
}));
vi.mock("@/components/cycles/progress-sidebar", () => ({
  CycleDetailsSidebar: (props: { cycleId: string }) => <aside aria-label={`sidebar of ${props.cycleId}`} />,
}));

const web = projectOf("WEB", "id-acme");
const ops = projectOf("OPS", "id-acme");
const ann = userOf("Ann");
const bob = userOf("bob");
/** The days of a cycle of a week, from start to end. */
const week = (start_date: string, end_date: string) => ({ start_date, end_date });

const frozen = (completed: number, total: number) => ({
  total_issues: total,
  backlog_issues: 0,
  unstarted_issues: total - completed,
  started_issues: 0,
  completed_issues: completed,
  cancelled_issues: 0,
  frozen_at: "2026-11-02T00:00:00Z",
});
// web's cycles, in nerve's order (by sort order): neither by status nor by name
const rows: CycleListItem[] = [
  rowOf(cycleOf(web, "S0", { status: "completed", ...week("2026-10-12", "2026-10-18") }), {
    total_issues: 4,
    completed_issues: 1,
  }),
  rowOf(cycleOf(web, "beta"), { total_issues: 2, backlog_issues: 2 }),
  rowOf(cycleOf(web, "S4", { status: "upcoming", ...week("2026-12-07", "2026-12-13") })),
  rowOf(cycleOf(web, "S2", { status: "current", ...week("2026-11-09", "2026-11-15") })),
  rowOf(cycleOf(web, "Alpha", { created_by_id: ann.id }), { assignee_ids: [ann.id, bob.id] }),
  rowOf(cycleOf(web, "S3", { status: "upcoming", ...week("2026-11-23", "2026-11-29") }), {
    total_issues: 5,
    completed_issues: 2,
    started_issues: 2,
    cancelled_issues: 1,
    // nerve keeps what it froze of a cycle whose days changed, so that it has not ended (R7)
    progress_snapshot: frozen(5, 5),
  }),
  rowOf(cycleOf(web, "S1", { status: "completed", ...week("2026-10-26", "2026-11-01") }), {
    total_issues: 4,
    completed_issues: 1,
    unstarted_issues: 3,
    progress_snapshot: frozen(4, 4),
  }),
];

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
let location: Location | undefined;
function Where() {
  location = useLocation();
  return null;
}
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeAll(() => vi.stubEnv("TZ", "America/Los_Angeles"));
afterAll(() => vi.unstubAllEnvs());
beforeEach(async () => {
  vi.useFakeTimers({ now: new Date(2026, 10, 11, 21, 30) });
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  held.cycles = listedCycles();
  held.filters = new CycleFilterStore();
  held.cycles.show("cycles", web.id, rows);
  held.cycles.show("cycles", ops.id, [
    rowOf(cycleOf(ops, "O1", { status: "upcoming", ...week("2026-11-16", "2026-11-22") })),
  ]);
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  await act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <MemoryRouter initialEntries={[`/acme/projects/${web.id}/cycles`]}>
          <CyclesView workspaceSlug="acme" projectId={web.id} />
          <Where />
        </MemoryRouter>
      </SWRConfig>
    )
  );
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

const settle = () => act(async () => vi.advanceTimersByTimeAsync(100));
const section = (label: string) => host.querySelector<HTMLElement>(`section[aria-label="${label}"]`);
const UPCOMING = "project_cycles.upcoming_cycle.label";
const COMPLETED = "project_cycles.completed_cycle.label";
/** A row's link and the name it shows (its title, after the ring of its progress). */
const links = (within: Element | null | undefined) =>
  [...(within?.querySelectorAll<HTMLAnchorElement>("a[href]") ?? [])].map((link) => ({
    link,
    name: link.querySelector(".truncate.text-13")?.textContent,
  }));
/** The names of the cycles a section of the list shows, in its order. */
const named = (label: string) => links(section(label)).map(({ name }) => name);
/** The link of the cycle of name. */
const linkOf = (name: string) => links(host).find((one) => one.name === name)?.link;
/** The row of the cycle of name: the element that holds its link and its actions. */
const rowNamed = (name: string) => linkOf(name)?.closest(".group");
const header = (label: string) => section(label)?.querySelector("button");
const text = () => host.textContent ?? "";

it("lists the cycle in progress first, then the upcoming cycles and the drafts, then the ended ones, closed", () => {
  expect(section(`in progress of ${web.id}`)).not.toBeNull();
  // in progress first, upcoming ones by the day they start, then the drafts by their names' lower case (R5)
  expect(header(UPCOMING)?.getAttribute("aria-expanded")).toBe("true");
  expect(header(UPCOMING)?.textContent).toContain("4");
  expect(named(UPCOMING)).toEqual(["S3", "S4", "Alpha", "beta"]);
  expect(header(COMPLETED)?.getAttribute("aria-expanded")).toBe("false");
  expect(header(COMPLETED)?.textContent).toContain("2");
  expect(named(COMPLETED)).toEqual([]);
});

it("shows the ended cycles in nerve's order once opened", async () => {
  await act(async () => header(COMPLETED)?.click());
  expect(named(COMPLETED)).toEqual(["S0", "S1"]);
});

it("links each cycle to its page", () => {
  expect(linkOf("S3")?.getAttribute("href")).toBe(`/acme/projects/${web.id}/cycles/c-WEB-S3`);
});

it("shows each cycle's progress, the completed of those not cancelled, frozen once ended, with a tick when done", async () => {
  await act(async () => header(COMPLETED)?.click());
  const ring = (name: string) => rowNamed(name)?.querySelector('[role="progressbar"]');
  // what the ring says over it: the percent, or a tick once done
  const mark = (name: string) => ring(name)?.parentElement?.nextElementSibling;
  // not ended: its live counts, though nerve keeps what it froze (R7)
  expect(ring("S3")?.getAttribute("aria-valuenow")).toBe("50");
  expect(mark("S3")?.textContent).toBe("50%");
  // ended and frozen: the counts it was frozen with, all done
  expect(ring("S1")?.getAttribute("aria-valuenow")).toBe("100");
  expect(mark("S1")?.textContent).toBe("");
  expect(mark("S1")?.querySelector("svg")).not.toBeNull();
  // ended, not frozen: live
  expect(ring("S0")?.getAttribute("aria-valuenow")).toBe("25");
  expect(mark("S0")?.textContent).toBe("25%");
});

it("says how many work items a draft and an upcoming cycle have, and not an ended one", async () => {
  await act(async () => header(COMPLETED)?.click());
  const count = (name: string) => rowNamed(name)?.querySelector('[title="work_items"]')?.textContent;
  expect(count("beta")).toBe("2");
  expect(count("S3")).toBe("5");
  expect(count("S0")).toBeUndefined();
});

it("shows a cycle's days as days of its project's calendar, not shifted by the browser's zone", () => {
  // the browser is in Los Angeles, where 2026-11-23 read as a time in UTC is the 22nd
  expect(new Date(2026, 10, 23).getTimezoneOffset()).toBe(480);
  expect(rowNamed("S3")?.textContent).toContain("Nov 23 - 29, 2026");
  expect(rowNamed("Alpha")?.textContent).not.toContain("2026");
});

it("shows a cycle's creator and its members by their avatars, with no star to favorite it", () => {
  const alpha = rowNamed("Alpha");
  expect(alpha?.querySelector('[title="common.created_by"] [data-avatar]')?.getAttribute("title")).toBe("Ann");
  const members = [...(alpha?.querySelectorAll("[data-avatar]") ?? [])].slice(1);
  expect(members.map((avatar) => avatar.getAttribute("data-avatar"))).toEqual([ann.id, bob.id]);
  expect(rowNamed("beta")?.querySelector('[title="common.created_by"]')).toBeNull();
  const buttons = [...(alpha?.querySelectorAll("button") ?? [])].map(
    (one) => one.getAttribute("aria-label") ?? one.textContent
  );
  expect(buttons).toEqual(["project_cycles.more_details", "options of c-WEB-Alpha"]);
});

it("previews a cycle by its 'more details', and closes the preview by it again", async () => {
  const more = () =>
    [...(rowNamed("S4")?.querySelectorAll("button") ?? [])].find(
      (one) => one.textContent === "project_cycles.more_details"
    );
  await act(async () => more()?.click());
  await settle();
  expect(location?.search).toBe("?peekCycle=c-WEB-S4");
  expect(host.querySelector('aside[aria-label="sidebar of c-WEB-S4"]')).not.toBeNull();
  expect(held.cycles.asked()).toEqual(["cycle of c-WEB-S4", "progress of c-WEB-S4"]);
  await act(async () => more()?.click());
  expect(location?.search).toBe("");
  expect(host.querySelector("aside")).toBeNull();
});

it("filters the project's cycles by nerve's status, by its own filters: another project's are their own", async () => {
  await act(async () => held.filters.updateFilters(ops.id, "default", { status: ["upcoming"] }));
  await act(async () => held.filters.updateFilters(web.id, "archived", { status: ["upcoming"] }));
  expect(named(UPCOMING)).toEqual(["S3", "S4", "Alpha", "beta"]);
  await act(async () => held.filters.updateFilters(web.id, "default", { status: ["draft", "completed"] }));
  expect(named(UPCOMING)).toEqual(["Alpha", "beta"]);
  expect(header(COMPLETED)?.textContent).toContain("2");
});

it("searches the project's cycles by name, whatever its case", async () => {
  await act(async () => held.filters.updateSearchQuery(ops.id, "default", "nothing"));
  await act(async () => held.filters.updateSearchQuery(web.id, "default", "ALP"));
  expect(named(UPCOMING)).toEqual(["Alpha"]);
  expect(header(COMPLETED)?.textContent).toContain("0");
});

it.each([
  { what: "filters", by: () => held.filters.updateFilters(web.id, "default", { end_date: ["2025-01-01;before"] }) },
  { what: "search", by: () => held.filters.updateSearchQuery(web.id, "default", "zzz") },
])("says, nothing matching its $what, that clearing it shows them all", async ({ what, by }) => {
  await act(async () => by());
  expect(section(`in progress of ${web.id}`)).toBeNull();
  expect(text()).toContain("project_cycles.no_matching_cycles");
  expect(text()).toContain(
    what === "search"
      ? "project_cycles.remove_search_criteria_to_see_all_cycles"
      : "project_cycles.remove_filters_to_see_all_cycles"
  );
});
