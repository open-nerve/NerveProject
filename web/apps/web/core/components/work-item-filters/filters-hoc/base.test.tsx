// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { observer } from "mobx-react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { WorkItemFilterStore, type IWorkItemFilterInstance } from "@nerve/shared-state";
import { EIssuesStoreType } from "@nerve/types";
import { WorkItemFiltersHOC } from "./base";
const config = { areAllConfigsInitialized: true, configs: [] };
vi.mock("@/hooks/store/work-item-filters/use-work-item-filters", () => ({ useWorkItemFilters: () => store }));
vi.mock("@/hooks/work-item-filters/use-work-item-filters-config", () => ({ useWorkItemFiltersConfig: () => config }));
let store: WorkItemFilterStore;
let host: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
const changed = vi.fn();
const shown: (string | undefined)[] = [];
const Probe = observer(function Probe({ filter }: { filter: IWorkItemFilterInstance }) {
  return <span>{filter.saveViewOptions?.label}</span>;
});
beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  store = new WorkItemFilterStore();
  changed.mockReset();
  shown.length = 0;
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
});
const render = (id: string, label: string, strict = false) => {
  const view = (
    <WorkItemFiltersHOC
      entityId={id}
      entityType={EIssuesStoreType.PROJECT}
      workspaceSlug="ws"
      filtersToShowByLayout={[]}
      initialWorkItemFilters={{
        richFilters: {},
        displayFilters: undefined,
        displayProperties: undefined,
        kanbanFilters: undefined,
      }}
      updateFilters={changed}
      saveViewOptions={{ label, isDisabled: true, onViewSave: () => {} }}
    >
      {({ filter }) => {
        shown.push(filter?.saveViewOptions?.label);
        return filter && <Probe filter={filter} />;
      }}
    </WorkItemFiltersHOC>
  );
  return act(async () => root.render(strict ? <StrictMode>{view}</StrictMode> : view));
};
it("updates a live filter's options after commit without notifying another component during render", async () => {
  const errors = vi.spyOn(console, "error");
  await render("A", "First");
  const filter = store.getFilter(EIssuesStoreType.PROJECT, "A");
  await render("A", "Next");
  expect(host.textContent).toBe("Next");
  expect(store.getFilter(EIssuesStoreType.PROJECT, "A")).toBe(filter);
  expect(changed).not.toHaveBeenCalled();
  expect(errors.mock.calls).toEqual([]);
});
it("switches and releases only the mounted entity, including StrictMode setup/cleanup", async () => {
  await render("A", "First", true);
  expect(store.getFilter(EIssuesStoreType.PROJECT, "A")).toBeDefined();
  await render("B", "Next", true);
  expect(store.getFilter(EIssuesStoreType.PROJECT, "A")).toBeUndefined();
  expect(store.getFilter(EIssuesStoreType.PROJECT, "B")).toBeDefined();
  expect(host.textContent).toBe("Next");
  await act(async () => root.render(null));
  expect(store.getFilter(EIssuesStoreType.PROJECT, "B")).toBeUndefined();
});

it("does not render a prior session's filter when its store is replaced for the same entity", async () => {
  await render("A", "First");
  shown.length = 0;
  store = new WorkItemFilterStore();
  await render("A", "Next");
  expect(host.textContent).toBe("Next");
  expect(shown).not.toContain("First");
});
