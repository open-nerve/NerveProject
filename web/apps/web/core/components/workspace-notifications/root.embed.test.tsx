// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { createRoot, type Root } from "react-dom/client";
import { observable, runInAction } from "mobx";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { IssueDetail, IssueIntake } from "@nerve/api-client";
import { until } from "@/lib/auth/fake-time";
import { refusal } from "@/lib/fake-refusal";
import type { InboxContentRoot } from "@/components/inbox/content";
import type { IntakeSelectionPort } from "@/components/inbox/content/selection-port";
import { useIssueOperations } from "@/components/issues/issue-detail/use-issue-operations";
import { usePeekIssueRead } from "@/components/issues/peek-overview/use-peek-issue-read";
import { issueOf } from "@/store/issue/fake-issues";
import type { TPeekIssue } from "@/store/issue/issue-details/root.store";
import type { NotificationSelection } from "./page-state";
import { NotificationPageStateProvider, useNotificationPageState } from "./page-state";
import { NotificationsRoot } from "./root";

// The notifications page's preview of a work item (P5b spec 2.5, R16, R17, R26, R27; re-95 I3, I4, m10, m11, O2, O7):
// it reads the work item by a read of its own (usePeekIssueRead, its mount and reconnect reads off), loading meanwhile,
// and judges once that read settles, by the work item as held: one in the intake (pending, snoozed, declined, a
// duplicate) embeds the intake's detail (the embed's port), never setting the peek; any other is peeked at, the peek
// joining the preview's read. The judgement is latched: neither a read again of its key (a reconnect, the peek's retry)
// nor the work item's change undoes it; a new generation (permissions, login) reads and judges anew. A failed read is
// the preview's own failure, with a retry; its close, the embed's and the peek's go through the page's landing. The
// page's state, the preview, the read hook and SWR are real; the stores, the fetch (held until a test answers it), the
// peek (reading through the real hook), the embed and the list (its region and cards) are stand-ins.

const fixture = vi.hoisted(() => ({
  reads: [] as ReturnType<typeof heldRead>[],
  /** The work item store's stand-in (useIssueDetail's), set before each test. */
  detail: undefined as unknown as ReturnType<typeof storeOf>,
  /** The selection InboxContentRoot was last given. */
  port: undefined as IntakeSelectionPort | undefined,
  /** The caller's login and his permissions' identity, which a test changes (observable, set before each test). */
  session: undefined as unknown as ReturnType<typeof sessionOf>,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/propel/empty-state", () => ({ EmptyStateCompact: () => <p data-empty-state /> }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({
  useSession: () => ({ status: "signed-in", loginId: fixture.session.loginId }),
}));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => ({ id: "id-acme", slug: "acme" }) }));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => ({ getProjectRoleByWorkspaceSlugAndProjectId: () => 15 }),
}));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => ({ fetchProject: () => new Promise(() => {}) }) }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => fixture.detail }));
vi.mock("@/components/issues/issue-detail/use-issue-operations", () => ({
  useIssueOperations: () => ({
    fetch: (_ws: string, _project: string, id: string) =>
      new Promise<void>((resolve, reject) => fixture.reads.push(heldRead(id, resolve, reject))),
  }),
}));
// the peek: it reads its work item through the real hook, with SWR's defaults; a comment's input, its retry, its close
vi.mock("@/components/issues/peek-overview", async () => ({
  IssuePeekOverview: (await import("mobx-react")).observer(function IssuePeekOverview(props: PeekProps) {
    const peek = fixture.detail.peekIssue;
    return peek ? <PeekContent peek={peek} {...props} /> : null;
  }),
}));
vi.mock("@/components/inbox/content", () => ({
  InboxContentRoot: (props: ComponentProps<typeof InboxContentRoot>) => {
    fixture.port = props.selection;
    const { selection } = props;
    return (
      <section ref={selection.region} tabIndex={-1} data-triage={props.inboxIssueId}>
        <input aria-label="embed input" />
        <button type="button" onClick={() => selection.leave(props.inboxIssueId, undefined)}>
          embed leave
        </button>
        <button type="button" onClick={() => selection.kind === "embed" && selection.close()}>
          embed close
        </button>
      </section>
    );
  },
}));

type PeekProps = { hostScope: string; embedIssue?: boolean; embedRemoveCurrentNotification?: () => void };
function PeekContent({ peek, ...props }: PeekProps & { peek: TPeekIssue }) {
  const { fetch } = useIssueOperations(peek.workspaceSlug, peek.projectId, peek.issueId);
  const read = usePeekIssueRead(peek, fetch);
  return (
    <section aria-label={props.hostScope} data-peek={peek.issueId}>
      <input aria-label="peek input" />
      <button type="button" onClick={() => void read.mutate()}>
        peek retry
      </button>
      <button type="button" onClick={props.embedRemoveCurrentNotification}>
        peek close
      </button>
    </section>
  );
}

/** The work items the stores hold, by id, as the fetch's answers write them. */
const canonical = observable.map<string, IssueDetail>();
/** The peeks the preview set, in order; the page's state, as the list last read it. */
let peeks: (TPeekIssue | undefined)[];
let pageState: ReturnType<typeof useNotificationPageState> | undefined;
/** A read of the work item id, held until a test answers it (its intake row, written into the stores) or fails it. */
const heldRead = (id: string, resolve: () => void, fail: (error: unknown) => void) => ({
  id,
  answer: (intake: IssueIntake | null) => {
    runInAction(() => canonical.set(id, { ...issueOf(id, "p-web", "id-acme"), intake } as IssueDetail));
    resolve();
  },
  fail,
});
const sessionOf = () => observable({ loginId: "one", identity: "permissions-1" });

function storeOf() {
  const peek = observable.box<TPeekIssue | undefined>(undefined, { deep: false });
  const workspaceRoot = { getWorkspaceBySlug: (slug: string) => ({ id: "id-acme", slug }) };
  return {
    get peekIssue() {
      return peek.get();
    },
    setPeekIssue: (next: TPeekIssue | undefined) => {
      peeks.push(next);
      runInAction(() => peek.set(next));
    },
    rootIssueStore: {
      active: true,
      rootStore: { workspaceRoot, user: { permission: { getProjectRoleByWorkspaceSlugAndProjectId: () => 15 } } },
      permissionReads: { identity: () => fixture.session.identity },
      issues: { getIssueDetailById: (id: string) => canonical.get(id) },
    },
  };
}

/** The list's stand-in: its region and a card for each notification, whose first button selects it. */
const CARDS: NotificationSelection[] = [
  { notificationId: "n-pending", issueId: "i-x", projectId: "p-web" },
  { notificationId: "n-again", issueId: "i-x", projectId: "p-web" },
  { notificationId: "n-plain", issueId: "i-y", projectId: "p-web" },
];
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

let host: HTMLDivElement, root: Root, cache: Map<string, unknown>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers();
  fixture.reads = [];
  fixture.port = undefined;
  fixture.session = sessionOf();
  fixture.detail = storeOf();
  peeks = [];
  runInAction(() => canonical.clear());
  cache = new Map();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  await act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => cache as never }}>
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
/** Presses the card of the notification: the page selects it. */
async function open(notificationId: string) {
  await act(async () => cardOf(notificationId)!.click());
  await settled();
}
const cardOf = (notificationId: string) =>
  host.querySelector<HTMLElement>(`[data-notification-id="${notificationId}"] button`);
/** The n-th read of the work item id, once it is out. */
async function readOf(id: string, n = 1) {
  await act(async () => until(() => fixture.reads.filter((read) => read.id === id).length >= n, `read ${n} of ${id}`));
  return fixture.reads.filter((read) => read.id === id)[n - 1]!;
}
/** The n-th read of the work item id answers its intake row, and the page follows. */
async function answered(id: string, intake: IssueIntake | null, n = 1) {
  (await readOf(id, n)).answer(intake);
  await settled();
}
const intakeOf = (status: IssueIntake["status"]): IssueIntake => ({
  status,
  snoozed_till: status === "snoozed" ? "2099-01-01T00:00:00.000Z" : null,
  duplicate_to_id: null,
});
const preview = () => host.querySelector<HTMLElement>('section[aria-label="notification.a11y.preview"]');
const peek = () => host.querySelector<HTMLElement>("section[data-peek]");
const embed = () => host.querySelector<HTMLElement>("section[data-triage]");
const loading = () => preview()?.querySelector('[role="status"]');
const alert = () => preview()?.querySelector('[role="alert"]') ?? null;
const press = (text: string) =>
  act(async () => [...host.querySelectorAll("button")].find((button) => button.textContent?.trim() === text)!.click());
/** Types text into the input named name, as a user's unsent comment. */
const typeInto = (name: string, text: string) => {
  const input = host.querySelector<HTMLInputElement>(`input[aria-label="${name}"]`)!;
  input.value = text;
  return input;
};
const peekOf = (issueId: string) => ({ workspaceSlug: "acme", projectId: "p-web", issueId });

describe("the preview's judgement, by its own read of the work item (R16)", () => {
  it("says it is loading while its read is out: neither a peek nor an embed", async () => {
    await open("n-pending");
    await readOf("i-x");
    expect(loading()).toBeTruthy();
    expect([peek(), embed(), peeks]).toEqual([null, null, []]);
  });

  it.each<[string, IssueIntake]>([
    ["pending", intakeOf("pending")],
    ["snoozed", intakeOf("snoozed")],
    ["declined", intakeOf("declined")],
    ["a duplicate", intakeOf("duplicate")],
  ])("embeds the intake's detail of a work item %s, by the embed's port, never setting the peek", async (_, intake) => {
    await open("n-pending");
    await answered("i-x", intake);
    expect(embed()?.dataset.triage).toBe("i-x");
    expect(preview()?.contains(embed())).toBe(true);
    const port = fixture.port!;
    expect(port.kind).toBe("embed");
    expect(port.region.current).toBe(embed());
    expect([peek(), loading(), peeks]).toEqual([null, null, []]);
  });

  it.each<[string, IssueIntake | null]>([
    ["never submitted to the intake", null],
    ["accepted", intakeOf("accepted")],
  ])("peeks at a work item %s, as before", async (_, intake) => {
    await open("n-plain");
    await answered("i-y", intake);
    expect(peek()?.getAttribute("aria-label")).toBe(JSON.stringify(["notification", "acme", "n-plain"]));
    expect(peeks).toEqual([peekOf("i-y")]);
    expect(embed()).toBeNull();
  });

  it("judges by this read, never by an answer SWR keeps: declined before, accepted now, it peeks", async () => {
    await open("n-pending");
    await answered("i-x", intakeOf("declined"));
    expect(embed()).not.toBeNull();
    await press("embed close");
    // another notification of the same work item: the stores hold it declined, this read is out
    await open("n-again");
    await readOf("i-x", 2);
    expect([loading() !== null, embed(), peek()]).toEqual([true, null, null]);
    await answered("i-x", intakeOf("accepted"), 2);
    expect(peek()?.dataset.peek).toBe("i-x");
    expect(embed()).toBeNull();
  });

  it("judges once: the work item accepted in the embed, it stays embedded, the same element", async () => {
    await open("n-pending");
    await answered("i-x", intakeOf("pending"));
    const embedded = embed();
    runInAction(() => canonical.set("i-x", { ...canonical.get("i-x")!, intake: intakeOf("accepted") }));
    await settled();
    expect(embed()).toBe(embedded);
    expect([peek(), peeks]).toEqual([null, []]);
  });
});

describe("one read of a work item not in the intake (R16; re-95 O2)", () => {
  it("is the preview's: the peek mounting after it joins it", async () => {
    await open("n-plain");
    await answered("i-y", null);
    expect(peek()).not.toBeNull();
    await settled();
    expect(fixture.reads.map((read) => read.id)).toEqual(["i-y"]);
  });
});

describe("the judgement latched (R27; re-95 I4)", () => {
  it.each<[string, string, IssueIntake | null, () => HTMLElement | null]>([
    ["embed", "n-pending", intakeOf("pending"), embed],
    ["peek", "n-plain", null, peek],
  ])(
    "past the dedupe window, the browser reconnecting: nothing reads again, the %s and its input stay",
    async (name, note, intake, shown) => {
      await open(note);
      await answered(note === "n-plain" ? "i-y" : "i-x", intake);
      const element = shown();
      expect(element).not.toBeNull();
      const input = typeInto(`${name} input`, "unsent");
      await act(async () => void (await vi.advanceTimersByTimeAsync(2500)));
      await act(async () => void window.dispatchEvent(new Event("online")));
      // SWR revalidates a timer's turn later, and tells a key's first hook alone (swr@2.4.2): the preview's
      await settled();
      expect(fixture.reads).toHaveLength(1);
      expect(shown()).toBe(element);
      expect(input.isConnected && input.value).toBe("unsent");
    }
  );

  it("the peek's retry, by its hook's mutate: the peek and its input stay", async () => {
    await open("n-plain");
    await answered("i-y", null);
    const peeked = peek();
    const input = typeInto("peek input", "unsent");
    await press("peek retry");
    await readOf("i-y", 2);
    expect([peek(), loading()]).toEqual([peeked, null]);
    await answered("i-y", null, 2);
    expect(peek()).toBe(peeked);
    expect(input.isConnected && input.value).toBe("unsent");
  });
});

describe("the judgement's generation (R27; re-95 m10)", () => {
  it.each<[string, () => void]>([
    ["permissions' identity", () => (fixture.session.identity = "permissions-2")],
    ["login", () => (fixture.session.loginId = "two")],
  ])("is judged anew by a new read once the caller's %s changes", async (_, change) => {
    await open("n-pending");
    await answered("i-x", intakeOf("pending"));
    expect(embed()).not.toBeNull();
    await act(async () => runInAction(change));
    await readOf("i-x", 2);
    await answered("i-x", intakeOf("accepted"), 2);
    expect(peek()?.dataset.peek).toBe("i-x");
    expect(embed()).toBeNull();
  });
});

describe("a read that fails (O-1 乙)", () => {
  it("shows the preview's own failure, the peek never mounted; its retry reads again and judges once settled", async () => {
    await open("n-plain");
    (await readOf("i-y")).fail(refusal(503, "service.unavailable"));
    await settled();
    expect(preview()?.contains(alert())).toBe(true);
    expect([peek(), embed(), peeks]).toEqual([null, null, []]);
    await press("common.retry");
    await answered("i-y", null, 2);
    expect(alert()).toBeNull();
    expect(peek()?.dataset.peek).toBe("i-y");
  });

  it("closes through the page's landing: the selection cleared, the focus on the card's main button", async () => {
    await open("n-plain");
    (await readOf("i-y")).fail(refusal(503, "service.unavailable"));
    await settled();
    const close = preview()?.querySelector<HTMLElement>('[aria-label="common.close_peek_view"]');
    expect(close).toBeTruthy();
    await act(async () => close!.click());
    await settled();
    expect(pageState?.selected).toBeUndefined();
    expect(document.activeElement).toBe(cardOf("n-plain"));
  });
});

describe("closing the preview, through the page's landing (R26; re-95 O7)", () => {
  it.each<[string, string, IssueIntake | null]>([
    ["peek close", "n-plain", null],
    ["embed close", "n-pending", intakeOf("pending")],
    ["embed leave", "n-pending", intakeOf("pending")],
  ])("by the %s: the selection cleared, the focus on the card's main button", async (button, note, intake) => {
    await open(note);
    await answered(note === "n-plain" ? "i-y" : "i-x", intake);
    expect(peek() ?? embed()).not.toBeNull();
    await press(button);
    await settled();
    expect(pageState?.selected).toBeUndefined();
    expect(document.activeElement).toBe(cardOf(note));
  });
});
