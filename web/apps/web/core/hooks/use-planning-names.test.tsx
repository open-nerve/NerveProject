// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// M6 P7a spec 2.3, R14: what names a work item's cycle or modules, read on demand, against the session's stores and a
// fake nerve. A value the planning store cannot name has its project's list read while the store has none, then the
// project's archive, each once a project whoever asks, by the keys the project's wrapper and its archive read them by;
// nothing is read for a caller without a role in the project, or while the domain is off.
import { act } from "react";
import type { ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { observer } from "mobx-react";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import type { PlanningDomain } from "@/lib/planning-switch";
import { useSessionSWR } from "@/lib/use-session-swr";
import { issueStoreTab } from "@/store/issue/fake-issues";
import { cycleRequests, rowOf } from "@/store/planning/fake-cycles";
import { moduleRequests, moduleRowOf } from "@/store/planning/fake-modules";
import { cycleOf, loadProjects, moduleOf } from "@/store/project/fake-projects";
import { usePlanningNames } from "./use-planning-names";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "names" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "names" }) }));
vi.mock("@/hooks/store/use-cycles", () => ({ useCycles: () => tab.app.planning.cycles }));
vi.mock("@/hooks/store/use-modules", () => ({ useModules: () => tab.app.planning.modules }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => tab.app.user.permission }));

let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let renderer: ReturnType<typeof createRoot>;

beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await issueStoreTab();
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

/** A domain as the tests see it: its values as nerve lists them, its project's lists, and the store's names. */
type Domain = {
  /** A value of the project as its list gives it, archived when it says so. */
  row: (name: string, archived?: boolean) => { id: string };
  /** The path of the project's list, archived (archived=true) or not. */
  path: () => string;
  /** The keys the project's wrapper reads its list by, and its archive's page its archive (use-project-fetch.ts). */
  keys: readonly [list: string, archive: string];
  /** The store's fetches of the project's list and of its archive. */
  fetches: readonly [list: (projectId: string) => Promise<unknown>, archive: (projectId: string) => Promise<unknown>];
  /** The value's name as the store has it. */
  name: (id: string) => string | undefined;
  /** The project with the domain turned off. */
  off: Partial<Project>;
};
const ARCHIVED = "2026-10-20T00:00:00Z";
const DOMAINS: Record<PlanningDomain, Domain> = {
  cycles: {
    row: (name, archived = false) => rowOf(cycleOf(tab.project, name, archived ? { archived_at: ARCHIVED } : {})),
    path: () => cycleRequests.list(tab.project),
    keys: ["PROJECT_CYCLES", "PROJECT_ARCHIVED_CYCLES"],
    fetches: [(id) => tab.app.planning.cycles.fetchCycles(id), (id) => tab.app.planning.cycles.fetchArchivedCycles(id)],
    name: (id) => tab.app.planning.cycles.getCycleById(id)?.name,
    off: { cycle_view: false },
  },
  modules: {
    row: (name, archived = false) =>
      moduleRowOf(moduleOf(tab.project, name, archived ? { archived_at: ARCHIVED } : {})),
    path: () => moduleRequests.list(tab.project),
    keys: ["PROJECT_MODULES", "PROJECT_ARCHIVED_MODULES"],
    fetches: [
      (id) => tab.app.planning.modules.fetchModules(id),
      (id) => tab.app.planning.modules.fetchArchivedModules(id),
    ],
    name: (id) => tab.app.planning.modules.getModuleById(id)?.name,
    off: { module_view: false },
  },
};

/** A work item's values of the domain, in the project, shown as the detail shows them: an observer of the store. */
const Reader = observer(function Reader({ domain, ids }: { domain: PlanningDomain; ids: string[] }) {
  usePlanningNames("ws", tab.project.id, domain, ids);
  return null;
});
/** The read another page makes of the project's list (its wrapper's) or of its archive (its archive page's). */
function PageRead({ domain, at }: { domain: PlanningDomain; at: 0 | 1 }) {
  const { keys, fetches } = DOMAINS[domain];
  useSessionSWR([keys[at], tab.project.id], (id) => fetches[at](id));
  return null;
}

const show = (node: ReactNode) =>
  act(async () => renderer.render(<SWRConfig value={{ provider: () => new Map() }}>{node}</SWRConfig>));
const settle = () => act(async () => vi.advanceTimersByTimeAsync(50));
/** The domain's reads so far, by their query: {} for the project's list, archived=true for its archive. */
const reads = (domain: PlanningDomain) =>
  tab.nerve.calls
    .filter((call) => call.method === "GET" && call.path === DOMAINS[domain].path())
    .map((call) => call.query);
async function readsUntil(domain: PlanningDomain, count: number, what: string) {
  await act(async () => until(() => reads(domain).length >= count, what));
  await settle();
}
/** nerve answers the domain's read at its place with these rows. */
async function answer(domain: PlanningDomain, at: number, rows: { id: string }[]) {
  await act(async () => {
    tab.nerve.calls.filter((call) => call.path === DOMAINS[domain].path())[at]!.answer(json(200, { data: rows }));
    await vi.advanceTimersByTimeAsync(20);
  });
}
/** The caller's project as nerve now lists it, with fields. */
async function reproject(fields: Partial<Project>) {
  await act(async () => {
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, ...fields }]);
  });
  await settle();
}

describe.each(["cycles", "modules"] as const)("%s", (domain) => {
  it("reads the project's list while the store has none, then its archive, each once for two readers", async () => {
    const { row, name } = DOMAINS[domain];
    const kept = row("Kept");
    const shelved = row("Shelved", true);
    await show(
      <>
        <Reader domain={domain} ids={[shelved.id]} />
        <Reader domain={domain} ids={[kept.id, shelved.id]} />
      </>
    );
    await readsUntil(domain, 1, "the list's read");
    expect(reads(domain), "the project's list, once").toEqual([{}]);
    await answer(domain, 0, [kept]);
    await readsUntil(domain, 2, "the archive's read");
    expect(reads(domain), "then its archive, once").toEqual([{}, { archived: "true" }]);
    expect([name(kept.id), name(shelved.id)]).toEqual(["Kept", undefined]);
    await answer(domain, 1, [shelved]);
    await settle();
    expect(name(shelved.id)).toBe("Shelved");
    expect(reads(domain), "every value named: nothing more").toHaveLength(2);
  });

  it("reads nothing while the store names every value, and the archive alone once it has the project's list", async () => {
    const { row, fetches } = DOMAINS[domain];
    const kept = row("Kept");
    const shelved = row("Shelved", true);
    const listing = fetches[0](tab.project.id);
    await readsUntil(domain, 1, "the store's own read of the list");
    await answer(domain, 0, [kept]);
    await listing;
    tab.nerve.calls.length = 0;
    await show(<Reader domain={domain} ids={[kept.id]} />);
    await settle();
    expect(reads(domain)).toEqual([]);
    await show(
      <>
        <Reader domain={domain} ids={[kept.id]} />
        <Reader domain={domain} ids={[shelved.id]} />
      </>
    );
    await readsUntil(domain, 1, "the archive's read");
    expect(reads(domain)).toEqual([{ archived: "true" }]);
  });

  it("shares the reads of the project's wrapper and its archive's page, by their keys", async () => {
    const { row, name } = DOMAINS[domain];
    const kept = row("Kept");
    const shelved = row("Shelved", true);
    await show(
      <>
        <PageRead domain={domain} at={0} />
        <PageRead domain={domain} at={1} />
        <Reader domain={domain} ids={[shelved.id]} />
      </>
    );
    await readsUntil(domain, 2, "the pages' reads");
    expect(reads(domain)).toEqual([{}, { archived: "true" }]);
    await answer(domain, 0, [kept]);
    await settle();
    expect(reads(domain), "the archive's read is the page's").toHaveLength(2);
    await answer(domain, 1, [shelved]);
    await settle();
    expect(name(shelved.id)).toBe("Shelved");
    expect(reads(domain)).toHaveLength(2);
  });

  it("reads nothing for a caller without a role in the project, nor while the domain is off", async () => {
    const shelved = DOMAINS[domain].row("Shelved", true);
    await reproject({ member_role: null });
    await show(<Reader domain={domain} ids={[shelved.id]} />);
    await settle();
    expect(reads(domain), "no role").toEqual([]);
    await reproject(DOMAINS[domain].off);
    expect(reads(domain), "turned off").toEqual([]);
    await reproject({});
    await readsUntil(domain, 1, "a member's read, while on");
    expect(reads(domain)).toEqual([{}]);
  });
});
