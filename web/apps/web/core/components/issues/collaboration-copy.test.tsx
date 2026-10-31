// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { heldChange } from "@/lib/auth/fake-tab";
import { toasts } from "@/lib/fake-toast";
import { CreateIssueToastActionItems } from "./create-issue-toast-action-items";
import { useWorkItemCommentOperations } from "./issue-detail/issue-activity/helper";
import { useProjectIssueMenuItems } from "./issue-layouts/quick-action-dropdowns/helper";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "copy" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: () => undefined }) }));
vi.mock("@/hooks/store/user", () => ({ useUser: () => tab.app.user }));
vi.mock("@/hooks/store/use-editor-asset", () => ({
  useEditorAsset: () => ({ uploadEditorAsset: vi.fn(), duplicateEditorAsset: vi.fn() }),
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>, root: Root, host: HTMLDivElement;
const clipboardDescriptor = Object.getOwnPropertyDescriptor(navigator, "clipboard");
let clipboard: ReturnType<typeof heldChange<void>>;
let write: ReturnType<typeof vi.fn<(text: string) => Promise<void>>>;
let escaped: unknown[];
const issue = issueDetailOf("x");
type ConsumerKind = "creation" | "comment" | "layout";
function Consumer({ kind }: { kind: ConsumerKind }) {
  const comments = useWorkItemCommentOperations("ws", "A", "x");
  const menu = useProjectIssueMenuItems({
    issue,
    workspaceSlug: "ws",
    projectIdentifier: "A",
    isEditingAllowed: true,
    isDeletingAllowed: true,
    setIssueToEdit: () => {},
    setCreateUpdateIssueModal: () => {},
    setDeleteIssueModal: () => {},
  });
  if (kind === "creation") return <CreateIssueToastActionItems workspaceSlug="ws" issueId="x" />;
  return (
    <button
      onClick={() => {
        const result =
          kind === "comment"
            ? comments.copyCommentLink("discussion")
            : menu.find((row) => row.key === "copy-link")!.action();
        void Promise.resolve(result).catch((error) => escaped.push(error));
      }}
    >
      Copy
    </button>
  );
}
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  tab = await issueStoreTab();
  tab.app.issue.issues.acceptRead(issue, tab.app.issue.captureRead());
  tab.nerve.calls.length = 0;
  clipboard = heldChange<void>();
  write = vi.fn(() => clipboard.sent);
  Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText: write } });
  toasts.length = 0;
  escaped = [];
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  clipboard.answer();
  await act(async () => root.unmount());
  host.remove();
  tab.app.dispose();
  if (clipboardDescriptor) Object.defineProperty(navigator, "clipboard", clipboardDescriptor);
  else Reflect.deleteProperty(navigator, "clipboard");
  vi.restoreAllMocks();
  vi.useRealTimers();
});
for (const kind of ["creation", "comment", "layout"] as const) {
  it.each(["success", "refusal"])(`copyWaitsForClipboardAndReportsOneOutcome: ${kind} %s`, async (outcome) => {
    await act(async () => root.render(<Consumer kind={kind} />));
    await act(async () => host.querySelector("button")!.click());
    expect(write).toHaveBeenCalledExactlyOnceWith(
      `${window.location.origin}/ws/browse/A-1${kind === "comment" ? "#comment-discussion" : ""}`
    );
    expect(toasts).toEqual([]);
    expect(host.textContent).not.toContain("Copied!");
    await act(async () => {
      if (outcome === "success") clipboard.answer();
      else clipboard.refuse(new DOMException("Denied", "NotAllowedError"));
      await vi.advanceTimersByTimeAsync(0);
    });
    expect(escaped).toEqual([]);
    expect(toasts).toHaveLength(1);
    expect(toasts[0]).toMatchObject(
      outcome === "success"
        ? {
            type: "success",
            title: "common.link_copied",
            message: kind === "comment" ? "issue.comments.copy_link.success" : "common.copied_to_clipboard",
          }
        : { type: "error", message: "something_went_wrong_please_try_again" }
    );
    expect(host.textContent).not.toContain("Copied!");
    expect(tab.nerve.calls).toEqual([]);
  });
}
