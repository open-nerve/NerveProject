// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps, type ReactNode } from "react";
import { observer } from "mobx-react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { IssueIntake } from "@nerve/api-client";
import type { EmptyStateCompact, EmptyStateDetailed } from "@nerve/propel/empty-state";
import { json } from "@/lib/auth/fake-nerve";
import { StoreContext } from "@/lib/store-context";
import type { RootStore } from "@/store/root.store";
import { detailOf, intakeIssueOf, loadList, open, pageOf } from "@/store/issue/intake/fake-intake";
import {
  answer,
  buttonSaying,
  callTo,
  chooseFromMenu,
  detailRegion,
  dialogNamed,
  opened,
  page,
  selected,
  settled,
  useIntakePageFixture,
} from "../../fake-intake-page";
import { IntakePageStateProvider } from "../../page-state";
import type { IntakeSelectionPort } from "../selection-port";
import type { IntakeActions } from "./intake-actions";
import { IntakeMobileHeader } from "./mobile";
import { IntakeHeader } from "./root";
import { useIntakeActions } from "./use-intake-actions";

// The selection's port (P5b spec 2.5, R18): the detail region of a work item of a project's intake, its headers and
// its action model read the selection through what shows it gives them, never the intake page's state: the action
// model needs no page around it, leaves a triaged work item for the row the port gives, and the headers draw the
// navigation the port gives; the intake page's port gives the page's own, as before. The stores, the action model and
// the headers are real, nerve a fake; the page's session is the fixture's (fake-intake-page.tsx).

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
vi.mock("@/hooks/use-profile-image", () => import("@/lib/fake-profile-image"));
vi.mock("@/components/issues/issue-detail/use-issue-operations", async (original) => ({
  ...(await original<typeof import("@/components/issues/issue-detail/use-issue-operations")>()),
  useIssueOperations: () => ({
    fetch: (_ws: string, _project: string, id: string) =>
      new Promise<void>((resolve) =>
        page.reads.push({
          id,
          answer: (detail) => {
            page.tab.app.issue.issues.acceptRead(detail, page.tab.app.issue.captureRead());
            resolve();
          },
        })
      ),
  }),
}));
vi.mock("@/components/inbox/content/detail", () => ({ IntakeDetail: () => <p data-body /> }));
vi.mock("@/components/inbox/filters/menu", () => ({ IntakeFiltersMenu: () => null }));
vi.mock("@/components/inbox/filters/order-by", () => ({ IntakeOrderBy: () => null }));
vi.mock("@/components/inbox/filters/applied", () => ({ IntakeAppliedFilters: () => null }));

useIntakePageFixture();

const TRIAGE = "/api/v0/intake-issues/i-x/triage";
const ISSUE = "/api/v0/issues/i-x";
const intakeOf = (status: IssueIntake["status"]): IssueIntake => ({
  status,
  snoozed_till: null,
  duplicate_to_id: null,
});
const x = intakeIssueOf("x");
const y = intakeIssueOf("y");

/** The action model, as last drawn. */
let model: IntakeActions | undefined;
beforeEach(() => {
  model = undefined;
});
// whether the detail is deleting the work item: the content root's own, not the port's
const leaving = { current: false };

/** A port of the test's: its moves recorded, the row it gives i-z, the page's list giving i-y; navigation as given. */
const portOf = (navigation: IntakeSelectionPort["navigation"] = () => null) => ({
  leave: vi.fn<IntakeSelectionPort["leave"]>(),
  neighbour: vi.fn<IntakeSelectionPort["neighbour"]>(() => "i-z"),
  region: { current: null },
  navigation,
});

/** The action model of i-x in p-web, given port; the headers, wide and narrow, given it too. */
const Detail = observer(function Detail({ port, headers }: { port: IntakeSelectionPort; headers: boolean }) {
  const actions = useIntakeActions("acme", "p-web", "i-x", { ...port, leaving });
  model = actions;
  if (!headers) return null;
  return (
    <>
      <IntakeHeader projectId="p-web" issueId="i-x" actions={actions} isSubmitting="saved" selection={port} />
      <IntakeMobileHeader issueId="i-x" actions={actions} isSubmitting="saved" selection={port} />
    </>
  );
});

/** Draws tree at the intake page's address, in the session's stores, i-x held pending; rejects if it throws. */
async function draw(tree: ReactNode) {
  await act(async () => page.tab.app.issue.issues.acceptRead(detailOf("x"), page.tab.app.issue.captureRead()));
  await act(async () =>
    page.root.render(
      <StoreContext.Provider value={page.tab.app}>
        <MemoryRouter initialEntries={["/acme/projects/p-web/intake?currentTab=open&inboxIssueId=i-x"]}>
          {tree}
        </MemoryRouter>
      </StoreContext.Provider>
    )
  );
}

/** Starts a write of the action model's, its first changes drawn; gives what awaits the write itself. */
async function started(write: () => Promise<void>): Promise<() => Promise<void>> {
  let out: Promise<void> = Promise.resolve();
  await act(async () => {
    out = write();
  });
  return () => out;
}

/** nerve triages i-x as intake says, and its read again in the triage's turn answers the same. */
async function confirm(intake: IssueIntake) {
  await answer(callTo("POST", TRIAGE), json(200, detailOf("x", { intake })));
  await answer(callTo("GET", ISSUE), json(200, detailOf("x", { intake })));
}

describe("the selection's port (R18)", () => {
  it("is all the action model reads of the selection: with no intake page's state around it, it offers its actions", async () => {
    await expect(draw(<Detail port={portOf()} headers={false} />)).resolves.toBeUndefined();
    expect(model?.offered).toEqual(["accept", "decline", "snooze", "duplicate", "copy", "delete"]);
  });

  it.each<[string, () => Promise<void>]>([
    [
      "declined",
      async () => {
        const declined = await started(() => model!.triage({ status: "declined" }));
        await confirm(intakeOf("declined"));
        await act(async () => void (await declined()));
      },
    ],
    [
      "accepted, as its modal closes",
      async () => {
        await act(async () => model!.take("accept"));
        const accepted = await started(() => model!.accept());
        await confirm(intakeOf("accepted"));
        await act(async () => void (await accepted()));
        await act(async () => model!.close());
      },
    ],
  ])("leaves the work item %s for the row the port gives as it began, never the page's list's", async (_, triage) => {
    // the page's state and list around it: the row after i-x in the list the page shows is i-y
    await act(async () => void (await loadList(page.tab.nerve, page.tab.store, "p-web", open, pageOf([x, y]))));
    const port = portOf();
    await draw(
      <IntakePageStateProvider>
        <Detail port={port} headers={false} />
      </IntakePageStateProvider>
    );
    await triage();
    expect(port.neighbour).toHaveBeenCalledWith("i-x");
    expect(port.leave).toHaveBeenCalledExactlyOnceWith("i-x", "i-z");
  });

  it("gives the headers the navigation they draw: the wide header the wide screen's, the narrow header the narrow's", async () => {
    await draw(
      <IntakePageStateProvider>
        <Detail
          port={portOf((screen) => (
            <i data-navigation={screen} />
          ))}
          headers
        />
      </IntakePageStateProvider>
    );
    const drawn = [...page.host.querySelectorAll<HTMLElement>("[data-navigation]")];
    expect(drawn.map((element) => [element.dataset.navigation, !!element.closest(".lg\\:hidden")])).toEqual([
      ["wide", false],
      ["narrow", true],
    ]);
    // none of the page's own: its list and its steps are the page's to give
    expect(page.host.querySelector('button[aria-label="inbox_issue.a11y.previous"]')).toBeNull();
    expect(page.host.querySelector('button[aria-label="inbox_issue.a11y.list_toggle"]')).toBeNull();
  });

  it("is, the intake page's, the page's navigation as before: the steps in both headers, the narrow one's list toggle", async () => {
    await opened([x, y], detailOf("x"));
    const named = (label: string) => [
      ...detailRegion()!.querySelectorAll<HTMLButtonElement>(`[aria-label="${label}"]`),
    ];
    expect(named("inbox_issue.a11y.previous")).toHaveLength(2);
    expect(named("inbox_issue.a11y.next")).toHaveLength(2);
    const [toggle, ...others] = named("inbox_issue.a11y.list_toggle");
    expect(others).toEqual([]);
    expect(toggle?.closest(".lg\\:hidden")).not.toBeNull();
    // the list beside the detail, shown; the toggle hides it on a narrow screen, and says so
    expect(toggle?.getAttribute("aria-expanded")).toBe("true");
    expect(page.host.querySelector('[class~="-translate-x-full"]')).toBeNull();
    await act(async () => toggle!.click());
    expect(named("inbox_issue.a11y.list_toggle")[0]?.getAttribute("aria-expanded")).toBe("false");
    expect(page.host.querySelector('[class~="-translate-x-full"]')).not.toBeNull();
    // a step selects the next row of the page's list
    await act(async () => named("inbox_issue.a11y.next")[0]!.click());
    await settled();
    expect(selected()).toBe("i-y");
  });

  it("is, the intake page's, its landing's region: the list hidden on a narrow screen, a decline lands on the detail", async () => {
    vi.stubGlobal("innerWidth", 800);
    await opened([x, y], detailOf("x"));
    const toggle = detailRegion()!.querySelector<HTMLButtonElement>('[aria-label="inbox_issue.a11y.list_toggle"]');
    await act(async () => toggle!.click());
    await chooseFromMenu("narrow", "inbox_issue.actions.decline");
    await act(async () => buttonSaying("decline", dialogNamed("inbox_issue.modals.decline.title"))!.click());
    await settled();
    await confirm(intakeOf("declined"));
    await settled();
    expect(selected()).toBe("i-y");
    // the page's landing: the list hidden, the detail region the port gave it, around the work item selected
    expect(document.activeElement).toBe(detailRegion());
  });
});
