// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { issueDetailOf, issueLinkOf, issueStoreTab } from "@/store/issue/fake-issues";
import { json } from "@/lib/auth/fake-nerve";
import { heldChange } from "@/lib/auth/fake-tab";
import { toasts } from "@/lib/fake-toast";
import { IssueDetailHost } from "./detail-host";
import { IdentifierText } from "./issue-detail/identifier-text";
import { IssueDetailQuickActions } from "./issue-detail/issue-detail-quick-actions";
import { IssueLinkItem } from "./issue-detail/links/link-item";
import { RelationIssueListItem } from "./relations/issue-list-item";
import { SubIssuesListItem } from "./issue-detail-widgets/sub-issues/issues-list/list-item";
import { useSubIssueOperations } from "./issue-detail-widgets/sub-issues/helper";
import { IssuePeekOverviewHeader } from "./peek-overview/header";

const binding = vi.hoisted(() => ({
  tab: undefined as Awaited<ReturnType<typeof issueStoreTab>> | undefined,
  peek: vi.fn(),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "copy" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
vi.mock("@/hooks/store/use-issue-detail", async (original) => ({
  ...(await original<typeof import("@/hooks/store/use-issue-detail")>()),
  useIssueDetail: () => binding.tab!.app.issue.issueDetail,
}));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => binding.tab!.app.projectRoot.project }));
vi.mock("@/hooks/store/use-issues", () => ({ useIssues: () => ({ issues: binding.tab!.app.issue.archivedIssues }) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/use-issue-peek-overview-redirection", () => ({
  default: () => ({ handleRedirection: binding.peek }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@makeplane/propel/components/tooltip", () => ({
  Tooltip: ({ children }: { children: ReactNode }) => children,
}));
// Copy controls and their hooks remain real. Unrelated fields and destructive actions are outside this contract.
vi.mock("./issue-layouts/quick-action-dropdowns", () => ({ WorkItemDetailQuickActions: () => null }));
vi.mock("./relations/properties", () => ({ RelationIssueProperty: () => null }));
vi.mock("./issue-detail-widgets/sub-issues/issues-list/properties", () => ({
  SubIssuesListItemProperties: () => null,
}));
vi.mock("./issue-detail-widgets/sub-issues/issues-list/root", () => ({ SubIssuesListRoot: () => null }));

const clipboardDescriptor = Object.getOwnPropertyDescriptor(navigator, "clipboard");
let host: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
let pending: ReturnType<typeof heldChange<void>>;
let write: ReturnType<typeof vi.fn<(text: string) => Promise<void>>>;
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
  pending = heldChange<void>();
  write = vi.fn(() => pending.sent);
  Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText: write } });
  binding.peek.mockReset();
  toasts.length = 0;
  binding.tab = await issueStoreTab();
  const { app, nerve } = binding.tab;
  app.issue.issues.acceptRead({ ...issueDetailOf("source"), sequence_id: 7 }, app.issue.captureRead());
  app.issue.issues.acceptRead({ ...issueDetailOf("target"), sequence_id: 8 }, app.issue.captureRead());
  nerve.respond = (call) => {
    if (call.path === "/api/v0/issues/source/links")
      call.answer(
        json(200, { data: [{ ...issueLinkOf("guide", "source"), title: "Guide", url: "https://example.test/guide" }] })
      );
  };
  await app.issue.issueDetail.link.fetchLinks("ws", "A", "source");
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  pending.answer();
  await act(async () => root.unmount());
  host.remove();
  binding.tab!.app.dispose();
  if (clipboardDescriptor) Object.defineProperty(navigator, "clipboard", clipboardDescriptor);
  else Reflect.deleteProperty(navigator, "clipboard");
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
function SubItem() {
  const operations = useSubIssueOperations("ws", "A", "source");
  return (
    <SubIssuesListItem
      workspaceSlug="ws"
      projectId="A"
      parentIssueId="source"
      rootIssueId="source"
      issueId="target"
      spacingLeft={0}
      canEdit
      handleIssueCrudState={() => {}}
      subIssueOperations={operations}
    />
  );
}
const consumers = ["identifier", "detail", "link", "relation", "sub-item", "peek", "archived-peek"] as const;
type Consumer = (typeof consumers)[number];
function consumer(kind: Consumer): ReactNode {
  if (kind === "identifier") return <IdentifierText identifier="A-7" enableClickToCopyIdentifier />;
  if (kind === "detail") return <IssueDetailQuickActions workspaceSlug="ws" projectId="A" issueId="source" />;
  if (kind === "link")
    return (
      <IssueLinkItem issueId="source" linkId="guide" isNotAllowed={false} linkOperations={{ remove: async () => {} }} />
    );
  if (kind === "relation")
    return (
      <RelationIssueListItem
        workspaceSlug="ws"
        issueId="source"
        relationKey="blocking"
        relationIssueId="target"
        disabled={false}
        handleIssueCrudState={() => {}}
      />
    );
  if (kind === "sub-item") return <SubItem />;
  return (
    <IssuePeekOverviewHeader
      workspaceSlug="ws"
      projectId="A"
      issueId="source"
      peekMode="side-peek"
      setPeekMode={() => {}}
      removeRoutePeekId={() => {}}
      disabled={false}
      embedIssue
      toggleDeleteIssueModal={() => {}}
      toggleArchiveIssueModal={() => {}}
      toggleEditIssueModal={() => {}}
      handleRestoreIssue={async () => {}}
      isSubmitting="saved"
    />
  );
}
async function clickCopy(kind: Consumer) {
  if (kind === "relation" || kind === "sub-item") {
    const trigger = host.querySelector<HTMLButtonElement>('button[aria-haspopup="menu"]')!;
    expect(trigger).not.toBeNull();
    await act(async () => {
      trigger.focus();
      trigger.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }));
      await vi.advanceTimersByTimeAsync(100);
    });
    const option = [...document.querySelectorAll<HTMLButtonElement>('[role="menuitem"]')].find((item) =>
      item.textContent?.includes("copy_link")
    )!;
    expect(option).toBeDefined();
    await act(async () => option.click());
    return;
  }
  const button =
    kind === "identifier"
      ? host.querySelector<HTMLButtonElement>("button")!
      : host.querySelector<HTMLButtonElement>('button[aria-label*="copy_link"]')!;
  expect(button).not.toBeNull();
  await act(async () => button.click());
}
for (const kind of consumers) {
  it.each([true, false])(
    `${kind}'s real copy waits for the clipboard and reports its actual settlement once (accepted=%s)`,
    async (accepted) => {
      if (kind === "archived-peek") {
        const app = binding.tab!.app;
        app.issue.issues.acceptRead(
          { ...issueDetailOf("source"), sequence_id: 7, archived_at: "2026-10-31T00:00:00Z" },
          app.issue.captureRead()
        );
      }
      await act(async () =>
        root.render(
          <MemoryRouter>
            <IssueDetailHost workspaceSlug="ws" projectId="A" issueId="source">
              {consumer(kind)}
            </IssueDetailHost>
          </MemoryRouter>
        )
      );
      await clickCopy(kind);
      const copied =
        kind === "identifier"
          ? "A-7"
          : kind === "link"
            ? "https://example.test/guide"
            : `${window.location.origin}/ws/browse/A-${kind === "relation" || kind === "sub-item" ? 8 : 7}`;
      expect.soft(write.mock.calls).toEqual([[copied]]);
      expect.soft(toasts).toHaveLength(0);
      await act(async () => {
        if (accepted) pending.answer();
        else pending.refuse(new DOMException("Permission denied", "NotAllowedError"));
      });
      expect(toasts).toHaveLength(1);
      if (accepted) expect(toasts[0]?.type).toBe("success");
      else
        expect(toasts[0]).toEqual({
          type: "error",
          title: "toast.error",
          message: "something_went_wrong_please_try_again",
        });
      expect(binding.peek).not.toHaveBeenCalled();
    }
  );
}
