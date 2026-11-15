// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The quick add of a group of a cycle's or a module's page grouped by the other domain (P7b spec 2.5, R12; P7a R2): the
// new work item joins the group's module or cycle and the page's own cycle or module, in one request. The page, its
// layout, the quick add, the stores and HTTP are real.
import { act, createElement, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { listIssue } from "@/store/issue/fake-issues";
import { rowOf } from "@/store/planning/fake-cycles";
import { moduleRowOf } from "@/store/planning/fake-modules";
import { cycleOf, moduleOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import {
  C1,
  M1,
  answeringPlanningLists,
  buttonOf,
  creates,
  cyclePreferencesOf,
  made,
  modulePreferencesOf,
  planningPageTab,
  quickAddForm,
  send,
} from "./fake-planning-page";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined, params: {} as Record<string, string> }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => bound.params,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
// jsdom has no viewport geometry or font loading.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));
vi.mock("@nerve/propel/emoji-icon-picker", () => ({ Logo: () => null }));
vi.mock("@/components/issues/create-issue-toast-action-items", () => ({ CreateIssueToastActionItems: () => null }));

let page: Awaited<ReturnType<typeof planningPageTab>>;
beforeEach(() => {
  bound.params = { workspaceSlug: "ws", projectId: "A" };
});
afterEach(async () => page.close());

/** Each page, grouped by the other domain: the page's group of it and what its new work item is sent with. */
const pages = {
  "a cycle's page grouped by module": {
    type: EIssuesStoreType.CYCLE,
    param: { cycleId: C1 },
    preferences: () => cyclePreferencesOf({ layout: "list", group_by: "module_id", order_by: "sort_order" }),
    group: () => moduleOf(page.project, "Alpha"),
    sends: (group: string) => ({ module_ids: [group], cycle_id: C1 }),
  },
  "a module's page grouped by cycle": {
    type: EIssuesStoreType.MODULE,
    param: { moduleId: M1 },
    preferences: () => modulePreferencesOf({ layout: "list", group_by: "cycle_id", order_by: "sort_order" }),
    group: () => cycleOf(page.project, "Sprint", { status: "current" }),
    sends: (group: string) => ({ cycle_id: group, module_ids: [M1] }),
  },
} as const;

it.each(Object.keys(pages) as (keyof typeof pages)[])(
  "sends the group's value and the page's own from the quick add of %s, in one request",
  async (name) => {
    const { type, param, preferences, group, sends } = pages[name];
    bound.params = { ...bound.params, ...param };
    page = await planningPageTab(type, { role: 15 });
    bound.app = page.app;
    const value = group();
    answeringPlanningLists({
      cycles: type === EIssuesStoreType.MODULE ? [rowOf(cycleOf(page.project, "Sprint", { status: "current" }))] : [],
      modules: type === EIssuesStoreType.CYCLE ? [moduleRowOf(moduleOf(page.project, "Alpha"))] : [],
      archivedModules: [],
    });
    const row = listIssue("first", 1000, type === EIssuesStoreType.CYCLE ? { cycle_id: C1 } : { module_ids: [M1] });
    await page.show(preferences(), { total: 1, groups: [{ key: value.id, total: 1, data: [row], next_cursor: null }] });
    await until(() => buttonOf("issue.new") !== undefined, "the group's quick add");
    await act(async () => buttonOf("issue.new")!.click());
    await send('input[name="name"]', "Made", quickAddForm);
    expect(creates()[0]!.body).toEqual({ name: "Made", ...sends(value.id) });
    await act(async () => creates()[0]!.answer(json(201, made("Made", sends(value.id)))));
    expect(creates()).toHaveLength(1);
  }
);
