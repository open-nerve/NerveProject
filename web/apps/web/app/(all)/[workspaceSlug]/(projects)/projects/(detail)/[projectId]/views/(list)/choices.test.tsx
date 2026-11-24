// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode } from "react";
import { runInAction } from "mobx";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { settle } from "@/components/common/fake-quick-actions";
import { listViews } from "@/components/views/fake-view-pages";
import type { ProjectStore } from "@/store/project/project.store";
import { its } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { ME, viewOf, viewsTab } from "@/store/views/fake-views";
import type { ViewStore } from "@/store/views/view.store";
import { ListPages, standInMembers } from "./fake-list-page";

// What the person chooses of a project's views list (P6a spec 2.3): the search of the views' names, whatever its case;
// the filters, by their owners (the project's members, its guests too) and the dates of their creation, applied above
// the list where they clear; and their order. The panel of the filters speaks the page's language. The caller is web's
// member (fake-permissions.ts); the views a views' store's over a fake nerve (fake-view-pages.tsx); the members a
// stand-in's (fake-list-page.tsx), whose arrival the test holds where it says.

const held = vi.hoisted(() => ({
  views: undefined as unknown as ViewStore,
  permissions: undefined as IUserPermissionStore | undefined,
  projects: undefined as ProjectStore | undefined,
  members: undefined as unknown as ReturnType<typeof import("./fake-list-page").standInMembers>,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "x" }) }));
vi.mock("@/hooks/store/use-views", () => ({ useViews: () => held.views }));
// the projects as the caller's workspace lists them, its projects fetched
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({ loader: "loaded", getProjectById: (id: string) => held.projects?.getProjectById(id) }),
}));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => held.members }));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => held.permissions,
  useUser: () => ({ data: { id: "u-me" } }),
}));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => null }));
vi.mock("@/components/assets/profile-image", () => ({ ProfileAvatar: () => null }));
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => null }));
vi.mock("@/components/core/app-header", () => ({
  AppHeader: (props: { header: ReactNode; mobileHeader?: ReactNode }) => (
    <>
      <header>{props.header}</header>
      <nav>{props.mobileHeader}</nav>
    </>
  ),
}));

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers({ now: new Date("2026-11-24T12:00:00Z") });
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  held.members = standInMembers();
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

/** The names of the views the list shows, in its order. */
const rows = () => [...host.querySelectorAll("a[href*='/views/v-']")].map((link) => link.textContent);
const header = () => host.querySelector("header") ?? undefined;
const named = (name: string, within: Element | undefined = host) =>
  [...(within?.querySelectorAll<HTMLButtonElement>("button") ?? [])].find(
    (one) => (one.getAttribute("aria-label") ?? one.textContent) === name
  );
const click = async (button: HTMLElement | undefined | null) => {
  await act(async () => button?.click());
  await settle();
};
/** Types text into input, as a person does. */
async function type(input: HTMLInputElement | null, text: string) {
  await act(async () => {
    if (!input) return;
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set?.call(input, text);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}
/** The options of the open filters' panel, by their titles. */
const options = () => [...(header()?.querySelectorAll<HTMLButtonElement>("button[aria-pressed]") ?? [])];
const option = (title: string) => options().find((one) => one.textContent === title);
/** Opens the header's menu of the list's order and chooses the item of title. */
async function order(title: string) {
  const button = [...(header()?.querySelectorAll<HTMLButtonElement>('button[aria-haspopup="menu"]') ?? [])][0];
  await click(button);
  const menu = document.getElementById(button?.getAttribute("aria-controls") ?? "");
  await click(
    [...(menu?.querySelectorAll<HTMLElement>('[role="menuitem"]') ?? [])].find((one) => one.textContent === title)
  );
}

/** web's member at web's views list, nerve listing these views of web. */
async function listed(views: (web: Parameters<typeof viewOf>[0]) => View[]) {
  const tab = await viewsTab(its.member);
  Object.assign(held, { views: tab.views, permissions: tab.permissions, projects: tab.projects });
  await act(async () => root.render(<ListPages path={`/${tab.acme.slug}/projects/${tab.web.id}/views`} />));
  await settle();
  await listViews(tab.nerve, tab.web, views(tab.web));
  return tab;
}
const three = (web: Parameters<typeof viewOf>[0]) => [
  viewOf(web, "Bugs", ME, { updated_at: "2026-11-03T09:00:00Z" }),
  viewOf(web, "Mine", "u-ann", { updated_at: "2026-11-05T09:00:00Z" }),
  viewOf(web, "Triage", "u-gus", { updated_at: "2026-11-04T09:00:00Z", created_at: "2026-11-23T09:00:00Z" }),
];

it("searches the views' names as typed, whatever its case", async () => {
  await listed(three);
  await type(header()?.querySelector("input") ?? null, "bUG");
  expect(rows()).toEqual(["Bugs"]);
  await type(header()?.querySelector("input") ?? null, "");
  expect(rows()).toEqual(["Mine", "Triage", "Bugs"]);
});

it("orders the views as chosen, the last changed first as the page opens", async () => {
  await listed(three);
  expect(rows()).toEqual(["Mine", "Triage", "Bugs"]);
  await order("project_view.sort_by.name");
  expect(rows()).toEqual(["Triage", "Mine", "Bugs"]);
  await order("common.order_by.asc");
  expect(rows()).toEqual(["Bugs", "Mine", "Triage"]);
});

/** The names of the filters applied above the list, as it shows them. */
const appliedNames = () =>
  [
    ...(named("common.clear_all")?.closest(".flex-wrap")?.querySelectorAll(":scope > div > span:first-child") ?? []),
  ].map((name) => name.textContent);

it("filters the views by their owners, the project's guests among them, applied above the list where they clear", async () => {
  await listed(three);
  await click(named("common.filters", header()));
  expect(held.members.asked).toContainEqual(["p-web", true]);
  expect(held.members.asked).not.toContainEqual(["p-web", false]);
  // six members: five shown, then all
  await click(named("common.view_all", header()));
  expect(options().map((one) => one.textContent)).toEqual([
    "1 week ago",
    "2 weeks ago",
    "1 month ago",
    "common.filter_options.custom",
    "you",
    "Ann",
    "Bob",
    "Cat",
    "Dee",
    "Gus",
  ]);
  await click(option("Gus"));
  expect(rows()).toEqual(["Triage"]);
  // the list's filters applied, above it, by the name the panel gives them: the owners', Gus
  expect(appliedNames()).toEqual(["common.created_by"]);
  expect(named("common.clear_all")?.closest(".flex-wrap")?.textContent).toContain("Gus");
  await click(named("common.clear_all"));
  expect(rows()).toEqual(["Mine", "Triage", "Bugs"]);
  expect(named("common.clear_all")).toBeUndefined();
});

it("filters the views by the dates of their creation", async () => {
  await listed(three);
  await click(named("common.filters", header()));
  await click(option("1 week ago"));
  expect(rows()).toEqual(["Mine", "Bugs"]);
  expect(appliedNames()).toEqual(["inbox_issue.filters.created_at"]);
});

it("speaks the page's language in the filters' panel", async () => {
  await listed(three);
  expect(header()?.querySelector("input")?.getAttribute("placeholder")).toBe("common.search.label");
  await click(named("common.filters", header()));
  const panel = header()?.querySelector<HTMLInputElement>("input[type='text']");
  expect(panel?.getAttribute("placeholder")).toBe("common.search.label");
  expect(named("inbox_issue.filters.created_at", header())).toBeDefined();
  expect(named("common.created_by", header())).toBeDefined();
  await type(panel ?? null, "zzz");
  expect(header()?.textContent?.match(/common\.search\.no_matches_found/g)).toHaveLength(2);
});

it("lists the project's members in the panel once they arrive", async () => {
  await listed(three);
  runInAction(() => held.members.arrived.set(false));
  await click(named("common.filters", header()));
  expect(option("Ann")).toBeUndefined();
  await act(async () => runInAction(() => held.members.arrived.set(true)));
  await settle();
  expect(option("Ann")).toBeDefined();
});
