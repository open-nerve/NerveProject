// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { afterAll, afterEach, beforeAll, beforeEach, expect, it, vi } from "vitest";
import { ModuleFilterStore, type ModuleListKind } from "@/store/planning/module_filter.store";
import { ModuleAppliedFiltersList } from "./applied-filters/root";
import { membersAtAnotherRouter } from "./fake-module-members";
import { ModuleViewHeader } from "./module-view-header";

// What a project's modules list and its archive are searched and filtered by (P6a spec 2.4, 2.6, R4, R14): the
// search, and the filters' menu, each of the project's list it is on, kept by the session's filters (ModuleFilterStore,
// the real one); the options of the menu under titles in the page's language: nerve's status (the list's alone, as
// the old page had it), the lead and the members, who are the members of the address's workspace (the member store's
// own, whose router names another, fake-module-members.ts), and the days a module starts and is due; and the filters
// applied, each removable, all clearable. The days are the project's calendar days, in a zone behind UTC.

const held = vi.hoisted(() => ({
  filters: undefined as unknown as import("@/store/planning/module_filter.store").ModuleFilterStore,
  members: undefined as unknown as () => unknown,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@/hooks/store/use-module-filters", () => ({ useModuleFilters: () => held.filters }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => held.members() }));
vi.mock("@/hooks/store/user", () => ({ useUser: () => ({ data: { id: "u-me" } }) }));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => ({ id: "id-acme" }) }));
vi.mock("@/components/assets/profile-image", () => ({
  ProfileAvatar: (props: { alt?: string }) => <i title={props.alt} />,
}));

const WEB = "p-web";
const OPS = "p-ops";

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
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
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  localStorage.clear();
  held.filters = new ModuleFilterStore();
  held.members = (await membersAtAnotherRouter()).useMember;
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
/** The header of web's list, or of its archive, at an address of acme. */
const header = (list: ModuleListKind) => render(<ModuleViewHeader workspaceSlug="acme" projectId={WEB} list={list} />);
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
/** The filters' button, named by its title. */
const filtersButton = () => button("common.filters");
async function openFilters() {
  await act(async () => filtersButton()?.click());
  await settle();
}
/** Each filter of the menu, in its order: its title (its header's button) and the options under it. */
const sections = () =>
  [...host.querySelectorAll("button[aria-expanded]")]
    .filter((one) => !one.getAttribute("aria-label")?.startsWith("common."))
    .map((one) => ({
      title: one.getAttribute("aria-label"),
      options: [...(one.closest(".py-2")?.querySelectorAll<HTMLButtonElement>("button[aria-pressed]") ?? [])],
    }));
const titles = () => sections().map(({ title }) => title);
/** The options of the filter of title, by what they say. */
const optionsOf = (title: string) =>
  sections()
    .find((one) => one.title?.startsWith(title))
    ?.options.map((one) => one.textContent);
const option = (title: string, text: string) =>
  sections()
    .find((one) => one.title?.startsWith(title))
    ?.options.find((one) => one.textContent === text);

it("searches the project's list as typed; Escape clears the search, then closes it", async () => {
  await header("default");
  await act(async () => button("common.search.label")?.click());
  expect(document.activeElement).toBe(field("common.search.label"));
  await type(field("common.search.label"), "Auth");
  expect(held.filters.getSearchQuery(WEB, "default")).toBe("Auth");
  expect(held.filters.getSearchQuery(WEB, "archived")).toBe("");
  expect(held.filters.getSearchQuery(OPS, "default")).toBe("");
  await press(field("common.search.label"), "Escape");
  expect(held.filters.getSearchQuery(WEB, "default")).toBe("");
  expect(field("common.search.label")).not.toBeNull();
  await press(field("common.search.label"), "Escape");
  expect(field("common.search.label")).toBeNull();
  expect(button("common.search.label")).not.toBeUndefined();
});

it("keeps the archive's search open on its search kept from before; its clear closes it", async () => {
  held.filters.updateSearchQuery(WEB, "archived", "Old");
  await header("archived");
  expect(field("common.search.label")?.value).toBe("Old");
  await act(async () => button("common.search.clear")?.click());
  expect(held.filters.getSearchQuery(WEB, "archived")).toBe("");
  expect(field("common.search.label")).toBeNull();
});

it("names the filters' button, which says whether its menu is open; the menu's search waits for the person", async () => {
  await header("default");
  expect(filtersButton()?.getAttribute("aria-expanded")).toBe("false");
  await openFilters();
  expect(filtersButton()?.getAttribute("aria-expanded")).toBe("true");
  expect(field("common.filter_options.search")).not.toBeNull();
  expect(document.activeElement).not.toBe(field("common.filter_options.search"));
  expect(host.querySelector("[autofocus]")).toBeNull();
  expect(titles()).toEqual(["project_module.status", "lead", "members", "start_date", "due_date"]);
});

it("offers the address's workspace members as leads and members, though the router names another workspace", async () => {
  await header("default");
  await openFilters();
  expect(optionsOf("lead")).toEqual(["Ann", "Bob"]);
  expect(optionsOf("members")).toEqual(["Ann", "Bob"]);
});

it("filters the project's list by status, lead, members and days, each option on and off, the list's own", async () => {
  await header("default");
  await openFilters();
  await act(async () => option("project_module.status", "project_modules.status.planned")?.click());
  await act(async () => option("project_module.status", "project_modules.status.in_progress")?.click());
  await act(async () => option("lead", "Ann")?.click());
  await act(async () => option("members", "Bob")?.click());
  await act(async () => option("start_date", "1 week from now")?.click());
  await act(async () => option("due_date", "2 weeks from now")?.click());
  expect(held.filters.getFilters(WEB, "default")).toEqual({
    status: ["planned", "in-progress"],
    lead: ["u-Ann"],
    members: ["u-Bob"],
    start_date: ["1_weeks;after;fromnow"],
    target_date: ["2_weeks;after;fromnow"],
  });
  await act(async () => option("project_module.status", "project_modules.status.planned")?.click());
  await act(async () => option("lead", "Ann")?.click());
  expect(held.filters.getFilters(WEB, "default")).toMatchObject({ status: ["in-progress"], lead: [] });
  expect(option("project_module.status", "project_modules.status.in_progress")?.getAttribute("aria-pressed")).toBe(
    "true"
  );
  expect(held.filters.getFilters(WEB, "archived")).toEqual({});
  expect(held.filters.getFilters(OPS, "default")).toEqual({});
});

it("finds the menu's options by what is typed into its search, as the page names them", async () => {
  await header("default");
  await openFilters();
  // "in_progress" is in the name of the status in-progress, not in its value
  await type(field("common.filter_options.search"), "in_progress");
  expect(optionsOf("project_module.status")).toEqual(["project_modules.status.in_progress"]);
  expect(optionsOf("lead")).toEqual([]);
  expect(optionsOf("start_date")).toEqual([]);
  await type(field("common.filter_options.search"), "bo");
  expect(optionsOf("lead")).toEqual(["Bob"]);
  expect(optionsOf("members")).toEqual(["Bob"]);
});

it("filters the project's archive by its lead, members and days, no status, the archive's own", async () => {
  held.filters.updateFilters(WEB, "default", { members: ["u-Bob"], target_date: ["1_months;after;fromnow"] });
  await header("archived");
  await openFilters();
  expect(titles()).toEqual(["lead", "members", "start_date", "due_date"]);
  // the list's filters are not the archive's
  expect(option("members", "Bob")?.getAttribute("aria-pressed")).toBe("false");
  await act(async () => option("members", "Ann")?.click());
  await act(async () => option("due_date", "1 month from now")?.click());
  expect(held.filters.getFilters(WEB, "archived")).toEqual({
    members: ["u-Ann"],
    target_date: ["1_months;after;fromnow"],
  });
  expect(held.filters.getFilters(WEB, "default")).toEqual({
    members: ["u-Bob"],
    target_date: ["1_months;after;fromnow"],
  });
});

it("says the filters applied, by translated names, each removable, and clears them all", async () => {
  // the browser is in Los Angeles, where 2026-11-30 read as a time in UTC is the 29th
  expect(new Date(2026, 10, 30).getTimezoneOffset()).toBe(480);
  held.filters.updateFilters(WEB, "default", {
    status: ["planned", "completed"],
    lead: ["u-Ann"],
    members: ["u-Bob", "u-gone"],
    target_date: ["2026-11-30;before"],
  });
  held.filters.updateFilters(WEB, "archived", { start_date: ["1_weeks;after;fromnow"] });
  await render(<ModuleAppliedFiltersList projectId={WEB} list="default" />);
  const said = host.textContent ?? "";
  for (const name of [
    "project_module.status",
    "project_modules.status.planned",
    "lead",
    "Ann",
    "members",
    "Bob",
    "distribution.former_member",
    "due_date",
    "Before Nov 30, 2026",
  ])
    expect(said).toContain(name);
  await act(async () => button("common.remove_item project_modules.status.planned")?.click());
  expect(held.filters.getFilters(WEB, "default").status).toEqual(["completed"]);
  await act(async () => button("common.remove_item Bob")?.click());
  expect(held.filters.getFilters(WEB, "default").members).toEqual(["u-gone"]);
  await act(async () => button("common.remove_item due_date")?.click());
  expect(held.filters.getFilters(WEB, "default").target_date).toEqual([]);
  await act(async () => button("common.clear_all")?.click());
  expect(held.filters.getFilters(WEB, "default")).toEqual({});
  expect(held.filters.getFilters(WEB, "archived")).toEqual({ start_date: ["1_weeks;after;fromnow"] });
  expect(host.innerHTML).toBe("");
});
