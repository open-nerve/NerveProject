// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { IIssueDisplayFilterOptions, IIssueDisplayProperties, TIssueOrderByOptions } from "@nerve/types";
import { SpreadsheetHeaderColumn } from "../spreadsheet-header-column";
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let view: ReturnType<typeof createRoot>;
let host: HTMLDivElement;
let changes: Partial<IIssueDisplayFilterOptions>[];
beforeEach(() => {
  vi.useFakeTimers();
  localStorage.clear();
  changes = [];
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  host = document.createElement("div");
  document.body.append(host);
  view = createRoot(host);
});
afterEach(async () => {
  await act(async () => view.unmount());
  host.remove();
  vi.unstubAllGlobals();
  vi.useRealTimers();
  localStorage.clear();
});
async function render(property: keyof IIssueDisplayProperties, order: TIssueOrderByOptions) {
  await act(async () =>
    view.render(
      <table>
        <thead>
          <tr>
            <SpreadsheetHeaderColumn
              property={property}
              displayProperties={{ [property]: true }}
              displayFilters={{ order_by: order }}
              handleDisplayFilterUpdate={(change) => changes.push(change)}
            />
          </tr>
        </thead>
      </table>
    )
  );
}
async function choose(index: number) {
  const trigger = host.querySelector("button")!;
  vi.spyOn(trigger, "getBoundingClientRect").mockReturnValue(new DOMRect(1, 1, 180, 40));
  await act(async () => trigger.click());
  const items = [...document.querySelectorAll<HTMLButtonElement>('[role="menuitem"]')];
  expect(items.length).toBeGreaterThan(index);
  await act(async () => items[index]!.click());
}
it.each([
  ["labels", "labels__name", "label_name", "-label_name"],
  ["assignee", "assignees__first_name", "assignee_name", "-assignee_name"],
  ["state", "state__name", "state_name", "-state_name"],
  ["due_date", "target_date", "-target_date", "target_date"],
  ["sub_issue_count", "sub_issues_count", "-sub_issues_count", "sub_issues_count"],
] as const)("%s menu sends the native order in both directions", async (property, held, first, second) => {
  await render(property, held);
  await choose(0);
  expect(changes).toEqual([{ order_by: first }]);
  await choose(1);
  expect(changes).toEqual([{ order_by: first }, { order_by: second }]);
});
it("confirmed aria-sort ignores global sorting keys and changes only with the confirmed preference", async () => {
  localStorage.setItem("spreadsheetViewSorting", JSON.stringify("-assignees__first_name_assignee"));
  localStorage.setItem("spreadsheetViewActiveSortingProperty", JSON.stringify("assignee"));
  const saved = localStorage.getItem("spreadsheetViewSorting");
  await render("labels", "labels__name");
  expect(host.querySelector("th")!.getAttribute("aria-sort")).toBe("ascending");
  await choose(1); // A failed or pending request supplies no new preference.
  expect(host.querySelector("th")!.getAttribute("aria-sort")).toBe("ascending");
  expect(localStorage.getItem("spreadsheetViewSorting")).toBe(saved);
  await render("labels", "-labels__name");
  expect(host.querySelector("th")!.getAttribute("aria-sort")).toBe("descending");
  await render("labels", "-created_at"); // Another project has another confirmed preference.
  expect(host.querySelector("th")!.getAttribute("aria-sort")).toBe("none");
});
