// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { ModulesListMobileHeader } from "@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(list)/mobile-header";
import { moduleRowOf } from "@/store/planning/fake-modules";
import { ModuleFilterStore } from "@/store/planning/module_filter.store";
import { moduleOf, projectOf } from "@/store/project/fake-projects";
import { listedModules } from "./fake-module-lists";
import { ModuleViewHeader } from "./module-view-header";
import { ModulesListView } from "./modules-list-view";

// How a project's modules list is laid out and ordered (P6a spec 2.4, 2.6, R4, R14), as a person chooses it: by the
// layout buttons of the list's header or the layout menu of a small screen's, and by the header's order menu, by the
// keyboard too: an option, the lowest first or, by its direction, the highest first; the manual order has no
// direction. The list follows what they choose, which the browser keeps for the project (useModuleDisplay). The
// modules are a stand-in store's; each row and card a stand-in that says which it is; the header's search and filters
// the session's (module-filters.test.tsx).

const held = vi.hoisted(() => ({
  modules: undefined as unknown as ReturnType<typeof import("./fake-module-lists").listedModules>,
  filters: undefined as unknown as import("@/store/planning/module_filter.store").ModuleFilterStore,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/hooks/store/use-modules", () => ({ useModules: () => held.modules }));
vi.mock("@/hooks/store/use-module-filters", () => ({ useModuleFilters: () => held.filters }));
vi.mock("@/hooks/use-planning-reads", () => ({ usePlanningReads: () => ({ readable: true, listed: true }) }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ workspace: { getWorkspaceMemberIds: () => [] } }) }));
vi.mock("./module-list-item", () => ({
  ModuleListItem: (props: { moduleId: string }) => <p data-shown="row">{props.moduleId}</p>,
}));
vi.mock("./module-card-item", () => ({
  ModuleCardItem: (props: { moduleId: string }) => <p data-shown="card">{props.moduleId}</p>,
}));
vi.mock("./module-peek-overview", () => ({ ModulePeekOverview: () => null, usePeekModule: () => [null, () => null] }));

const web = projectOf("WEB", "id-acme");
// web's modules in nerve's order: by neither their names, their progress nor their manual order
const beta = moduleRowOf(moduleOf(web, "Beta", { sort_order: 3 }), { total_issues: 4, completed_issues: 1 });
const alpha = moduleRowOf(moduleOf(web, "alpha", { sort_order: 2 }), { total_issues: 2, completed_issues: 2 });
const gamma = moduleRowOf(moduleOf(web, "Gamma", { sort_order: 1 }));

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers();
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  localStorage.clear();
  held.modules = listedModules();
  held.modules.show("modules", web.id, [beta, alpha, gamma]);
  held.filters = new ModuleFilterStore();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  await act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <MemoryRouter initialEntries={[`/acme/projects/${web.id}/modules`]}>
          <header>
            <ModuleViewHeader workspaceSlug="acme" projectId={web.id} list="default" />
          </header>
          <nav>
            <ModulesListMobileHeader workspaceSlug="acme" projectId={web.id} />
          </nav>
          <ModulesListView workspaceSlug="acme" projectId={web.id} />
        </MemoryRouter>
      </SWRConfig>
    )
  );
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

/** What the list shows of web's modules: rows or cards, and the modules, in its order. */
const shown = () => {
  const items = [...host.querySelectorAll<HTMLElement>("[data-shown]")];
  return { as: [...new Set(items.map((item) => item.dataset.shown))], modules: items.map((item) => item.textContent) };
};
const ids = (...rows: { id: string }[]) => rows.map((row) => row.id);
const button = (name: string) =>
  [...host.querySelectorAll("button")].find((one) => (one.getAttribute("aria-label") ?? one.textContent) === name);
const orderButton = () =>
  [...host.querySelectorAll("header button")].find((one) =>
    one.getAttribute("aria-label")?.startsWith("common.order_by.label")
  ) as HTMLButtonElement | undefined;
const menu = () => document.querySelector<HTMLElement>('[role="menu"]');
const items = () => [...(menu()?.querySelectorAll<HTMLElement>('[role="menuitem"]') ?? [])];
const active = () => document.getElementById(menu()?.getAttribute("aria-activedescendant") ?? "")?.textContent;

/** A key pressed on target, as a keyboard does it: down, then up (a button's Enter and Space click it). */
async function press(target: Element, key: string) {
  await act(async () => {
    const down = new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true });
    target.dispatchEvent(down);
    const up = new KeyboardEvent("keyup", { key, bubbles: true, cancelable: true });
    target.dispatchEvent(up);
    if (!down.defaultPrevented && !up.defaultPrevented && key === "Enter" && target instanceof HTMLButtonElement)
      target.click();
    await vi.advanceTimersByTimeAsync(50);
  });
}
/** Moves the open menu's active item down to the one of title, at most steps times. */
async function arrowTo(title: string, steps: number): Promise<void> {
  const open = menu();
  if (!open || active() === title || steps === 0) return;
  await press(open, "ArrowDown");
  await arrowTo(title, steps - 1);
}
/** Chooses the order's item of title by the keyboard: the menu opened by Enter, the arrows to it, Enter on it. */
async function orderByKeyboard(title: string) {
  const opener = orderButton();
  if (!opener) throw new Error("No order button");
  await act(async () => opener.focus());
  await press(opener, "Enter");
  await arrowTo(title, 10);
  if (active() !== title) throw new Error(`The order offers no ${title}`);
  await press(menu() ?? opener, "Enter");
}
/** Chooses the order's item of title by the pointer. */
async function orderBy(title: string) {
  await act(async () => orderButton()?.click());
  await act(async () => vi.advanceTimersByTimeAsync(50));
  const item = items().find((one) => one.textContent === title);
  if (!item) throw new Error(`The order offers no ${title}`);
  await act(async () => item.click());
  await act(async () => vi.advanceTimersByTimeAsync(50));
}

it("lays the modules out as rows, by name, until the person chooses otherwise", () => {
  expect(shown()).toEqual({ as: ["row"], modules: ids(alpha, beta, gamma) });
  expect(button("project_modules.layout.list")?.getAttribute("aria-pressed")).toBe("true");
  expect(button("project_modules.layout.board")?.getAttribute("aria-pressed")).toBe("false");
});

it("lays them out as cards, and back as rows, by the header's layout buttons", async () => {
  await act(async () => button("project_modules.layout.board")?.click());
  expect(shown()).toEqual({ as: ["card"], modules: ids(alpha, beta, gamma) });
  expect(button("project_modules.layout.board")?.getAttribute("aria-pressed")).toBe("true");
  await act(async () => button("project_modules.layout.list")?.click());
  expect(shown()).toEqual({ as: ["row"], modules: ids(alpha, beta, gamma) });
});

it("lays them out by a small screen's layout menu too", async () => {
  const layoutMenu = [...host.querySelectorAll("nav button")].find((one) => one.textContent === "common.layout");
  await act(async () => (layoutMenu as HTMLButtonElement | undefined)?.click());
  await act(async () => vi.advanceTimersByTimeAsync(50));
  const gallery = items().find((one) => one.textContent === "project_modules.layout.board");
  await act(async () => gallery?.click());
  expect(shown()).toEqual({ as: ["card"], modules: ids(alpha, beta, gamma) });
});

it("orders them by the option the keyboard chooses in the header's order menu, the lowest first", async () => {
  expect(orderButton()?.getAttribute("aria-label")).toBe("common.order_by.label: project_modules.order_by.name");
  expect(orderButton()?.getAttribute("aria-expanded")).toBe("false");
  await orderByKeyboard("project_modules.order_by.progress");
  expect(shown().modules).toEqual(ids(gamma, beta, alpha));
  expect(orderButton()?.getAttribute("aria-label")).toBe("common.order_by.label: project_modules.order_by.progress");
  expect(menu()).toBeNull();
  expect(document.activeElement).toBe(orderButton());
});

it("turns the order around by its direction, which another option keeps", async () => {
  await orderBy("common.order_by.desc");
  expect(shown().modules).toEqual(ids(gamma, beta, alpha));
  await orderBy("project_modules.order_by.progress");
  expect(shown().modules).toEqual(ids(alpha, beta, gamma));
  await orderByKeyboard("common.order_by.asc");
  expect(shown().modules).toEqual(ids(gamma, beta, alpha));
});

it("orders them manually with no direction to choose, though the order before was the highest first", async () => {
  await orderBy("common.order_by.desc");
  await orderBy("project_modules.order_by.manual");
  expect(shown().modules).toEqual(ids(gamma, alpha, beta));
  await act(async () => orderButton()?.click());
  await act(async () => vi.advanceTimersByTimeAsync(50));
  expect(items().map((one) => one.textContent)).toEqual([
    "project_modules.order_by.name",
    "project_modules.order_by.progress",
    "project_modules.order_by.issues",
    "project_modules.order_by.due_date",
    "project_modules.order_by.created_at",
    "project_modules.order_by.manual",
  ]);
});
