// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";
import type { IssueIntake } from "@nerve/api-client";
import type { EmptyStateCompact, EmptyStateDetailed } from "@nerve/propel/empty-state";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { toasts } from "@/lib/fake-toast";
import type { RootStore } from "@/store/root.store";
import { issueOf } from "@/store/issue/fake-issues";
import { acme, detailOf, intakeIssueOf } from "@/store/issue/intake/fake-intake";
import { stateOf } from "@/store/project/fake-projects";
import {
  answer,
  buttonSaying,
  callTo,
  callsTo,
  chooseFromMenu,
  detailRegion,
  dialogNamed,
  escape,
  listRegion,
  menuButton,
  opened,
  page,
  pressByKeyboard,
  pressByPointer,
  rowOf,
  selected,
  settled,
  shown,
  useIntakePageFixture,
} from "../../fake-intake-page";

// What a triage of the selected work item of a project's intake does with the selection and the focus (P5a spec 2.6,
// 2.8, R14, R35; re-94 I4, m10), its modals', its header's and its list's real: once nerve has confirmed it, the
// selection moves, by a replace, to the row after the one triaged as the triage began, else the row before, else none,
// and the focus with it (the row, or the list); marking a duplicate leaves it, the focus then on the detail (the work
// item closed, no primary action) — a conflict too, on the new header's primary action if it has one; a refusal leaves
// both, the focus in the modal, or back at the control that took the action; a cancel gives the focus back to that
// control (the menu's button for a menu's item), whatever the press focused; a selection the user changed meanwhile
// is left as it is. While a triage is out, the header's controls wait and its modal does not close. Never a console
// error. The page's session is the fixture's (fake-intake-page.tsx).

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
vi.mock("@/hooks/store/user/user-user-profile", () => ({
  useUserProfile: () => ({ data: { start_of_the_week: 1 } }),
}));
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
// the accept modal's own test is accept.test.tsx
vi.mock("@/components/issues/issue-modal/modal", () => ({ CreateUpdateIssueModal: () => null }));

useIntakePageFixture();

const TRIAGE = (id: string) => `/api/v0/intake-issues/${id}/triage`;
const ISSUE = (id: string) => `/api/v0/issues/${id}`;
const intakeOf = (status: IssueIntake["status"], fields: Partial<IssueIntake> = {}): IssueIntake => ({
  status,
  snoozed_till: null,
  duplicate_to_id: null,
  ...fields,
});
const [x, y, z] = ["x", "y", "z"].map((name) => intakeIssueOf(name));
// 25 November 2026, 15:10 in the browser's zone; a snooze held till 30 November
const NOW = new Date(2026, 10, 25, 15, 10);
const snoozed = intakeOf("snoozed", { snoozed_till: new Date(2026, 10, 30).toISOString() });

/** nerve triages the work item as intake says, then its read again in the triage's turn answers the same. */
async function confirm(id: string, intake: IssueIntake) {
  await answer(callTo("POST", TRIAGE(id)), json(200, detailOf(id.slice(2), { intake })));
  await answer(callTo("GET", ISSUE(id)), json(200, detailOf(id.slice(2), { intake })));
  await settled();
}
const decline = async () => {
  await act(async () => buttonSaying("decline", dialogNamed("inbox_issue.modals.decline.title"))!.click());
  await settled();
};

describe("the selection after a triage (R14)", () => {
  it("moves to the row after, by a replace, once nerve has declined it, the focus with it; Back never returns", async () => {
    await opened([x, y, z], detailOf("x"));
    await pressByKeyboard(buttonSaying("inbox_issue.actions.decline", detailRegion()!));
    expect(dialogNamed("inbox_issue.modals.decline.title")?.textContent).toContain(
      "inbox_issue.modals.decline.content"
    );
    await decline();
    expect((await callTo("POST", TRIAGE("i-x"))).body).toEqual({ status: "declined" });
    expect(selected()).toBe("i-x");
    await confirm("i-x", intakeOf("declined"));
    expect(page.where).toEqual({ search: "?currentTab=open&inboxIssueId=i-y", type: "REPLACE" });
    expect(document.activeElement).toBe(rowOf("i-y"));
    await act(async () => void page.navigate(-1));
    expect(page.where?.search).toBe("");
  });

  it("moves to the row before for the last row, and clears it for the only one, the focus on the list", async () => {
    await opened([x, y], detailOf("x"));
    await act(async () => rowOf("i-y")!.click());
    await shown(detailOf("y"));
    await pressByKeyboard(buttonSaying("inbox_issue.actions.decline", detailRegion()!));
    await decline();
    await confirm("i-y", intakeOf("declined"));
    expect(page.where).toEqual({ search: "?currentTab=open&inboxIssueId=i-x", type: "REPLACE" });
    expect(document.activeElement).toBe(rowOf("i-x"));
    await shown(detailOf("x"), 2);
    await pressByKeyboard(buttonSaying("inbox_issue.actions.decline", detailRegion()!));
    await decline();
    await confirm("i-x", intakeOf("declined"));
    expect(page.where).toEqual({ search: "?currentTab=open", type: "REPLACE" });
    expect(document.activeElement).toBe(listRegion());
  });

  it("moves after a snooze from the narrow header's menu: tomorrow's first moment, the status and it alone", async () => {
    vi.setSystemTime(NOW);
    await opened([x, y, z], detailOf("x"));
    await chooseFromMenu("narrow", "inbox_issue.actions.snooze");
    await act(async () =>
      buttonSaying("inbox_issue.actions.snooze", dialogNamed("inbox_issue.modals.snooze.title"))!.click()
    );
    expect((await callTo("POST", TRIAGE("i-x"))).body).toEqual({
      status: "snoozed",
      snoozed_till: new Date(2026, 10, 26).toISOString(),
    });
    await confirm("i-x", intakeOf("snoozed", { snoozed_till: new Date(2026, 10, 26).toISOString() }));
    expect(page.where).toEqual({ search: "?currentTab=open&inboxIssueId=i-y", type: "REPLACE" });
    expect(document.activeElement).toBe(rowOf("i-y"));
  });

  it("moves after an unsnooze, which sends the pending status alone; a new snooze starts at the held day", async () => {
    vi.setSystemTime(NOW);
    await opened([x, y], detailOf("x", { intake: snoozed }));
    await chooseFromMenu("wide", "inbox_issue.actions.resnooze");
    expect(dialogNamed("inbox_issue.modals.snooze.title")?.textContent).toContain("Nov 30, 2026");
    await act(async () => buttonSaying("common.cancel", dialogNamed("inbox_issue.modals.snooze.title"))!.click());
    await settled();
    await chooseFromMenu("wide", "inbox_issue.actions.unsnooze");
    expect((await callTo("POST", TRIAGE("i-x"))).body).toEqual({ status: "pending" });
    await confirm("i-x", intakeOf("pending"));
    expect(page.where).toEqual({ search: "?currentTab=open&inboxIssueId=i-y", type: "REPLACE" });
    expect(document.activeElement).toBe(rowOf("i-y"));
  });

  it("stays on a duplicate, the focus on the detail: the work item closed, the opener gone with its header", async () => {
    await opened([x, y], detailOf("x"));
    await chooseFromMenu("wide", "inbox_issue.actions.mark_as_duplicate");
    const search = await callTo("GET", `/api/v0/workspaces/${acme.slug}/issue-search`);
    expect(search.query).toEqual({ project_id: "p-web" });
    await answer(search, json(200, { data: [issueOf("i-d", "p-web", acme.id), issueOf("i-x", "p-web", acme.id)] }));
    // the search reads the project's states, which its candidates show
    const web = page.tab.app.projectRoot.project.getProjectById("p-web")!;
    await answer(
      callTo("GET", "/api/v0/projects/p-web/states"),
      json(200, { data: [stateOf(web, "Todo", "unstarted", 1)] })
    );
    const options = [...document.querySelectorAll<HTMLElement>('[role="option"]')];
    expect(options.map((option) => option.textContent)).toEqual([expect.stringContaining("i-d")]);
    await act(async () => {
      options[0]!.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true, button: 0 }));
      options[0]!.click();
    });
    expect((await callTo("POST", TRIAGE("i-x"))).body).toEqual({ status: "duplicate", duplicate_to_id: "i-d" });
    // it cannot be closed while nerve marks it
    await escape();
    expect(dialogNamed("inbox_issue.actions.mark_as_duplicate")).toBeDefined();
    await confirm("i-x", intakeOf("duplicate", { duplicate_to_id: "i-d" }));
    expect(dialogNamed("inbox_issue.actions.mark_as_duplicate")).toBeUndefined();
    expect(page.where).toEqual({ search: "?currentTab=open&inboxIssueId=i-x", type: "REPLACE" });
    expect(document.activeElement).toBe(detailRegion());
  });

  it("stays on a refusal, which is said, the modal open with the focus in it; nothing read again", async () => {
    await opened([x, y], detailOf("x"));
    await pressByKeyboard(buttonSaying("inbox_issue.actions.decline", detailRegion()!));
    await decline();
    await answer(callTo("POST", TRIAGE("i-x")), problem(503, "server_busy"));
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.server_busy" }]);
    expect(selected()).toBe("i-x");
    expect(dialogNamed("inbox_issue.modals.decline.title")?.contains(document.activeElement)).toBe(true);
    expect(callsTo("GET", ISSUE("i-x"))).toEqual([]);
  });

  it("is left as the user changed it while the triage was out, the focus where he put it", async () => {
    await opened([x, y, z], detailOf("x", { intake: snoozed }));
    await chooseFromMenu("wide", "inbox_issue.actions.unsnooze");
    await callTo("POST", TRIAGE("i-x"));
    await pressByKeyboard(rowOf("i-z"));
    await settled();
    // nerve confirms it, read again in its turn, once the user is on another work item
    await confirm("i-x", intakeOf("pending"));
    expect(callsTo("GET", ISSUE("i-x"))).toHaveLength(1);
    expect(page.where).toEqual({ search: "?currentTab=open&inboxIssueId=i-z", type: "PUSH" });
    expect(document.activeElement).toBe(rowOf("i-z"));
  });

  it("is cleared once nerve has deleted it, by a replace, the focus on the list", async () => {
    await opened([x, y], detailOf("x"));
    await chooseFromMenu("wide", "inbox_issue.actions.delete");
    expect(dialogNamed("inbox_issue.modals.delete.title")?.textContent).toContain("inbox_issue.modals.delete.content");
    await act(async () => buttonSaying("delete", dialogNamed("inbox_issue.modals.delete.title"))!.click());
    await answer(callTo("DELETE", "/api/v0/intake-issues/i-x"), noContent());
    await answer(callTo("GET", ISSUE("i-x")), problem(404, "issue.not_found"));
    await settled();
    expect(page.where?.type).toBe("REPLACE");
    expect(document.activeElement).toBe(listRegion());
    expect(toasts).toEqual([{ type: "success", title: "success", message: "inbox_issue.modals.delete.success" }]);
  });
});

describe("the focus after a triage that stays, a refusal and a cancel (R35)", () => {
  it("lands on the new header's primary action once a conflict read it again: Accept, for a pending one", async () => {
    await opened([x, y], detailOf("x", { intake: snoozed }));
    await chooseFromMenu("wide", "inbox_issue.actions.unsnooze");
    await answer(callTo("POST", TRIAGE("i-x")), problem(409, "issue.intake_transition_not_allowed"));
    await answer(callTo("GET", ISSUE("i-x")), json(200, detailOf("x")));
    await settled();
    expect(toasts).toHaveLength(1);
    expect(selected()).toBe("i-x");
    expect(document.activeElement).toBe(buttonSaying("inbox_issue.actions.accept", detailRegion()!));
  });

  it("goes back, as a modal is cancelled, to the button that opened it, though the pointer's press did not focus it", async () => {
    await opened([x, y], detailOf("x"));
    const opener = buttonSaying("inbox_issue.actions.decline", detailRegion()!);
    await pressByPointer(opener);
    await settled();
    await act(async () => buttonSaying("common.cancel", dialogNamed("inbox_issue.modals.decline.title"))!.click());
    await settled();
    expect(document.activeElement).toBe(opener);
  });

  it("lands on the detail as a modal is cancelled whose opener a read again withdrew, never on the page's body", async () => {
    await opened([x, y], detailOf("x"));
    await pressByKeyboard(buttonSaying("inbox_issue.actions.decline", detailRegion()!));
    await settled();
    // someone else accepts it meanwhile, which a read again shows: its header offers no triage
    let again: Promise<unknown> = Promise.resolve();
    await act(async () => {
      again = page.tab.app.issue.issues.refreshAuthoritative("i-x");
    });
    await answer(callTo("GET", ISSUE("i-x")), json(200, detailOf("x", { intake: intakeOf("accepted") })));
    await act(async () => void (await again));
    expect(buttonSaying("inbox_issue.actions.decline", detailRegion()!)).toBeUndefined();
    await act(async () => buttonSaying("common.cancel", dialogNamed("inbox_issue.modals.decline.title"))!.click());
    await settled();
    expect(selected()).toBe("i-x");
    expect(document.activeElement).toBe(detailRegion());
  });

  it.each(["wide", "narrow"] as const)(
    "goes back, as a modal opened from the %s header's menu is cancelled, to the menu's button",
    async (header) => {
      vi.setSystemTime(NOW);
      await opened([x, y], detailOf("x"));
      await chooseFromMenu(header, "inbox_issue.actions.snooze");
      await act(async () => buttonSaying("common.cancel", dialogNamed("inbox_issue.modals.snooze.title"))!.click());
      await settled();
      expect(document.activeElement).toBe(menuButton(header));
    }
  );

  it("goes back to the menu's button after a refused unsnooze, which no modal holds, once its controls wait no more", async () => {
    await opened([x, y], detailOf("x", { intake: snoozed }));
    await chooseFromMenu("wide", "inbox_issue.actions.unsnooze");
    const triage = await callTo("POST", TRIAGE("i-x"));
    // the header's controls wait while the triage is out
    expect(buttonSaying("inbox_issue.actions.decline", detailRegion()!)?.disabled).toBe(true);
    expect(menuButton("wide")?.disabled).toBe(true);
    await answer(triage, problem(403, "forbidden"));
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.forbidden" }]);
    expect(buttonSaying("inbox_issue.actions.decline", detailRegion()!)?.disabled).toBe(false);
    expect(document.activeElement).toBe(menuButton("wide"));
  });
});
