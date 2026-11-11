// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { CycleFilterStore } from "@/store/planning/cycle_filter.store";
import { CycleAppliedFiltersList } from "./applied-filters";
import { CyclesViewHeader } from "./cycles-view-header";

// What a project's cycles list and its archive are searched and filtered by (P5a spec 2.3, 2.5, R4): the search, and
// the filters' menu, each of the project's list it is on, kept by the session's filters (CycleFilterStore, the real
// one); the options of the menu under titles in the page's language, nerve's status (the list's alone: each archived
// cycle has ended) and the days a cycle starts and ends; and the filters applied, each removable, all clearable.

const held = vi.hoisted(() => ({ filters: undefined as unknown as CycleFilterStore }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@/hooks/store/use-cycle-filters", () => ({ useCycleFilters: () => held.filters }));

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers({ now: new Date(2026, 10, 11, 12) });
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  held.filters = new CycleFilterStore();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

const render = (node: ReactNode) => act(async () => root.render(node));
const settle = () => act(async () => vi.advanceTimersByTimeAsync(100));
const button = (name: string) =>
  [...host.querySelectorAll("button")].find((one) => (one.getAttribute("aria-label") ?? one.textContent) === name);
const field = (name: string) => host.querySelector<HTMLInputElement>(`input[aria-label="${name}"]`);
/** Types text into input, as a person does. */
async function type(input: HTMLInputElement | null, text: string) {
  await act(async () => {
    if (!input) return;
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set?.call(input, text);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}
const press = (input: HTMLInputElement | null, key: string) =>
  act(async () => input?.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true })));
/** The titles of the filters' menu, in its order. */
const titles = () =>
  [...host.querySelectorAll("button[aria-expanded]")].slice(1).map((one) => one.getAttribute("aria-label"));
const option = (title: string) =>
  [...host.querySelectorAll<HTMLButtonElement>("button[aria-pressed]")].find((one) => one.textContent === title);

it("searches the project's list as typed; Escape clears the search, then closes it", async () => {
  await render(<CyclesViewHeader projectId="p-web" list="default" />);
  await act(async () => button("common.search.label")?.click());
  await type(field("common.search.label"), "Sprint");
  expect(held.filters.getSearchQuery("p-web", "default")).toBe("Sprint");
  expect(held.filters.getSearchQuery("p-web", "archived")).toBe("");
  expect(held.filters.getSearchQuery("p-ops", "default")).toBe("");
  await press(field("common.search.label"), "Escape");
  expect(held.filters.getSearchQuery("p-web", "default")).toBe("");
  expect(field("common.search.label")).not.toBeNull();
  await press(field("common.search.label"), "Escape");
  expect(field("common.search.label")).toBeNull();
  expect(button("common.search.label")).not.toBeUndefined();
});

it("keeps the search open on the project's search kept from before", async () => {
  held.filters.updateSearchQuery("p-web", "archived", "S0");
  await render(<CyclesViewHeader projectId="p-web" list="archived" />);
  expect(field("common.search.label")?.value).toBe("S0");
  await act(async () => button("common.search.clear")?.click());
  expect(held.filters.getSearchQuery("p-web", "archived")).toBe("");
  expect(field("common.search.label")).toBeNull();
});

it("filters the project's list by nerve's status and its days, each option on and off, under translated titles", async () => {
  await render(<CyclesViewHeader projectId="p-web" list="default" />);
  await act(async () => button("common.filters")?.click());
  await settle();
  expect(titles()).toEqual(["project_cycles.filters.status", "start_date", "end_date"]);
  // the menu opens on its button, not on its search: Tab goes into it (FiltersDropdown)
  expect(field("common.filter_options.search")).not.toBeNull();
  expect(document.activeElement).not.toBe(field("common.filter_options.search"));
  await act(async () => option("project_cycles.status.draft")?.click());
  await act(async () => option("project_cycles.status.yet_to_start")?.click());
  await act(async () => option("1 week from now")?.click());
  expect(held.filters.getFilters("p-web", "default")).toEqual({
    status: ["draft", "upcoming"],
    start_date: ["1_weeks;after;fromnow"],
  });
  await act(async () => option("project_cycles.status.draft")?.click());
  expect(held.filters.getFilters("p-web", "default").status).toEqual(["upcoming"]);
  expect(held.filters.getFilters("p-web", "archived")).toEqual({});
  expect(held.filters.getFilters("p-ops", "default")).toEqual({});
  expect(option("project_cycles.status.yet_to_start")?.getAttribute("aria-pressed")).toBe("true");
});

it("finds the menu's options by what is typed into its search, as the page names them", async () => {
  await render(<CyclesViewHeader projectId="p-web" list="default" />);
  await act(async () => button("common.filters")?.click());
  await settle();
  // "yet" is in the name of the status upcoming ("yet to start"), not in its value
  await type(field("common.filter_options.search"), "yet");
  expect(option("project_cycles.status.yet_to_start")).not.toBeUndefined();
  expect(option("project_cycles.status.draft")).toBeUndefined();
  expect(option("1 week from now")).toBeUndefined();
  expect(host.textContent).toContain("common.search.no_matches_found");
});

it("filters the project's archive by its days alone", async () => {
  await render(<CyclesViewHeader projectId="p-web" list="archived" />);
  await act(async () => button("common.filters")?.click());
  await settle();
  expect(titles()).toEqual(["start_date", "end_date"]);
  await act(async () => option("2 weeks from now")?.click());
  expect(held.filters.getFilters("p-web", "archived")).toEqual({ start_date: ["2_weeks;after;fromnow"] });
  expect(held.filters.getFilters("p-web", "default")).toEqual({});
});

it("says the filters applied, by translated names, each removable, and clears them all", async () => {
  held.filters.updateFilters("p-web", "default", {
    status: ["draft", "current"],
    end_date: ["2026-11-30;before"],
  });
  held.filters.updateFilters("p-web", "archived", { start_date: ["1_weeks;after;fromnow"] });
  await render(<CycleAppliedFiltersList projectId="p-web" list="default" />);
  const said = host.textContent ?? "";
  expect(said).toContain("project_cycles.filters.status");
  expect(said).toContain("project_cycles.status.draft");
  expect(said).toContain("end_date");
  expect(said).toContain("Before Nov 30, 2026");
  await act(async () => button("common.remove_item project_cycles.status.draft")?.click());
  expect(held.filters.getFilters("p-web", "default").status).toEqual(["current"]);
  await act(async () => button("common.remove_item end_date")?.click());
  expect(held.filters.getFilters("p-web", "default").end_date).toEqual([]);
  await act(async () => button("common.clear_all")?.click());
  expect(held.filters.getFilters("p-web", "default")).toEqual({});
  expect(held.filters.getFilters("p-web", "archived")).toEqual({ start_date: ["1_weeks;after;fromnow"] });
  expect(host.innerHTML).toBe("");
});
