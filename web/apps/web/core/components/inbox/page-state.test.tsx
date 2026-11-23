// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter, Route, Routes, useNavigate } from "react-router";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import type { IntakeFilters } from "@/store/issue/intake/filters";
import { defaultFilters } from "@/store/issue/intake/filters";
import { IntakePageStateProvider, useIntakePageState } from "./page-state";

// The intake page's state (P5a spec 2.6, R5): the filters and the order of the tab the address names (?currentTab, an
// unknown one the open tab), which the intake's layout holds, never a store, so that they end with the page; keyed by
// the tab, so that a change of tab, by the tabs or a Back, returns to that tab's own first filters. The state is read
// as the page renders it.

const held: {
  state: ReturnType<typeof useIntakePageState> | undefined;
  navigate: ReturnType<typeof useNavigate> | undefined;
} = { state: undefined, navigate: undefined };
function Read() {
  held.state = useIntakePageState();
  held.navigate = useNavigate();
  return null;
}

let host: HTMLDivElement, root: Root;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  held.state = undefined;
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});

const address = (search: string) => `/acme/projects/p-web/intake${search}`;
/** The intake page opens at the address, its state under the layout's provider. */
const mount = (search = "?currentTab=open") =>
  act(async () =>
    root.render(
      <MemoryRouter initialEntries={[address(search)]}>
        <Routes>
          <Route
            path="/:workspaceSlug/projects/:projectId/intake"
            element={
              <IntakePageStateProvider>
                <Read />
              </IntakePageStateProvider>
            }
          />
        </Routes>
      </MemoryRouter>
    )
  );
/** The page leaves: nothing of it is mounted. */
const leave = () => act(async () => root.render(null));
/** The page goes to the address's search, as the tabs or a link go there. */
const go = (search: string) => act(async () => void held.navigate!(address(search)));
/** The page's filters as it changes them, then as it renders with them. */
async function change(does: (filters: IntakeFilters) => IntakeFilters) {
  await act(async () => held.state!.setFilters(does));
  return held.state!.filters;
}

describe("the intake page's state", () => {
  it("opens on the tab the address names, with its first filters; an unknown tab or none is the open tab", async () => {
    await mount("?currentTab=closed");
    expect(held.state?.filters).toEqual(defaultFilters("closed"));
    await leave();
    await mount("?currentTab=archived&inboxIssueId=i-x");
    expect(held.state?.filters).toEqual(defaultFilters("open"));
    await leave();
    await mount("");
    expect(held.state?.filters).toEqual(defaultFilters("open"));
  });

  it("changes the filters of the tab shown", async () => {
    await mount();
    const filters = await change((current) => ({ ...current, priorities: ["urgent"], order: "created_at" }));
    expect(filters).toEqual({ ...defaultFilters("open"), priorities: ["urgent"], order: "created_at" });
  });

  it("returns to a tab's first filters as the tab changes, and again as it comes back (keyed by the tab)", async () => {
    await mount();
    await change((current) => ({ ...current, statuses: ["pending", "snoozed"], priorities: ["urgent"] }));
    await go("?currentTab=closed");
    expect(held.state?.filters).toEqual(defaultFilters("closed"));
    await go("?currentTab=open");
    expect(held.state?.filters).toEqual(defaultFilters("open"));
  });

  it("keeps the filters as the selection changes within the tab", async () => {
    await mount();
    await change((current) => ({ ...current, labels: ["l-bug"] }));
    await go("?currentTab=open&inboxIssueId=i-x");
    expect(held.state?.filters).toEqual({ ...defaultFilters("open"), labels: ["l-bug"] });
  });

  it("ends with the page: once it leaves and comes back, the tab's first filters", async () => {
    await mount();
    await change((current) => ({ ...current, priorities: ["high"] }));
    await leave();
    await mount();
    expect(held.state?.filters).toEqual(defaultFilters("open"));
  });
});
