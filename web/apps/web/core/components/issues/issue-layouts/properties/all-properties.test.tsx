// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type PropsWithChildren } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { issueOf } from "@/store/issue/fake-issues";
import { IssueProperties } from "./all-properties";

let identifier: string | undefined;
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({ getProjectById: () => (identifier ? { identifier } : undefined) }),
}));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => ({ getStateById: () => undefined }) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("../../issue-detail/use-issue-operations", () => ({ useDetailCommand: () => vi.fn() }));
vi.mock("@makeplane/propel/components/tooltip", () => ({ Tooltip: ({ children }: PropsWithChildren) => children }));
vi.mock("@/components/dropdowns/date", () => ({ DateDropdown: () => null }));
vi.mock("@/components/dropdowns/date-range", () => ({ DateRangeDropdown: () => null }));
vi.mock("@/components/dropdowns/member/dropdown", () => ({ MemberDropdown: () => null }));
vi.mock("@/components/dropdowns/priority", () => ({ PriorityDropdown: () => null }));
vi.mock("@/components/dropdowns/state/dropdown", () => ({ StateDropdown: () => null }));
vi.mock("./labels", () => ({ IssuePropertyLabels: () => null }));
let parent: HTMLDivElement;
let host: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
const rowClick = vi.fn(),
  rowKey = vi.fn();
function Location() {
  const location = useLocation();
  return <output>{location.pathname + location.hash}</output>;
}
beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  identifier = "WEB";
  rowClick.mockClear();
  rowKey.mockClear();
  parent = document.createElement("div");
  host = document.createElement("div");
  parent.append(host);
  document.body.append(parent);
  root = createRoot(host);
  parent.addEventListener("click", rowClick);
  parent.addEventListener("keydown", rowKey);
});
afterEach(async () => {
  await act(async () => root.unmount());
  parent.remove();
});
async function render(archived = false, sequence = 42) {
  const issue = {
    ...issueOf("parent"),
    sequence_id: sequence,
    sub_issues_count: 2,
    archived_at: archived ? "2026-10-31T00:00:00Z" : null,
  };
  await act(async () =>
    root.render(
      <MemoryRouter initialEntries={["/ws/projects/A/issues"]}>
        <Routes>
          <Route
            path="/:workspaceSlug/*"
            element={
              <>
                <IssueProperties
                  issue={issue}
                  updateIssue={undefined}
                  displayProperties={{ sub_issue_count: true }}
                  isReadOnly
                  className=""
                  activeLayout="list"
                />
                <Location />
              </>
            }
          />
        </Routes>
      </MemoryRouter>
    )
  );
}
it.each([false, true])(
  "clicks the visible %s archived count and reaches browse without a fragment or row action",
  async (archived) => {
    await render(archived);
    const count = host.querySelector<HTMLElement>(".text-caption-sm-regular")!;
    expect(count.textContent).toBe("2");
    await act(async () => count.click());
    expect(host.querySelector("output")?.textContent).toBe("/ws/browse/WEB-42");
    expect(rowClick).not.toHaveBeenCalled();
  }
);
it("provides a named keyboard link without delegating Enter to the row", async () => {
  await render();
  const link = host.querySelector("a")!;
  expect(link.getAttribute("aria-label")).toBe("common.sub_work_items: 2");
  link.focus();
  expect(document.activeElement).toBe(link);
  await act(async () => link.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true })));
  expect(rowKey).not.toHaveBeenCalled();
});
it("shows a plain count while the project directory is unavailable", async () => {
  identifier = undefined;
  await render();
  expect(host.querySelector("a")).toBeNull();
  expect(host.querySelector(".text-caption-sm-regular")?.textContent).toBe("2");
  await act(async () => host.querySelector<HTMLElement>(".text-caption-sm-regular")!.click());
  expect(host.querySelector("output")?.textContent).toBe("/ws/projects/A/issues");
});
it.each([0, -1, Number.NaN, Number.MAX_SAFE_INTEGER + 1])(
  "does not generate a URL for invalid sequence %s",
  async (sequence) => {
    await render(false, sequence);
    expect(host.querySelector("a")).toBeNull();
  }
);
