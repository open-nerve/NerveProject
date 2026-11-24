// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { until } from "@/lib/auth/fake-time";
import type { NotificationSelection } from "./page-state";
import { NotificationPageStateProvider, useNotificationPageState } from "./page-state";
import { NotificationsRoot } from "./root";

// The notifications page's preview closes in every state of it (P5b spec 2.5, R26; T11 m1): while it reads the
// project of a work item whose project the caller holds no role in (an archived one's), and while it reads the work
// item to judge how to show it, it says it is loading beside a button named Close, which leaves the notification
// through the page's landing: the selection cleared, the focus on the card's main button; the read answering later
// shows nothing. Below 768 px the list has no width while a notification is selected (sidebar/root.tsx): the close is
// the way back to it. The page's state, its landing, the preview, the read hooks and SWR are real; the caller's roles,
// the project's and the work item's fetches (held until a test answers them), the stores, the peek, the embed and the
// list (its region and cards) are stand-ins.

const fixture = vi.hoisted(() => ({
  /** The projects read, by their ids: each read held, never answered here. */
  projectReads: [] as string[],
  /** The work item's reads, each answered by its resolve. */
  issueReads: [] as (() => void)[],
  /** The work item store's stand-in (useIssueDetail's), one object, as the page's own. */
  detail: {
    setPeekIssue: (_: unknown) => {},
    rootIssueStore: {
      rootStore: { workspaceRoot: { getWorkspaceBySlug: (slug: string) => ({ id: "id-acme", slug }) } },
      permissionReads: { identity: () => "identity-1" },
      issues: { getIssueDetailById: (_: string) => undefined },
    },
  },
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/propel/empty-state", () => ({ EmptyStateCompact: () => <p data-empty-state /> }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => ({ id: "id-acme", slug: "acme" }) }));
// the caller is a member of p-web, and holds no role in p-old (an archived project's)
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => ({
    getProjectRoleByWorkspaceSlugAndProjectId: (_: string, projectId?: string) =>
      projectId === "p-web" ? 15 : undefined,
  }),
}));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    fetchProject: (id: string) => new Promise(() => void fixture.projectReads.push(id)),
  }),
}));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => fixture.detail }));
vi.mock("@/hooks/use-project-labels-read", () => ({ useProjectLabelsRead: () => undefined }));
vi.mock("@/components/issues/issue-detail/use-issue-operations", () => ({
  useIssueOperations: () => ({
    fetch: () => new Promise<void>((resolve) => void fixture.issueReads.push(resolve)),
  }),
}));
vi.mock("@/components/issues/peek-overview", () => ({ IssuePeekOverview: () => <section data-peek /> }));
vi.mock("@/components/inbox/content", () => ({ InboxContentRoot: () => <section data-triage /> }));

/** The list's stand-in: its region and a card for each notification, whose first button selects it. */
const CARDS: NotificationSelection[] = [
  { notificationId: "n-web", issueId: "i-web-1", projectId: "p-web" },
  { notificationId: "n-old", issueId: "i-old-1", projectId: "p-old" },
];
let pageState: ReturnType<typeof useNotificationPageState> | undefined;
function List() {
  const state = useNotificationPageState();
  pageState = state;
  return (
    <div data-notification-list tabIndex={-1} aria-label="list">
      <ul>
        {CARDS.map((card) => (
          <li key={card.notificationId} data-notification-id={card.notificationId}>
            <button type="button" onClick={() => state.select(card)}>
              {card.notificationId}
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}

let host: HTMLDivElement, root: Root;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers();
  fixture.projectReads = [];
  fixture.issueReads = [];
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  await act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <NotificationPageStateProvider>
          <List />
          <NotificationsRoot />
        </NotificationPageStateProvider>
      </SWRConfig>
    )
  );
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.useRealTimers();
});

const settled = () => act(async () => void (await vi.advanceTimersByTimeAsync(50)));
const cardOf = (notificationId: string) =>
  host.querySelector<HTMLElement>(`[data-notification-id="${notificationId}"] button`);
const preview = () => host.querySelector<HTMLElement>('section[aria-label="notification.a11y.preview"]');
/** The preview's buttons named Close (fake-i18n gives the key). */
const closes = () => [...(preview()?.querySelectorAll<HTMLButtonElement>('button[aria-label="close"]') ?? [])];
/** The keyboard's press of the close: it has the focus first, as Tab gives it, then Enter clicks it. */
const pressClose = () =>
  act(async () => {
    closes()[0]!.focus();
    closes()[0]!.click();
  });

describe("the preview's close while it reads (T11 m1)", () => {
  it("is drawn while it reads the work item to judge how to show it: pressed, the notification is left through the page's landing", async () => {
    await act(async () => cardOf("n-web")!.click());
    await act(async () => until(() => fixture.issueReads.length === 1, "the work item's read"));
    expect(preview()?.querySelector('[role="status"]')).toBeTruthy();
    expect(closes()).toHaveLength(1);
    await pressClose();
    await settled();
    expect(pageState?.selected).toBeUndefined();
    expect(document.activeElement).toBe(cardOf("n-web"));
    // its read answering later shows nothing: nothing is selected
    await act(async () => fixture.issueReads[0]!());
    await settled();
    expect(host.querySelector("[data-peek], [data-triage]")).toBeNull();
    expect(preview()?.querySelector("[data-empty-state]")).toBeTruthy();
  });

  it("is drawn while it reads the project of one the caller holds no role in (an archived one's): pressed, the same", async () => {
    await act(async () => cardOf("n-old")!.click());
    await act(async () => until(() => fixture.projectReads.length === 1, "the project's read"));
    expect(fixture.projectReads).toEqual(["p-old"]);
    expect(preview()?.querySelector('[role="status"]')).toBeTruthy();
    expect(closes()).toHaveLength(1);
    await pressClose();
    await settled();
    expect(pageState?.selected).toBeUndefined();
    expect(document.activeElement).toBe(cardOf("n-old"));
  });
});
