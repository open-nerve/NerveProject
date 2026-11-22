// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { until } from "@/lib/auth/fake-time";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import type { TPeekIssue } from "@/store/issue/issue-details/root.store";
import type { NotificationSelection } from "./page-state";
import { NotificationPageStateProvider, useNotificationPageState } from "./page-state";
import { NotificationsRoot } from "./root";

// The notifications page's preview (P4 spec 2.6, R12, re-93 I4): nothing selected, it says so; else it peeks at the
// selected notification's work item (IssuePeekOverview embedded, a stand-in here, scoped to the notification, which
// closes by clearing the selection), once the caller's role in its project is known. A project he does not hold (an
// archived one's) is read first (useProjectRead), the preview saying it is loading meanwhile; once nerve refuses that
// read, or gives no role in it, the selection is cleared and a toast says why: never a blank pane. An intake work item
// opens as any other. The page leaving clears the peek (W6). The caller's roles, the project's read and the peek are
// stand-ins'; SWR is real.

const fixture = vi.hoisted(() => ({
  roles: new Set<string>(),
  reads: [] as { id: string; resolve: (project: unknown) => void; reject: (error: unknown) => void }[],
  detail: { peekIssue: undefined as TPeekIssue | undefined, setPeekIssue: (_: TPeekIssue | undefined) => {} },
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/propel/empty-state", () => ({ EmptyStateCompact: () => <p data-empty-state /> }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => ({ id: "id-acme", slug: "acme" }) }));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => ({
    getProjectRoleByWorkspaceSlugAndProjectId: (slug: string, projectId?: string) =>
      slug === "acme" && projectId !== undefined && fixture.roles.has(projectId) ? 15 : undefined,
  }),
}));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    fetchProject: (id: string) => new Promise((resolve, reject) => fixture.reads.push({ id, resolve, reject })),
  }),
}));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => fixture.detail }));
vi.mock("@/components/issues/peek-overview", () => ({
  IssuePeekOverview: (props: {
    hostScope: string;
    embedIssue?: boolean;
    embedRemoveCurrentNotification?: () => void;
  }) => (
    <section aria-label={props.hostScope} data-embed={String(props.embedIssue)}>
      <button type="button" onClick={props.embedRemoveCurrentNotification}>
        close
      </button>
    </section>
  ),
}));
vi.mock("@/components/inbox/content", () => ({ InboxContentRoot: () => <p data-triage /> }));

const held: { state: ReturnType<typeof useNotificationPageState> | undefined } = { state: undefined };
function Read() {
  held.state = useNotificationPageState();
  return null;
}

let host: HTMLDivElement, root: Root, cache: Map<string, unknown>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers();
  toasts.length = 0;
  fixture.roles = new Set(["p-web"]);
  fixture.reads = [];
  const detail = fixture.detail;
  detail.peekIssue = undefined;
  detail.setPeekIssue = (peek) => (detail.peekIssue = peek);
  cache = new Map();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.useRealTimers();
});

const render = () =>
  act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => cache as never }}>
        <NotificationPageStateProvider>
          <Read />
          <NotificationsRoot />
        </NotificationPageStateProvider>
      </SWRConfig>
    )
  );
const select = (selection: NotificationSelection) => act(async () => held.state?.select(selection));
/** The peek's stand-in, inside the preview's region. */
const peek = () => host.querySelector<HTMLElement>("section[data-embed]");
const placeholder = () => host.querySelector('[role="status"]');
const emptyState = () => host.querySelector("[data-empty-state]");
const scopeOf = (selection: NotificationSelection) =>
  JSON.stringify(["notification", "acme", selection.notificationId]);
const peekOf = (selection: NotificationSelection) => ({
  workspaceSlug: "acme",
  projectId: selection.projectId,
  issueId: selection.issueId,
});
/** The read of the project the preview made, once it is out. */
async function projectRead(id: string) {
  await act(async () => until(() => fixture.reads.some((read) => read.id === id), "the project's read"));
  return fixture.reads.find((read) => read.id === id)!;
}
const settled = () => act(async () => vi.advanceTimersByTimeAsync(50));

const one: NotificationSelection = { notificationId: "n-one", issueId: "i-web-1", projectId: "p-web" };
const two: NotificationSelection = { notificationId: "n-two", issueId: "i-web-2", projectId: "p-web" };
const archived: NotificationSelection = { notificationId: "n-old", issueId: "i-old-1", projectId: "p-old" };

describe("NotificationsRoot, the preview", () => {
  it("is a region named the preview, whatever it shows: that nothing is selected, or the selected one's work item", async () => {
    await render();
    const region = () => host.querySelector('section[aria-label="notification.a11y.preview"]');
    expect(region()?.contains(emptyState())).toBe(true);
    await select(one);
    expect(region()?.contains(peek())).toBe(true);
  });

  it("says nothing is selected, and peeks at nothing", async () => {
    await render();
    expect(emptyState()).not.toBeNull();
    expect(peek()).toBeNull();
    expect(fixture.detail.peekIssue).toBeUndefined();
  });

  it("peeks at the selected notification's work item, embedded and scoped to the notification, reading no project", async () => {
    await render();
    await select(one);
    expect(peek()?.getAttribute("aria-label")).toBe(scopeOf(one));
    expect(peek()?.dataset.embed).toBe("true");
    expect(fixture.detail.peekIssue).toEqual(peekOf(one));
    expect(emptyState()).toBeNull();
    await settled();
    expect(fixture.reads).toEqual([]);
  });

  it("peeks at another notification's work item in place of the first", async () => {
    await render();
    await select(one);
    await select(two);
    expect(peek()?.getAttribute("aria-label")).toBe(scopeOf(two));
    expect(fixture.detail.peekIssue).toEqual(peekOf(two));
  });

  it("reads a project the caller holds no role in (an archived one's), saying it is loading meanwhile, then peeks", async () => {
    await render();
    await select(archived);
    const read = await projectRead("p-old");
    expect(placeholder()?.textContent).toContain("common.loading");
    expect(peek()).toBeNull();
    expect(fixture.detail.peekIssue).toBeUndefined();
    fixture.roles.add("p-old");
    read.resolve({ id: "p-old" });
    await settled();
    expect(placeholder()).toBeNull();
    expect(peek()?.getAttribute("aria-label")).toBe(scopeOf(archived));
    expect(fixture.detail.peekIssue).toEqual(peekOf(archived));
  });

  it("clears the selection and says why once nerve refuses the project's read, never a blank pane (re-93 I4)", async () => {
    await render();
    await select(archived);
    (await projectRead("p-old")).reject(refusal(404, "project.not_found"));
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.project_not_found" }]);
    expect(held.state?.selected).toBeUndefined();
    expect(emptyState()).not.toBeNull();
    expect(placeholder()).toBeNull();
  });

  it("judges the project by the read it makes: a refusal an earlier preview had is not this one's (whole-branch m1)", async () => {
    await render();
    await select(archived);
    (await projectRead("p-old")).reject(refusal(503, "service.unavailable"));
    await settled();
    expect(toasts).toHaveLength(1);
    expect(held.state?.selected).toBeUndefined();

    // nerve is reached again: selected once more, the preview waits for its own read, which gives the project
    await select(archived);
    await act(async () => until(() => fixture.reads.length === 2, "the second read of the project"));
    expect(toasts).toHaveLength(1);
    expect(placeholder()?.textContent).toContain("common.loading");
    fixture.roles.add("p-old");
    fixture.reads[1]?.resolve({ id: "p-old" });
    await settled();
    expect(toasts).toHaveLength(1);
    expect(held.state?.selected).toEqual(archived);
    expect(peek()?.getAttribute("aria-label")).toBe(scopeOf(archived));
    // one read of the project for each preview
    expect(fixture.reads).toHaveLength(2);
  });

  it("clears the selection and says why once nerve gives the project but no role in it (one the caller has left)", async () => {
    await render();
    await select(archived);
    (await projectRead("p-old")).resolve({ id: "p-old", member_role: null });
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.issue_not_found" }]);
    expect(held.state?.selected).toBeUndefined();
    expect(emptyState()).not.toBeNull();
  });

  it("opens an intake work item's notification as a plain peek, no triage", async () => {
    await render();
    const intake: NotificationSelection = { notificationId: "n-intake", issueId: "i-web-9", projectId: "p-web" };
    await select(intake);
    expect(peek()?.getAttribute("aria-label")).toBe(scopeOf(intake));
    expect(host.querySelector("[data-triage]")).toBeNull();
  });

  it("clears the selection as the peek closes", async () => {
    await render();
    await select(one);
    await act(async () => peek()?.querySelector("button")?.click());
    expect(held.state?.selected).toBeUndefined();
    expect(emptyState()).not.toBeNull();
  });

  it("clears the peek as the page leaves (W6)", async () => {
    await render();
    await select(one);
    expect(fixture.detail.peekIssue).toEqual(peekOf(one));
    await act(async () => root.render(null));
    expect(fixture.detail.peekIssue).toBeUndefined();
  });
});
