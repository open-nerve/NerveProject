// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { issueOf } from "@/store/issue/fake-issues";
import { SpreadsheetSubIssueColumn } from "./sub-issue-column";

let identifier: string | undefined;
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => ({ getProjectIdentifierById: () => identifier }) }));
let host: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
function Location() {
  const location = useLocation();
  return <output>{location.pathname + location.hash}</output>;
}
beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  identifier = "WEB";
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});
async function render(archived = false, sequence = 42, count = 2) {
  const issue = {
    ...issueOf("parent"),
    sequence_id: sequence,
    sub_issues_count: count,
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
                <SpreadsheetSubIssueColumn issue={issue} />
                <Location />
              </>
            }
          />
        </Routes>
      </MemoryRouter>
    )
  );
}
it.each([false, true])("opens the visible %s archived parent directly, with no fragment", async (archived) => {
  await render(archived);
  const link = host.querySelector("a")!;
  expect(link.textContent).toBe("2 sub-work items");
  expect(link.getAttribute("href")).toBe("/ws/browse/WEB-42");
  await act(async () => link.click());
  expect(host.querySelector("output")?.textContent).toBe("/ws/browse/WEB-42");
});
it("retains the count without a bad link when the directory has not loaded", async () => {
  identifier = undefined;
  await render();
  expect(host.querySelector("a")).toBeNull();
  expect(host.textContent).toContain("2 sub-work items");
});
it.each([0, -1, Number.NaN, Number.MAX_SAFE_INTEGER + 1])(
  "does not generate an invalid sequence link: %s",
  async (sequence) => {
    await render(false, sequence);
    expect(host.querySelector("a")).toBeNull();
  }
);
it("does not offer a link for an empty count", async () => {
  await render(false, 42, 0);
  expect(host.querySelector("a")).toBeNull();
  expect(host.textContent).toContain("0 sub-work items");
});
