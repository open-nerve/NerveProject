// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { Command } from "cmdk";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import type { TPowerKContext } from "@/components/power-k/core/types";
import { listedCycles } from "@/components/cycles/fake-cycle-lists";
import { refusal } from "@/lib/fake-refusal";
import { useSessionSWR } from "@/lib/use-session-swr";
import { rowOf } from "@/store/planning/fake-cycles";
import { cycleOf } from "@/store/project/fake-projects";
import type { ProjectStore } from "@/store/project/project.store";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { PowerKOpenProjectCyclesMenu } from "./project-cycles-menu";

// The command palette's page that opens one of the route's project's cycles (P5a spec 2.7): nerve's list of the
// project's cycles not archived, by the read the project's pages make (its wrapper's, whose key it shares), a member's
// of a project not archived whose cycles are on. The caller's role is the permission store's own (fake-permissions.ts),
// the cycles a stand-in store's (fake-cycle-lists.ts) whose reads nerve answers when the test says.

const held = vi.hoisted(() => ({
  cycles: undefined as unknown as ReturnType<typeof listedCycles>,
  permissions: undefined as IUserPermissionStore | undefined,
  projects: undefined as ProjectStore | undefined,
  project: {} as Partial<Project>,
  select: vi.fn(),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "x" }) }));
vi.mock("@/hooks/store/use-cycles", () => ({ useCycles: () => held.cycles }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => held.permissions }));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    getProjectById: (id: string | null | undefined): Project | undefined => {
      const project = held.projects?.getProjectById(id);
      return project && { ...project, ...held.project };
    },
  }),
}));

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
const scrollDescriptor = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "scrollIntoView");
beforeEach(() => {
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
  held.select.mockReset();
  held.cycles = listedCycles();
  held.project = {};
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  // oxlint-disable-next-line no-extend-native -- restores jsdom's own after the stand-in.
  if (scrollDescriptor) Object.defineProperty(HTMLElement.prototype, "scrollIntoView", scrollDescriptor);
  else Reflect.deleteProperty(HTMLElement.prototype, "scrollIntoView");
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

const settle = () => act(async () => vi.advanceTimersByTimeAsync(100));
const options = () => [...host.querySelectorAll('[role="option"]')];

/** The read of the project's cycles its wrapper makes on its pages (use-project-fetch.ts), by its key. */
function WrapperRead({ projectId }: { projectId: string }) {
  useSessionSWR(["PROJECT_CYCLES", projectId], (id) => held.cycles.fetchCycles(id));
  return null;
}

/**
 * The caller of a role in acme and in web opens the page at web's address, web being as project says; on a page of the
 * project (beside its wrapper's read) when wrapped says so.
 */
async function opened(caller: Caller, project: Partial<Project> = {}, wrapped = false) {
  const { permissions, projects, acme, web } = await callerInWeb(caller.workspaceRole, caller.projectRole);
  Object.assign(held, { permissions, projects, project });
  const context: TPowerKContext = {
    params: { workspaceSlug: acme.slug, projectId: web.id },
    activeCommand: null,
    activeContext: null,
    shouldShowContextBasedActions: true,
    setShouldShowContextBasedActions: () => {},
    navigate: () => {},
    closePalette: () => {},
    setActiveCommand: () => {},
    setActivePage: () => {},
  };
  await act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        {wrapped && <WrapperRead projectId={web.id} />}
        <Command shouldFilter={false}>
          <Command.List>
            <PowerKOpenProjectCyclesMenu context={context} handleSelect={held.select} />
          </Command.List>
        </Command>
      </SWRConfig>
    )
  );
  await settle();
  return web;
}

it("lists the route's project's cycles as nerve answers the read its pages make, and opens the one chosen", async () => {
  const web = await opened(its.member);
  expect(held.cycles.asked()).toEqual([`cycles of ${web.id}`]);
  expect(host.querySelector('[role="status"]')).not.toBeNull();
  const sprint = cycleOf(web, "Sprint", { status: "current" });
  const plan = cycleOf(web, "Plan", { status: "upcoming" });
  await held.cycles.answer("cycles", [rowOf(sprint), rowOf(plan)]);
  await settle();
  expect(options().map((option) => option.textContent)).toEqual(["Sprint", "Plan"]);
  await act(async () => (options()[1] as HTMLElement).click());
  expect(held.select.mock.calls).toEqual([[rowOf(plan)]]);
});

it("shares its project's wrapper's read, on a page of the project: nerve is asked once", async () => {
  const web = await opened(its.member, {}, true);
  expect(held.cycles.asked()).toEqual([`cycles of ${web.id}`]);
});

it("says why nerve could not list the project's cycles, waiting no more", async () => {
  await opened(its.guest);
  await held.cycles.refuse("cycles", refusal(409, "planning.cycles_disabled"));
  await settle();
  expect(host.querySelector('[role="status"]')).toBeNull();
  expect(host.textContent).toBe("errors.planning_cycles_disabled");
});

it.each([
  ["its cycles are turned off", its.member, { cycle_view: false }],
  ["it is archived", its.member, { archived_at: "2026-10-31T00:00:00Z" }],
  ["the caller is not its member", its.outsider, {}],
] as const)("reads nothing of the project's cycles when %s", async (_, caller, project) => {
  await opened(caller, project);
  expect(held.cycles.asked()).toEqual([]);
});
