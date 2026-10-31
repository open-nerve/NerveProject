// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { E_SORT_ORDER } from "@nerve/constants";
import { IssueActivity } from "./root";

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/components/issues/issue-detail/issue-activity", () => import("./activity-filter"));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => ({ getProjectById: () => ({ id: "p" }) }) }));
vi.mock("./helper", () => ({ useWorkItemCommentOperations: () => ({}) }));
vi.mock("@/components/comments/comment-create", () => ({ CommentCreate: () => null }));
vi.mock("./activity-comment-root", () => ({ IssueActivityCommentRoot: () => null }));
let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  localStorage.clear();
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  localStorage.clear();
  vi.restoreAllMocks();
  vi.useRealTimers();
});
const render = () => act(async () => root.render(<IssueActivity workspaceSlug="ws" projectId="p" issueId="i" />));
const tick = () => act(async () => vi.advanceTimersByTimeAsync(200));
const trigger = () => host.querySelector<HTMLButtonElement>("button[aria-expanded]")!;
const option = (key: string) =>
  [...host.querySelectorAll<HTMLButtonElement>("button[aria-pressed]")].find((item) =>
    item.textContent?.includes(`common.${key}`)
  )!;
async function press(button: HTMLButtonElement, key: string) {
  await act(async () => {
    button.focus();
    const down = new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true });
    const up = new KeyboardEvent("keyup", { key, bubbles: true, cancelable: true });
    button.dispatchEvent(down);
    button.dispatchEvent(up);
    // jsdom has no browser activation default; only native buttons receive that default here.
    if (!down.defaultPrevented && !up.defaultPrevented && ["Enter", " "].includes(key)) button.click();
  });
  await tick();
}
it("filtersKeepOneChoiceAndReturnFocus with named native controls and persisted sorting", async () => {
  await render();
  expect(host.querySelector("button button")).toBeNull();
  expect(trigger().textContent).toContain("common.filters");
  await press(trigger(), "Enter");
  expect(host.querySelectorAll('button[aria-pressed="true"]')).toHaveLength(4);
  await press(option("updates"), " ");
  expect(option("updates").getAttribute("aria-pressed")).toBe("false");
  await press(option("state"), "Enter");
  await press(option("assignee"), " ");
  expect(option("comments").getAttribute("aria-disabled")).toBe("true");
  await press(option("comments"), "Enter");
  expect(option("comments").getAttribute("aria-pressed")).toBe("true");
  expect(JSON.parse(localStorage.getItem("issue_activity_filters")!)).toEqual(["COMMENT"]);
  await press(option("comments"), "Escape");
  expect(trigger().getAttribute("aria-expanded")).toBe("false");
  expect(document.activeElement).toBe(trigger());
  const sort = host.querySelector<HTMLButtonElement>('button[aria-label="aria_labels.toggle_sort_order"]')!;
  await press(sort, " ");
  expect(JSON.parse(localStorage.getItem("activity_sort_order")!)).toBe(E_SORT_ORDER.DESC);
  await act(async () => root.render(null));
  await render();
  await press(trigger(), " ");
  expect(host.querySelectorAll('button[aria-pressed="true"]')).toHaveLength(1);
  expect(option("comments").getAttribute("aria-pressed")).toBe("true");
  await press(option("updates"), "Enter");
  expect(option("comments").getAttribute("aria-disabled")).toBe("false");
  await press(option("updates"), "Escape");
  expect(document.activeElement).toBe(trigger());
  await press(host.querySelector<HTMLButtonElement>('button[aria-label="aria_labels.toggle_sort_order"]')!, "Enter");
  expect(JSON.parse(localStorage.getItem("activity_sort_order")!)).toBe(E_SORT_ORDER.ASC);
});
