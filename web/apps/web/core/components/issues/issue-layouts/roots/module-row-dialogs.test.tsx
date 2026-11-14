// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A row's edit and copy dialogs on a module's page (P6b spec 2.7; P5b T13): the layout's command host opens them as
// the module page's (its store type: a copy then joins the module while it takes work items, the module actions' tests
// in core/hooks), and holds them. The dialog reads its work item with its resources; the read of its sub-issues checks
// the work item itself against nerve (M4 P12 D22), and while that check is out the list takes the row off the page,
// then reads its first page again: the dialog stays open with what it was opened with while the row goes and comes
// back, in each layout. The layouts, rows, menus, the dialog, the stores and HTTP are real; the dialog says the store
// type it was opened with.
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import type { RootStore } from "@/store/root.store";
import type { IssuesModalProps } from "../../issue-modal/modal";
import { M1, layouts, modulePreferencesOf, outlivesTheRow, planningPageTab, type Layout } from "./fake-planning-page";

const bound = vi.hoisted(() => ({
  app: undefined as RootStore | undefined,
  params: {} as Record<string, string>,
  opened: new Set<string | undefined>(),
}));
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
// jsdom has no viewport geometry; rows, cards, days and HTTP are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));
// The row's dialog: the real one, which says the store type it is open with.
vi.mock("@/components/issues/issue-modal/modal", async (original) => {
  const real = await original<typeof import("@/components/issues/issue-modal/modal")>();
  return {
    ...real,
    CreateUpdateIssueModal: (props: IssuesModalProps) => {
      if (props.isOpen) bound.opened.add(props.storeType);
      return <real.CreateUpdateIssueModal {...props} />;
    },
  };
});

let page: Awaited<ReturnType<typeof planningPageTab>>;
beforeEach(() => {
  bound.params = { workspaceSlug: "ws", projectId: "A", moduleId: M1 };
  bound.opened.clear();
});
afterEach(async () => page.close());

/** The module's page in layout, its one work item answered; nerve's list of it. */
async function modulePage(layout: Layout) {
  page = await planningPageTab(EIssuesStoreType.MODULE);
  bound.app = page.app;
  const list = page.listOf(layout);
  await page.show(modulePreferencesOf(layouts[layout].display), list);
  return list;
}

it.each(Object.keys(layouts) as Layout[])(
  "keeps a row's edit dialog, the module page's, open with its work item in a module's %s layout while the row is checked",
  async (layout) => {
    const list = await modulePage(layout);
    await outlivesTheRow("common.actions.edit", "first", list);
    expect([...bound.opened]).toEqual([EIssuesStoreType.MODULE]);
  }
);

it("keeps a row's copy dialog, the module page's, open with the copy on a module's page while the row is checked", async () => {
  const list = await modulePage("list");
  await outlivesTheRow("common.actions.make_a_copy", "first (copy)", list);
  expect([...bound.opened]).toEqual([EIssuesStoreType.MODULE]);
});
