// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ReactNode } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { IssueCycleActivity } from "./cycle";

const activity = vi.hoisted(() => ({ current: undefined as Record<string, unknown> | undefined }));
vi.mock("@/hooks/store/use-issue-detail", () => ({
  useIssueDetail: () => ({ activity: { getActivityById: () => activity.current } }),
}));
vi.mock("./index", () => ({
  IssueActivityBlockComponent: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});

const render = async (row: Record<string, unknown>) => {
  activity.current = { project_id: "p", old_identifier: null, new_identifier: null, ...row };
  // Each row its own id: the observer re-renders on new props.
  const id = String(row.verb);
  await act(async () =>
    root.render(<IssueCycleActivity activityId={id} issueId="i" workspaceSlug="ws" ends={undefined} />)
  );
  return host.querySelector("a");
};

// The server writes a cycle left, or deleted, as `deleted` from its name to "" (M6 design 3.3, 8): the row names
// and links the cycle it left, from old_value and old_identifier.
it("namesTheCycleAWorkItemLeft from its old value", async () => {
  const link = await render({ verb: "deleted", old_value: "Sprint 3", new_value: "", old_identifier: "c3" });
  expect(host.textContent).toContain("removed the work item from the cycle");
  expect(link?.textContent?.trim()).toBe("Sprint 3");
  expect(link?.getAttribute("href")).toBe("/ws/projects/p/cycles/c3");
});

// Joining and moving name the cycle joined, from new_value and new_identifier.
it.each([
  ["created", "added this work item to the cycle"],
  ["updated", "set the cycle to"],
])("namesTheCycleAWorkItemJoinedOrMovedTo from its new value: %s", async (verb, text) => {
  const link = await render({ verb, old_value: "Sprint 1", new_value: "Sprint 2", new_identifier: "c2" });
  expect(host.textContent).toContain(text);
  expect(link?.textContent?.trim()).toBe("Sprint 2");
  expect(link?.getAttribute("href")).toBe("/ws/projects/p/cycles/c2");
});
