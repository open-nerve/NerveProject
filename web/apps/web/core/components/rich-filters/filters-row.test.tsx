// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The filters row's own words are the page's language's (P6a spec 2.7, R21): a modal's row says "Filters" on its Add
// filter while it holds no condition; a page's row says Clear all, Save view, Update view, and Confirming while the
// update is out (fake-i18n: a text is its key; en's words are unchanged). The row and the filter are real, over a
// session's stores; what Save view and Update view do is the filter's options (the test's).
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { IWorkItemFilterInstance } from "@nerve/shared-state";
import { EIssuesStoreType } from "@nerve/types";
import { issueStoreTab } from "@/store/issue/fake-issues";
import { FiltersRow } from "./filters-row";

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
const binding = vi.hoisted(() => ({ app: undefined as unknown }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return binding.app;
  },
  StoreContext: (await import("react")).createContext(undefined),
}));

let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
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
  tab = await issueStoreTab();
  binding.app = tab.app;
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  tab.app.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

/** A filter of a view's page, its configs ready, holding conditions or none; Update view waits for the test. */
function filterOf(id: string, conditions: boolean) {
  let release: (() => void) | undefined;
  const filter: IWorkItemFilterInstance = tab.app.workItemFilters.getOrCreateFilter({
    entityType: EIssuesStoreType.PROJECT_VIEW,
    entityId: id,
    initialExpression: conditions ? { and: [{ priority__in: "high" }] } : undefined,
    onExpressionChange: () => {},
    expressionOptions: {
      saveViewOptions: { onViewSave: () => {} },
      updateViewOptions: {
        hasAdditionalChanges: true,
        onViewUpdate: () => new Promise<void>((resolve) => (release = resolve)),
      },
    },
  });
  filter.configManager.setAreConfigsReady(true);
  filter.toggleVisibility(true);
  return { filter, release: () => release?.() };
}
const buttons = () => [...host.querySelectorAll("button")].map((button) => button.textContent?.trim());
const buttonOf = (text: string) => [...host.querySelectorAll("button")].find((b) => b.textContent?.trim() === text);

it("says Filters on a modal's Add filter while its row holds no condition", async () => {
  const { filter } = filterOf("v-none", false);
  await act(async () => root.render(<FiltersRow filter={filter} variant="modal" />));
  expect(buttons()).toContain("common.filters");
});

it("says Clear all, Save view and Update view in the page's language, and Confirming while the update is out", async () => {
  const { filter, release } = filterOf("v-some", true);
  await act(async () => root.render(<FiltersRow filter={filter} />));
  await act(async () => vi.advanceTimersByTimeAsync(500));
  expect(buttons()).toEqual(
    expect.arrayContaining(["common.clear_all", "common.filter_options.save_view", "common.filter_options.update_view"])
  );
  await act(async () => buttonOf("common.filter_options.update_view")!.click());
  expect(buttonOf("common.filter_options.confirming")?.disabled).toBe(true);
  await act(async () => release());
  await act(async () => vi.advanceTimersByTimeAsync(500));
  // the update done, the display still the page's change (hasAdditionalChanges): Update view again
  expect(buttonOf("common.filter_options.update_view")?.disabled).toBe(false);
});
