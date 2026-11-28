// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The workspace's filters' options (M7/P6b spec 2.4, 2.9): on a workspace's view's page the state filter offers the
// states of each project of the workspace the caller is a member of, unarchived, as the state store gives them (the
// workspace's states the shell reads, each project's own once read); a state of another project is none of them. The
// page, the stores, SWR and HTTP are real (fake-workspace-view-saving.tsx); the app's header a frame.
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { COLLECTION_OPERATOR, FILTER_FIELD_TYPE } from "@nerve/types";
import { answered } from "@/lib/auth/fake-nerve";
import { projectOf, stateOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import { workspaceSavingTab } from "./fake-workspace-view-saving";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));
vi.mock("@/components/issues/peek-overview", () => ({ IssuePeekOverview: () => null }));
vi.mock("@/components/core/app-header", () => ({
  AppHeader: (props: { header: ReactNode }) => <header>{props.header}</header>,
}));

let page: Awaited<ReturnType<typeof workspaceSavingTab>>;
beforeEach(() => {
  window.localStorage.clear();
});
afterEach(async () => page.close());

it("offers the states of the caller's projects of the workspace as the state filter's options", async () => {
  page = await workspaceSavingTab();
  bound.app = page.app;
  const a = page.project;
  const b = page.app.projectRoot.project.getProjectById("B")!;
  const other = projectOf("Z", page.workspace.id, { id: "Z" });
  // the workspace's states, as the shell reads them (the tab's own answer of a project's states set aside)
  const respond = page.nerve.respond;
  page.nerve.respond = undefined;
  await answered(
    page.nerve,
    () => page.app.state.fetchWorkspaceStates({ id: page.workspace.id, slug: "ws" }),
    ["GET", "/api/v0/workspaces/ws/states"],
    {
      data: [
        stateOf(a, "Todo", "unstarted", 1, { id: "s-a" }),
        stateOf(b, "Doing", "started", 1, { id: "s-b" }),
        stateOf(other, "Done", "completed", 1, { id: "s-z" }),
      ],
    },
    "the workspace's states"
  );
  page.nerve.respond = respond;
  await page.open("all-issues");
  const config = page
    .filter("all-issues")
    .configManager.getConfigByProperty("state_id")
    ?.getOperatorConfig(COLLECTION_OPERATOR.IN);
  expect(config?.type).toBe(FILTER_FIELD_TYPE.MULTI_SELECT);
  if (config?.type !== FILTER_FIELD_TYPE.MULTI_SELECT) throw new Error("No state filter");
  const options = typeof config.getOptions === "function" ? await config.getOptions() : config.getOptions;
  expect(options.map((option) => option.value).toSorted()).toEqual(["s-a", "s-b"]);
});
