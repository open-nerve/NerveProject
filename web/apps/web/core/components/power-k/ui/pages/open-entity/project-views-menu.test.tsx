// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { Command } from "cmdk";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import type { TPowerKContext } from "@/components/power-k/core/types";
import { json, problem } from "@/lib/auth/fake-nerve";
import { useSessionSWR } from "@/lib/use-session-swr";
import { its, type Caller } from "@/store/user/fake-permissions";
import { ME, loadViews, viewOf, viewRequests, viewsTab } from "@/store/views/fake-views";
import { PowerKOpenProjectViewsMenu } from "./project-views-menu";

// The command palette's page that opens one of the route's project's views (P6a spec 2.8, R13): nerve's list of the
// project's views, by the views list's read, whose key it shares (["PROJECT_VIEWS", projectId]), for those the ways
// into the project's views are offered to (viewEntryOn: its views turned on, an admin's, a member's or a guest's role;
// views-entry.test.ts has its table). It waits for nerve's answer, says why nerve could not give it, and says so of a
// project of none; while the gate is closed it says why at once and reads nothing. The caller's role is the permission
// store's own, the views a views store over a fake nerve that answers each request when the test says (fake-views.ts).

type Tab = Awaited<ReturnType<typeof viewsTab>>;
const held = vi.hoisted(() => ({ tab: undefined as unknown as Tab, select: vi.fn() }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "x" }) }));
vi.mock("@/hooks/store/use-views", () => ({ useViews: () => held.tab.views }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => held.tab.permissions }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => held.tab.projects }));

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
/** nerve's reads of web's views so far. */
const reads = () => held.tab.nerve.calls.filter((call) => call.path === viewRequests.list(held.tab.web));

/** The views list's read of the project's views (its page.tsx), by its key. */
function ListPageRead({ projectId }: { projectId: string }) {
  useSessionSWR(["PROJECT_VIEWS", projectId], (id) => held.tab.views.fetchProjectViews(id));
  return null;
}

/**
 * The caller of a role in acme and in web opens the page at web's address, web being as fields say; beside the views
 * list's read of web's views when beside says so.
 */
async function opened(caller: Caller, fields: Partial<Project> = {}, beside = false) {
  held.tab = await viewsTab(caller, fields);
  const { acme, web } = held.tab;
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
        {beside && <ListPageRead projectId={web.id} />}
        <Command shouldFilter={false}>
          <Command.List>
            <PowerKOpenProjectViewsMenu context={context} handleSelect={held.select} />
          </Command.List>
        </Command>
      </SWRConfig>
    )
  );
  await settle();
  return held.tab;
}

it("lists the route's project's views to its guest as nerve answers the read, in nerve's order, and opens the one chosen", async () => {
  const { web } = await opened(its.guest);
  expect(reads()).toHaveLength(1);
  expect(host.querySelector('[role="status"]')).not.toBeNull();
  const bugs = viewOf(web, "Bugs", "u-ann");
  const mine = viewOf(web, "Mine", ME);
  await act(async () => reads()[0]!.answer(json(200, { data: [bugs, mine] })));
  await settle();
  expect(options().map((option) => option.textContent)).toEqual(["Bugs", "Mine"]);
  await act(async () => (options()[1] as HTMLElement).click());
  expect(held.select.mock.calls).toEqual([[mine]]);
});

it("shares the views list's read beside it, its key the list's: nerve is asked once", async () => {
  const { web } = await opened(its.member, {}, true);
  expect(reads()).toHaveLength(1);
  expect(host.querySelector('[role="status"]')).not.toBeNull();
  await act(async () => reads()[0]!.answer(json(200, { data: [viewOf(web, "Bugs", ME)] })));
  await settle();
  expect(options().map((option) => option.textContent)).toEqual(["Bugs"]);
  expect(reads()).toHaveLength(1);
});

it("says why nerve could not list the project's views, waiting no more", async () => {
  await opened(its.member);
  expect(reads()).toHaveLength(1);
  await act(async () => reads()[0]!.answer(problem(500, "internal_error")));
  await settle();
  expect(host.querySelector('[role="status"]')).toBeNull();
  expect(host.textContent).toBe("errors.internal_error");
});

it("says the project has no views when nerve lists none", async () => {
  await opened(its.admin);
  expect(reads()).toHaveLength(1);
  await act(async () => reads()[0]!.answer(json(200, { data: [] })));
  await settle();
  expect(host.querySelector('[role="status"]')).toBeNull();
  expect(host.textContent).toBe("power_k.menus.no_views");
});

// While the gate is closed the page reads nothing and says why at once, waiting for nothing: the views turned off, in
// the words the views list says it in (ViewsDisabled); no role in the project, as nerve answers its reads.
it.each([
  ["its views are turned off", its.member, { issue_views_view: false }, "disabled_project.empty_state.view.title"],
  ["the caller is not its member", its.outsider, {}, "errors.project_not_found"],
] as const)("reads none of the project's views and says why when %s", async (_, caller, fields, why) => {
  await opened(caller, fields);
  expect(reads()).toEqual([]);
  expect(host.querySelector('[role="status"]')).toBeNull();
  expect(host.textContent).toBe(why);
});

it("says why rather than list the views the store still holds once they are turned off", async () => {
  const { web, views, nerve } = await opened(its.member, { issue_views_view: false });
  await loadViews(nerve, views, web, [viewOf(web, "Bugs", ME)]);
  await settle();
  expect(views.getProjectViews(web.id)).toHaveLength(1);
  expect(options()).toEqual([]);
  expect(host.textContent).toBe("disabled_project.empty_state.view.title");
});
