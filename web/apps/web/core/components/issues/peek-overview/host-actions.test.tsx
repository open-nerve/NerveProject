// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { heldChange } from "@/lib/auth/fake-tab";
import { IssueDetailHost } from "../detail-host";
import { IssueParentSelect } from "../issue-detail/parent-select";
import { IssueView } from "./view";
import { openIssuePeek } from "./selection";
import type { WorkItemDetailQuickActions } from "../issue-layouts/quick-action-dropdowns";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "host" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", () => ({
  Link: ({ children, to }: { children: ReactNode; to: string }) => <a href={to}>{children}</a>,
}));
vi.mock("@/hooks/store/use-issue-detail", async (original) => ({
  ...(await original<typeof import("@/hooks/store/use-issue-detail")>()),
  useIssueDetail: () => tab.app.issue.issueDetail,
}));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-issues", () => ({ useIssues: () => ({ issues: tab.app.issue.archivedIssues }) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@makeplane/propel/components/tooltip", () => ({
  Tooltip: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("../issue-detail/issue-activity", () => ({ IssueActivity: () => null }));
vi.mock("../issue-detail-widgets", () => ({ IssueDetailWidgets: () => null }));
vi.mock("./issue-detail", () => ({ PeekOverviewIssueDetails: () => null }));
vi.mock("./properties", () => ({ PeekOverviewProperties: () => null }));
vi.mock("../issue-layouts/quick-action-dropdowns", () => ({
  WorkItemDetailQuickActions: (props: ComponentProps<typeof WorkItemDetailQuickActions>) => {
    actions = props;
    return null;
  },
}));
vi.mock("../parent-issues-list-modal", () => ({ ParentIssuesListModal: () => null }));
vi.mock("../issue-detail/issue-identifier", () => ({ IssueIdentifier: () => <span>Parent key</span> }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let actions: ComponentProps<typeof WorkItemDetailQuickActions>;
let pending: ReturnType<typeof heldChange<void>>;
let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
const notificationClose = vi.fn();
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  tab = await issueStoreTab();
  pending = heldChange<void>();
  notificationClose.mockReset();
  for (const id of ["old", "new"]) tab.app.issue.issues.acceptRead(issueDetailOf(id), tab.app.issue.captureRead());
  vi.spyOn(tab.app.issue.issueDetail, "removeIssue").mockImplementation(() => pending.sent);
  vi.spyOn(tab.app.issue.issueDetail, "archiveIssue").mockImplementation(() => pending.sent);
  container = document.createElement("div");
  document.body.append(container);
  root = createRoot(container);
});
afterEach(async () => {
  await act(async () => root.unmount());
  container.remove();
  tab.app.dispose();
  vi.useRealTimers();
});
const render = async (id: string, embed = true) => {
  openIssuePeek(
    tab.app.issue.issueDetail,
    { workspaceSlug: "ws", projectId: "A", issueId: id },
    document.activeElement instanceof HTMLElement ? document.activeElement : undefined
  );
  await act(async () =>
    root.render(
      <IssueDetailHost workspaceSlug="ws" projectId="A" issueId={id}>
        <IssueView
          workspaceSlug="ws"
          projectId="A"
          issueId={id}
          isLoading={false}
          is_archived={false}
          embedIssue={embed}
          embedRemoveCurrentNotification={notificationClose}
          issueOperations={{ fetch: async () => {}, update: async () => {}, restore: () => pending.sent }}
        />
      </IssueDetailHost>
    )
  );
};
// The archive dialog owns confirmation before invoking the page's navigation callback.
function command(name: "handleDelete" | "handleArchive" | "handleRestore") {
  const current = actions;
  return Promise.resolve(current[name]!()).then(() => {
    if (name === "handleArchive") current.onArchiveConfirmed?.();
  });
}
it.each(["handleDelete", "handleArchive", "handleRestore"] as const)(
  "a late %s cannot close the next peek or notification",
  async (name) => {
    await render("old");
    let result!: Promise<unknown>;
    await act(async () => {
      result = command(name);
    });
    await render("new");
    await act(async () => {
      pending.answer();
      await result;
    });
    expect(tab.app.issue.issueDetail.peekIssue?.issueId).toBe("new");
    expect(notificationClose).not.toHaveBeenCalled();
  }
);
it.each(["handleDelete", "handleArchive", "handleRestore"] as const)(
  "a confirmed current %s closes its peek",
  async (name) => {
    await render("old");
    let result!: Promise<unknown>;
    await act(async () => {
      result = command(name);
    });
    await act(async () => {
      pending.answer();
      await result;
    });
    expect(tab.app.issue.issueDetail.peekIssue).toBeUndefined();
  }
);
it("a refused deletion remains rejected for its confirmation dialog", async () => {
  await render("old");
  const failure = new Error("refused");
  const result = Promise.resolve(actions.handleDelete!());
  const rejected = expect(result).rejects.toBe(failure);
  await act(async () => pending.refuse(failure));
  await rejected;
  expect(tab.app.issue.issueDetail.peekIssue?.issueId).toBe("old");
});

it.each(["handleDelete", "handleArchive", "handleRestore"] as const)(
  "a late %s cannot close a reopened selection of the same issue",
  async (name) => {
    await render("old");
    let result!: Promise<unknown>;
    await act(async () => {
      result = command(name);
    });
    await render("old");
    const replacement = tab.app.issue.issueDetail.peekIssue;
    await act(async () => {
      pending.answer();
      await result;
    });
    expect(tab.app.issue.issueDetail.peekIssue).toBe(replacement);
    expect(notificationClose).not.toHaveBeenCalled();
  }
);
it("Escape does not dismiss the peek underneath its local confirmation dialog", async () => {
  await render("old", false);
  await act(async () => actions.toggleDeleteIssueModal!(true));
  await act(async () => document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" })));
  expect(tab.app.issue.issueDetail.peekIssue?.issueId).toBe("old");
});
it("the peek layout trigger has only one native button", async () => {
  await render("old", false);
  expect(container.querySelectorAll("button button")).toHaveLength(0);
});

it("parent controls use separate native buttons beside the parent link", async () => {
  tab.app.issue.issues.acceptRead({ ...issueDetailOf("old"), parent_id: "parent" }, tab.app.issue.captureRead());
  tab.app.issue.issues.acceptRead(issueDetailOf("parent"), tab.app.issue.captureRead());
  const remove = vi.fn();
  await act(async () =>
    root.render(
      <IssueDetailHost workspaceSlug="ws" projectId="A" issueId="old">
        <IssueParentSelect
          workspaceSlug="ws"
          projectId="A"
          issueId="old"
          disabled={false}
          workItemLink="/parent"
          handleParentIssue={async () => {}}
          handleRemoveSubIssue={remove}
        />
      </IssueDetailHost>
    )
  );
  expect(container.querySelectorAll("button a, button button")).toHaveLength(0);
  const removeButton = container.querySelector<HTMLButtonElement>('button[aria-label="common.remove"]');
  expect(removeButton).not.toBeNull();
  await act(async () => removeButton!.click());
  expect(remove).toHaveBeenCalledExactlyOnceWith("ws", "A", "parent", "old");
});

it("Escape belongs to the actual trigger rather than another row with the same global ID", async () => {
  const other = document.createElement("button");
  other.id = "issue-old";
  const trigger = document.createElement("button");
  document.body.append(other, trigger);
  try {
    trigger.focus();
    await render("old", false);
    container.querySelector<HTMLButtonElement>("button")!.focus();
    await act(async () => document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true })));
    expect(document.activeElement).toBe(trigger);
  } finally {
    other.remove();
    trigger.remove();
  }
});
it("Escape already handled by a child does not close the peek", async () => {
  await render("old", false);
  const event = new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true });
  event.preventDefault();
  await act(async () => document.dispatchEvent(event));
  expect(tab.app.issue.issueDetail.peekIssue?.issueId).toBe("old");
});

it.each([false, true])("does not focus an unmounted or hidden trigger (hidden=%s)", async (hidden) => {
  const trigger = document.createElement("button");
  document.body.append(trigger);
  trigger.focus();
  await render("old", false);
  if (hidden) trigger.setAttribute("inert", "");
  else trigger.remove();
  const focus = vi.spyOn(trigger, "focus");
  await act(async () => document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" })));
  expect(focus).not.toHaveBeenCalled();
  expect(tab.app.issue.issueDetail.peekIssue).toBeUndefined();
  trigger.remove();
});
it("a previous selection cannot restore focus after a same-issue replacement", async () => {
  const trigger = document.createElement("button");
  document.body.append(trigger);
  try {
    trigger.focus();
    await render("old", false);
    let result!: Promise<unknown>;
    await act(async () => {
      result = Promise.resolve(actions.handleDelete!());
    });
    await render("old", false);
    const focus = vi.spyOn(trigger, "focus");
    await act(async () => {
      pending.answer();
      await result;
    });
    expect(focus).not.toHaveBeenCalled();
    expect(tab.app.issue.issueDetail.peekIssue?.issueId).toBe("old");
  } finally {
    trigger.remove();
  }
});
