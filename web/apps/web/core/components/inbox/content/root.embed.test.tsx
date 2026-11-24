// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, createRef, type ComponentProps } from "react";
import { MemoryRouter, useLocation } from "react-router";
import { SWRConfig } from "swr";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { IssueIntake } from "@nerve/api-client";
import type { EmptyStateCompact, EmptyStateDetailed } from "@nerve/propel/empty-state";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { refusal } from "@/lib/fake-refusal";
import { StoreContext } from "@/lib/store-context";
import type { RootStore } from "@/store/root.store";
import { detailOf } from "@/store/issue/intake/fake-intake";
import type { CreateUpdateIssueModal } from "@/components/issues/issue-modal/modal";
import {
  answer,
  buttonSaying,
  callTo,
  chooseFromMenu,
  detailRegion,
  dialogNamed,
  page,
  settled,
  shown,
  useIntakePageFixture,
} from "../fake-intake-page";
import { InboxContentRoot } from "./root";
import type { IntakeSelectionPort } from "./selection-port";

// The detail region of a work item of a project's intake embedded in the notifications page's preview (P5b spec 2.5,
// R17, R30): its port the embed's, no intake page around it. Its headers draw no navigation (no previous, next or list
// toggle) but a button named Close, its port's close; a triage nerve confirmed leaves the selection where it is, the
// focus on the new header's primary action, else on the embed's region; a deletion leaves it (the port's leave). Open
// is offered on every status: an item not accepted opens in the intake page, selected in the tab that lists it, one
// accepted in the browse page; copy copies the browse page's address still. Before its detail shows, its read loading
// or failed, it draws its Close too (T11 m1); the intake page's selection none. The stores, the headers, the action
// model and the modals are real (the accept modal's form a stand-in), nerve a fake; the session is
// fake-intake-page.tsx's.

const held = vi.hoisted(() => ({ copy: vi.fn(), where: "", fails: [] as ((error: unknown) => void)[] }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", async () => ({
  rootStore: {},
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/propel/empty-state", () => ({
  EmptyStateCompact: (props: ComponentProps<typeof EmptyStateCompact>) => <p>{props.title}</p>,
  EmptyStateDetailed: (props: ComponentProps<typeof EmptyStateDetailed>) => <p>{props.title}</p>,
}));
vi.mock("@nerve/utils", async (original) => ({
  ...(await original<typeof import("@nerve/utils")>()),
  copyUrlToClipboard: held.copy,
}));
vi.mock("@/hooks/use-profile-image", () => import("@/lib/fake-profile-image"));
vi.mock("@/hooks/store/user/user-user-profile", () => ({
  useUserProfile: () => ({ data: { start_of_the_week: 1 } }),
}));
vi.mock("@/components/issues/issue-detail/use-issue-operations", async (original) => ({
  ...(await original<typeof import("@/components/issues/issue-detail/use-issue-operations")>()),
  useIssueOperations: () => ({
    fetch: (_ws: string, _project: string, id: string) =>
      new Promise<void>((resolve, reject) => {
        held.fails.push(reject);
        page.reads.push({
          id,
          answer: (detail) => {
            page.tab.app.issue.issues.acceptRead(detail, page.tab.app.issue.captureRead());
            resolve();
          },
        });
      }),
  }),
}));
vi.mock("@/components/inbox/content/detail", () => ({ IntakeDetail: () => <p data-body /> }));
// the accept modal's form: its one button accepts the work item, then closes the modal (accept.test.tsx is its own)
vi.mock("@/components/issues/issue-modal/modal", () => ({
  CreateUpdateIssueModal: (props: ComponentProps<typeof CreateUpdateIssueModal>) => (
    <div role="dialog" aria-label="accept">
      <button
        type="button"
        onClick={async () => {
          await props.beforeFormSubmit?.();
          props.onClose();
        }}
      >
        confirm accept
      </button>
    </div>
  ),
}));

useIntakePageFixture();

const TRIAGE = "/api/v0/intake-issues/i-x/triage";
const ISSUE = "/api/v0/issues/i-x";
const intakeOf = (status: IssueIntake["status"], snoozedTill: string | null = null): IssueIntake => ({
  status,
  snoozed_till: snoozedTill,
  duplicate_to_id: null,
});
// 25 November 2026, 15:10 in the browser's zone
const NOW = new Date(2026, 10, 25, 15, 10);
const BROWSE = "/acme/browse/WEB-7";

/** The embed's port: its leave and its close recorded, its region the detail region's element. */
const portOf = () =>
  ({
    kind: "embed",
    leave: vi.fn<IntakeSelectionPort["leave"]>(),
    region: createRef<HTMLElement>(),
    close: vi.fn<() => void>(),
  }) satisfies IntakeSelectionPort;
let port: ReturnType<typeof portOf>;
beforeEach(() => {
  held.copy.mockReset();
  held.copy.mockResolvedValue(undefined);
  held.fails = [];
  port = portOf();
});

function Where() {
  const location = useLocation();
  held.where = `${location.pathname}${location.search}`;
  return null;
}

/** Draws the detail region of i-x of p-web (its project WEB) in the notifications page, given selection. */
const drawn = (selection: IntakeSelectionPort = port) =>
  act(async () =>
    page.root.render(
      <StoreContext.Provider value={page.tab.app}>
        <SWRConfig value={{ provider: () => new Map() }}>
          <MemoryRouter initialEntries={["/acme/notifications"]}>
            <Where />
            <InboxContentRoot workspaceSlug="acme" projectId="p-web" inboxIssueId="i-x" selection={selection} />
          </MemoryRouter>
        </SWRConfig>
      </StoreContext.Provider>
    )
  );
/** Embeds i-x in the notifications page, its read answering intake. */
async function embedded(intake: IssueIntake = intakeOf("pending")) {
  await drawn();
  await shown(detailOf("x", { intake, sequence_id: 7 }));
}

/** nerve triages i-x as intake says, then its read again in the triage's turn answers the same. */
async function confirm(intake: IssueIntake) {
  await answer(callTo("POST", TRIAGE), json(200, detailOf("x", { intake, sequence_id: 7 })));
  await answer(callTo("GET", ISSUE), json(200, detailOf("x", { intake, sequence_id: 7 })));
  await settled();
}
const named = (label: string) => [...detailRegion()!.querySelectorAll<HTMLElement>(`[aria-label="${label}"]`)];
/** The wide header's link that opens the work item. */
const openLink = () =>
  [...detailRegion()!.querySelectorAll<HTMLAnchorElement>("a")].find(
    (link) => link.textContent?.trim() === "inbox_issue.actions.open"
  );

describe("the intake's detail embedded in the notifications page (R17)", () => {
  it("draws no navigation, but a button named Close in each header, which closes the embed", async () => {
    await embedded();
    expect(named("inbox_issue.a11y.previous")).toEqual([]);
    expect(named("inbox_issue.a11y.next")).toEqual([]);
    expect(named("inbox_issue.a11y.list_toggle")).toEqual([]);
    const closes = [...detailRegion()!.querySelectorAll<HTMLButtonElement>('button[aria-label="close"]')];
    expect(closes.map((button) => !!button.closest(".lg\\:hidden"))).toEqual([false, true]);
    await act(async () => closes[0]!.click());
    expect(port.close).toHaveBeenCalledOnce();
    expect(port.leave).not.toHaveBeenCalled();
  });

  it("stays on a work item an admin declined, the focus on the embed's region (no primary action)", async () => {
    await embedded();
    await act(async () => buttonSaying("inbox_issue.actions.decline", detailRegion()!)!.click());
    await act(async () => buttonSaying("decline", dialogNamed("inbox_issue.modals.decline.title"))!.click());
    await confirm(intakeOf("declined"));
    expect(port.leave).not.toHaveBeenCalled();
    expect(port.close).not.toHaveBeenCalled();
    expect(dialogNamed("inbox_issue.modals.decline.title")).toBeUndefined();
    expect(document.activeElement).toBe(detailRegion());
    expect(port.region.current).toBe(detailRegion());
  });

  it("stays on a work item an admin snoozed, the focus on the new header's primary action (accepting)", async () => {
    vi.setSystemTime(NOW);
    await embedded();
    await chooseFromMenu("wide", "inbox_issue.actions.snooze");
    await act(async () =>
      buttonSaying("inbox_issue.actions.snooze", dialogNamed("inbox_issue.modals.snooze.title"))!.click()
    );
    await confirm(intakeOf("snoozed", new Date(2026, 10, 26).toISOString()));
    expect(port.leave).not.toHaveBeenCalled();
    expect(document.activeElement).toBe(buttonSaying("inbox_issue.actions.accept", detailRegion()!));
  });

  it("stays on a work item an admin accepted, once the accept modal closes, the focus on the embed's region", async () => {
    await embedded();
    await act(async () => buttonSaying("inbox_issue.actions.accept", detailRegion()!)!.click());
    await act(async () => buttonSaying("confirm accept", document.querySelector('[aria-label="accept"]')!)!.click());
    await confirm(intakeOf("accepted"));
    expect(document.querySelector('[aria-label="accept"]')).toBeNull();
    expect(port.leave).not.toHaveBeenCalled();
    expect(document.activeElement).toBe(detailRegion());
    // accepted, it is read-only, and it opens in the browse page
    expect(buttonSaying("inbox_issue.actions.accept", detailRegion()!)).toBeUndefined();
    expect(openLink()?.getAttribute("href")).toBe(BROWSE);
  });

  it("leaves a work item the caller deleted: the port's leave, for none", async () => {
    await embedded();
    await chooseFromMenu("wide", "inbox_issue.actions.delete");
    await act(async () => buttonSaying("delete", dialogNamed("inbox_issue.modals.delete.title"))!.click());
    await answer(callTo("DELETE", "/api/v0/intake-issues/i-x"), noContent());
    await answer(callTo("GET", ISSUE), problem(404, "issue.not_found"));
    await settled();
    expect(port.leave).toHaveBeenCalledExactlyOnceWith("i-x", undefined);
  });
});

describe("opening and copying from the embed (R30)", () => {
  it.each<[string, IssueIntake, string]>([
    ["pending", intakeOf("pending"), "/acme/projects/p-web/intake?currentTab=open&inboxIssueId=i-x"],
    [
      "snoozed",
      intakeOf("snoozed", "2026-12-30T00:00:00.000Z"),
      "/acme/projects/p-web/intake?currentTab=open&inboxIssueId=i-x",
    ],
    ["declined", intakeOf("declined"), "/acme/projects/p-web/intake?currentTab=closed&inboxIssueId=i-x"],
    ["a duplicate", intakeOf("duplicate"), "/acme/projects/p-web/intake?currentTab=closed&inboxIssueId=i-x"],
    ["accepted", intakeOf("accepted"), BROWSE],
  ])(
    "opens a work item %s at its address: the intake page's, selecting it, until accepted",
    async (_, intake, address) => {
      await embedded(intake);
      expect(openLink()?.getAttribute("href")).toBe(address);
      await act(async () => openLink()!.click());
      expect(held.where).toBe(address);
    }
  );

  it("opens a pending one from the narrow header's menu at the intake page's address too", async () => {
    await embedded();
    await chooseFromMenu("narrow", "inbox_issue.actions.open");
    expect(held.where).toBe("/acme/projects/p-web/intake?currentTab=open&inboxIssueId=i-x");
  });

  it("copies the browse page's address of a pending one, not the intake page's", async () => {
    await embedded();
    await act(async () => buttonSaying("inbox_issue.actions.copy", detailRegion()!)!.click());
    await settled();
    expect(held.copy).toHaveBeenCalledExactlyOnceWith(BROWSE);
  });
});

describe("the embed's close before its detail shows (T11 m1)", () => {
  /** The region's buttons named Close. */
  const closes = () => [...detailRegion()!.querySelectorAll<HTMLButtonElement>('button[aria-label="close"]')];
  /** The region's read, once it is out, fails as nerve is busy (no refusal of the work item: kept, with its retry). */
  const failed = async () => {
    await act(async () => until(() => held.fails.length === 1, "the region's read"));
    await act(async () => held.fails[0]!(refusal(503, "service.unavailable")));
    await settled();
    expect(detailRegion()!.querySelector('[role="alert"]')).toBeTruthy();
  };

  it("is drawn above the loading while the embed's read is out: pressed, the port's close", async () => {
    await drawn();
    await act(async () => until(() => page.reads.length === 1, "the embed's read"));
    expect(detailRegion()!.querySelector('[role="status"]')?.textContent).toBe("common.loading");
    expect(closes()).toHaveLength(1);
    await act(async () => closes()[0]!.click());
    expect(port.close).toHaveBeenCalledOnce();
    expect(port.leave).not.toHaveBeenCalled();
  });

  it("is drawn above the read's failure and its retry: pressed, the port's close", async () => {
    await drawn();
    await failed();
    expect(buttonSaying("common.retry", detailRegion()!)).toBeTruthy();
    expect(closes()).toHaveLength(1);
    await act(async () => closes()[0]!.click());
    expect(port.close).toHaveBeenCalledOnce();
    expect(port.leave).not.toHaveBeenCalled();
  });

  it("is not drawn on the intake page (its selection's kind page), its read out or failed: its list is beside it", async () => {
    const onPage = {
      kind: "page",
      leave: vi.fn<IntakeSelectionPort["leave"]>(),
      neighbour: () => undefined,
      region: createRef<HTMLElement>(),
      navigation: () => null,
    } satisfies IntakeSelectionPort;
    await drawn(onPage);
    await act(async () => until(() => page.reads.length === 1, "the page's read"));
    expect(detailRegion()!.querySelector('[role="status"]')).toBeTruthy();
    expect(closes()).toEqual([]);
    await failed();
    expect(closes()).toEqual([]);
    expect(onPage.leave).not.toHaveBeenCalled();
  });
});
