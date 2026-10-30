// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useState, type ComponentProps } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { issueDetailOf } from "@/store/issue/fake-issues";
import { heldChange, type HeldChange } from "@/lib/auth/fake-tab";
import { ApiError } from "@/lib/api-error";
import type { RichTextEditor } from "@/components/editor/rich-text";
import { IssueMainContent } from "./issue-detail/main-content";
import type { TIssueOperations } from "./issue-detail/root";

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/hooks/store/use-editor-asset", () => ({ useEditorAsset: () => ({}) }));
vi.mock("@/hooks/store/use-workspace", () => ({
  useWorkspace: () => ({ getWorkspaceBySlug: () => ({ id: "id-ws" }) }),
}));
vi.mock("@/hooks/store/user", () => ({ useUser: () => ({ data: undefined }) }));
vi.mock("@/hooks/use-window-size", () => ({ default: () => [1024, 800] }));
vi.mock("@/hooks/store/use-issue-detail", () => ({
  useIssueDetail: () => ({ issue: { getIssueDetailById: () => issueDetailOf("x") } }),
}));
vi.mock("./issue-detail/issue-identifier", () => ({ IssueIdentifier: () => null }));
vi.mock("./issue-detail/description-versions", () => ({ IssueDescriptionVersions: () => null }));
vi.mock("./issue-detail/issue-activity", () => ({ IssueActivity: () => null }));
vi.mock("./issue-detail/parent", () => ({ IssueParentDetail: () => null }));
vi.mock("./issue-detail/reactions", () => ({ IssueReaction: () => null }));
vi.mock("./issue-detail-widgets", () => ({ IssueDetailWidgets: () => null }));
vi.mock("./peek-overview/properties", () => ({ PeekOverviewProperties: () => null }));
vi.mock("@/components/editor/rich-text", () => ({
  RichTextEditor: (props: ComponentProps<typeof RichTextEditor>) => {
    const [value, setValue] = useState(props.initialValue);
    return (
      <textarea
        aria-label="description"
        value={value}
        onChange={(event) => {
          setValue(event.target.value);
          props.onChange?.({}, event.target.value);
        }}
      />
    );
  },
}));
let host: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
let requests: HeldChange<void>[];
const update = vi.fn<TIssueOperations["update"]>();
beforeEach(() => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  requests = [];
  update.mockReset().mockImplementation(() => {
    const next = heldChange<void>();
    requests.push(next);
    return next.sent;
  });
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.useRealTimers();
});
const edit = async (index: number, value: string) =>
  act(async () => {
    const textarea = host.querySelectorAll("textarea")[index]!;
    Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value")!.set!.call(textarea, value);
    textarea.dispatchEvent(new Event("input", { bubbles: true }));
  });
it("the actual detail content retains a failed description status when the title subsequently succeeds", async () => {
  await act(async () =>
    root.render(
      <IssueMainContent
        workspaceSlug="ws"
        projectId="A"
        issueId="x"
        isEditable
        isArchived={false}
        issueOperations={{ fetch: async () => {}, update }}
      />
    )
  );
  await edit(1, "unsaved description");
  await act(async () => vi.advanceTimersByTimeAsync(1500));
  await edit(0, "saved title");
  await act(async () => vi.advanceTimersByTimeAsync(1500));
  expect(update).toHaveBeenCalledTimes(2);
  await act(async () => requests[0]!.refuse(new ApiError(403, { status: 403, code: "forbidden", title: "Forbidden" })));
  await act(async () => requests[1]!.answer());
  await act(async () => vi.advanceTimersByTimeAsync(2000));
  expect(host.textContent).toContain("save_failed");
  expect([...host.querySelectorAll("span")].some((span) => span.textContent === "saved")).toBe(false);
  expect(host.querySelectorAll("textarea")[1]!.value).toBe("unsaved description");
});
