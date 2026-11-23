// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, forwardRef, useImperativeHandle, useState, type ComponentProps } from "react";
import { useFormContext } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";
import type { IssueDetail, IssueIntake } from "@nerve/api-client";
import type { EmptyStateCompact, EmptyStateDetailed } from "@nerve/propel/empty-state";
import type { RichTextEditor } from "@/components/editor/rich-text";
import type { IssueFormValues } from "@/components/issues/issue-modal/types";
import { json, problem } from "@/lib/auth/fake-nerve";
import { toasts } from "@/lib/fake-toast";
import type { RootStore } from "@/store/root.store";
import { issueCommentOf } from "@/store/issue/fake-issues";
import { detailOf, intakeIssueOf } from "@/store/issue/intake/fake-intake";
import { stateOf } from "@/store/project/fake-projects";
import {
  answer,
  buttonSaying,
  callTo,
  callsTo,
  detailRegion,
  dialogNamed,
  listRegion,
  opened,
  page,
  pressByKeyboard,
  rowOf,
  selected,
  settled,
  useIntakePageFixture,
} from "../fake-intake-page";

// The accept modal of a work item of a project's intake (P5a spec 2.8, R12, R14, R29; re-94 I2, m10): M4's issue
// modal, real, through its props alone. Its form starts at the work item as held when it opened, its state at the
// project's default state; its submit first accepts the work item by a triage (beforeFormSubmit) and nothing else, a
// refusal of it stopping the update; once accepted the modal says so, and the update of what the form changed follows
// as M4's (PATCH /issues). The selection moves (a replace, to the row after) only once that update is confirmed and the
// modal closed, or as the modal is cancelled once the work item is accepted; an update refused stays under its field,
// the modal open, the selection kept, and its retry sends the update alone, as for a work item someone else accepted.
// A conflict closes the modal, the selection kept (or cleared for a work item gone from the intake). The page is the
// fixture's (fake-intake-page.tsx); the form's editor and its properties but a priority are stand-ins.

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
vi.mock("@nerve/propel/emoji-icon-picker", () => ({ Logo: () => null }));
vi.mock("@/hooks/use-profile-image", () => import("@/lib/fake-profile-image"));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/context/use-issue-modal", () => ({
  useIssueModal: () => ({ allowedProjectIds: ["p-web"], selectedParentIssue: null, setSelectedParentIssue: () => {} }),
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
vi.mock("@/components/issues/create-issue-toast-action-items", () => ({ CreateIssueToastActionItems: () => null }));
vi.mock("@/components/issues/issue-modal/components/parent-tag", () => ({ IssueParentTag: () => null }));
vi.mock("@/components/issues/issue-modal/components/default-properties", () => ({
  IssueDefaultProperties: () => <Properties />,
}));
vi.mock("@/components/editor/rich-text", () => ({
  RichTextEditor: forwardRef((props: ComponentProps<typeof RichTextEditor>, ref) => {
    const [value] = useState(props.initialValue);
    useImperativeHandle(ref, () => ({ isEditorReadyToDiscard: () => true, clearEditor: () => {} }));
    return <output data-description>{value}</output>;
  }),
}));

/** The form's properties: the state it holds, and a priority the user may pick. */
function Properties() {
  const { setValue, watch } = useFormContext<IssueFormValues>();
  return (
    <>
      <output data-state>{watch("state_id")}</output>
      <button type="button" onClick={() => setValue("priority", "urgent", { shouldDirty: true })}>
        urgent
      </button>
    </>
  );
}

useIntakePageFixture();

const TRIAGE = "/api/v0/intake-issues/i-x/triage";
const ISSUE = "/api/v0/issues/i-x";
const COMMENTS = `${ISSUE}/comments`;
const comments = json(200, { data: [issueCommentOf("c-1", "i-x")], next_cursor: null });
const accepted: IssueIntake = { status: "accepted", snoozed_till: null, duplicate_to_id: null };
const [x, y] = ["x", "y"].map((name) => intakeIssueOf(name));
const MOVE = "inbox_issue.actions.move";
const ACCEPTED = "inbox_issue.modals.accept.accepted";

/** Opens the page on x and y, x shown, the project's states read (Todo its default), and x's accept modal. */
async function accepting() {
  await opened([x, y], detailOf("x", { state_id: "s-triage" }));
  const states = page.tab.app.state.fetchProjectStates("p-web");
  const web = page.tab.app.projectRoot.project.getProjectById("p-web")!;
  await answer(
    callTo("GET", "/api/v0/projects/p-web/states"),
    json(200, {
      data: [
        stateOf(web, "Backlog", "backlog", 1, { id: "s-backlog" }),
        stateOf(web, "Todo", "unstarted", 2, { id: "s-todo", default: true }),
      ],
    })
  );
  await act(async () => void (await states));
  await pressByKeyboard(buttonSaying("inbox_issue.actions.accept", detailRegion()!));
  await settled();
}
const modal = () => dialogNamed(MOVE) ?? dialogNamed(ACCEPTED);
const submit = async () => {
  await act(async () => modal()!.querySelector("form")!.requestSubmit());
  await settled();
};
const pickUrgent = () => act(async () => buttonSaying("urgent", modal()!)!.click());
/** The accept triage answered, its read again answering detail. */
async function acceptedAs(detail: IssueDetail) {
  await answer(callTo("POST", TRIAGE), json(200, detail));
  await answer(callTo("GET", ISSUE), json(200, detail));
  await settled();
}
/**
 * A comment written on x, its creation answered: its read again of x in its turn (the n-th read of x) is out, x not
 * held meanwhile; gives what answers that read and the comments' read, then waits for the comment's write to end.
 */
async function commenting(n = 1) {
  const app = page.tab.app.issue;
  const written = app.issueDetail.comment.createComment("acme", "p-web", "i-x", { comment_html: "<p>c</p>" });
  await answer(callTo("POST", COMMENTS), json(201, issueCommentOf("c-1", "i-x")));
  await callTo("GET", ISSUE, n);
  expect(app.issues.getIssueDetailById("i-x")).toBeUndefined();
  return async (detail: IssueDetail) => {
    await answer(callTo("GET", ISSUE, n), json(200, detail));
    await answer(callTo("GET", COMMENTS), comments);
    await act(async () => void (await written));
  };
}
const refusedUnder = (field: string) =>
  json(422, { status: 422, code: "validation_failed", title: "", errors: [{ field, code: "required", message: "" }] });

describe("the accept modal (R12, re-94 I2)", () => {
  it("starts at the project's default state, the work item as held when it opened, named to move it", async () => {
    await accepting();
    expect(modal()).toBe(dialogNamed(MOVE));
    expect(modal()!.querySelector("[data-state]")?.textContent).toBe("s-todo");
    expect(modal()!.querySelector<HTMLInputElement>('input[name="name"]')?.value).toBe("x");
  });

  it("accepts by a triage alone, then sends the field changed; the selection moves only after both, the modal closed", async () => {
    await accepting();
    await pickUrgent();
    await submit();
    expect((await callTo("POST", TRIAGE)).body).toEqual({ status: "accepted" });
    expect(callsTo("PATCH", ISSUE)).toEqual([]);
    // the read again gives the work item renamed elsewhere: the form keeps what it opened with
    await acceptedAs(detailOf("x", { name: "renamed", intake: accepted, state_id: "s-todo" }));
    const update = await callTo("PATCH", ISSUE);
    expect(update.body).toMatchObject({ priority: "urgent" });
    expect(update.body).not.toHaveProperty("name");
    expect(update.body).not.toHaveProperty("state_id");
    expect(modal()).toBe(dialogNamed(ACCEPTED));
    expect(selected()).toBe("i-x");
    await answer(update, json(200, detailOf("x", { name: "renamed", intake: accepted, priority: "urgent" })));
    await settled();
    expect(modal()).toBeUndefined();
    expect(page.where).toEqual({ search: "?currentTab=open&inboxIssueId=i-y", type: "REPLACE" });
    expect(document.activeElement).toBe(rowOf("i-y"));
  });

  it("sends no update once the triage is refused, which is said; the modal open, the selection kept", async () => {
    await accepting();
    await pickUrgent();
    await submit();
    await answer(callTo("POST", TRIAGE), problem(403, "forbidden"));
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.forbidden" }]);
    expect(callsTo("PATCH", ISSUE)).toEqual([]);
    expect(modal()).toBe(dialogNamed(MOVE));
    expect(selected()).toBe("i-x");
  });

  it("keeps an update refused under its field, saying the work item is accepted; its retry sends the update alone", async () => {
    await accepting();
    await pickUrgent();
    await submit();
    await acceptedAs(detailOf("x", { intake: accepted, state_id: "s-todo" }));
    await answer(callTo("PATCH", ISSUE), refusedUnder("name"));
    await settled();
    expect(modal()).toBe(dialogNamed(ACCEPTED));
    expect(modal()!.querySelector('input[name="name"]')?.getAttribute("aria-invalid")).toBe("true");
    expect(selected()).toBe("i-x");
    await submit();
    expect(callsTo("POST", TRIAGE)).toHaveLength(1);
    await answer(callTo("PATCH", ISSUE, 2), json(200, detailOf("x", { intake: accepted, priority: "urgent" })));
    await settled();
    expect(modal()).toBeUndefined();
    expect(page.where).toEqual({ search: "?currentTab=open&inboxIssueId=i-y", type: "REPLACE" });
  });

  it("moves the selection as the modal is cancelled after a refused update: the work item is accepted", async () => {
    await accepting();
    await pickUrgent();
    await submit();
    await acceptedAs(detailOf("x", { intake: accepted, state_id: "s-todo" }));
    await answer(callTo("PATCH", ISSUE), problem(503, "server_busy"));
    await settled();
    expect(selected()).toBe("i-x");
    await act(async () => buttonSaying("discard", modal()!)!.click());
    await settled();
    expect(page.where).toEqual({ search: "?currentTab=open&inboxIssueId=i-y", type: "REPLACE" });
    expect(document.activeElement).toBe(rowOf("i-y"));
  });

  it("sends the update alone, its first submit too, for a work item someone else accepted meanwhile (R29)", async () => {
    await accepting();
    let again: Promise<unknown> = Promise.resolve();
    await act(async () => {
      again = page.tab.app.issue.issues.refreshAuthoritative("i-x");
    });
    await answer(callTo("GET", ISSUE), json(200, detailOf("x", { intake: accepted, state_id: "s-todo" })));
    await act(async () => void (await again));
    await pickUrgent();
    await submit();
    expect(callsTo("POST", TRIAGE)).toEqual([]);
    await answer(callTo("PATCH", ISSUE), json(200, detailOf("x", { intake: accepted, priority: "urgent" })));
    await settled();
    expect(page.where).toEqual({ search: "?currentTab=open&inboxIssueId=i-y", type: "REPLACE" });
  });

  it("accepts by a triage a work item not held as it is submitted, a comment's read again out (R29 is for one held)", async () => {
    await accepting();
    const commented = await commenting();
    await submit();
    await commented(detailOf("x", { state_id: "s-triage" }));
    expect((await callTo("POST", TRIAGE)).body).toEqual({ status: "accepted" });
    await acceptedAs(detailOf("x", { intake: accepted, state_id: "s-todo" }));
  });

  it("sends a retry's update alone, though a comment's read again is out as it is submitted: it accepted the work item", async () => {
    await accepting();
    await pickUrgent();
    await submit();
    await acceptedAs(detailOf("x", { intake: accepted, state_id: "s-todo" }));
    await answer(callTo("PATCH", ISSUE), problem(503, "server_busy"));
    const commented = await commenting(2);
    await submit();
    await commented(detailOf("x", { intake: accepted, state_id: "s-todo" }));
    expect(callsTo("POST", TRIAGE)).toHaveLength(1);
    await answer(callTo("PATCH", ISSUE, 2), json(200, detailOf("x", { intake: accepted, priority: "urgent" })));
    // the update reads the comments again in its turn, then the modal closes
    await answer(callTo("GET", COMMENTS, 2), comments);
    expect(modal()).toBeUndefined();
  });

  it("closes on a conflict, which is said, the selection kept, the focus on the detail its read again redrew", async () => {
    await accepting();
    await submit();
    await answer(callTo("POST", TRIAGE), problem(409, "issue.intake_transition_not_allowed"));
    await answer(callTo("GET", ISSUE), json(200, detailOf("x", { intake: { ...accepted, status: "declined" } })));
    await settled();
    expect(toasts).toEqual([
      { type: "error", title: "toast.error", message: "errors.issue_intake_transition_not_allowed" },
    ]);
    expect(modal()).toBeUndefined();
    expect(callsTo("PATCH", ISSUE)).toEqual([]);
    expect(selected()).toBe("i-x");
    expect(document.activeElement).toBe(detailRegion());
  });

  it("closes once nerve says the work item is gone from the intake, the selection cleared, the focus on the list", async () => {
    await accepting();
    await submit();
    await answer(callTo("POST", TRIAGE), problem(404, "issue.intake_not_found"));
    await settled();
    expect(toasts).toHaveLength(1);
    expect(modal()).toBeUndefined();
    expect(callsTo("PATCH", ISSUE)).toEqual([]);
    expect(page.where?.type).toBe("REPLACE");
    expect(selected()).not.toBe("i-x");
    expect(document.activeElement).toBe(listRegion());
  });

  it("is cancelled with nothing sent, the selection kept, the focus back on Accept", async () => {
    await accepting();
    await act(async () => buttonSaying("discard", modal()!)!.click());
    await settled();
    expect(modal()).toBeUndefined();
    expect(page.tab.nerve.calls.filter((call) => call.method !== "GET")).toEqual([]);
    expect(selected()).toBe("i-x");
    expect(document.activeElement).toBe(buttonSaying("inbox_issue.actions.accept", detailRegion()!));
  });
});
