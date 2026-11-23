// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// P5b spec 2.4, R13: a work item's generic surfaces, its preview's properties and its detail's sidebar, show no State
// while it is in its project's intake (its detail held with its intake row, pending, snoozed, declined or a duplicate),
// nerve keeping it in the project's triage state; one accepted, one never submitted to an intake (intake null) and one
// held as a list's row alone show it as before. The other properties stay as they are.
import { act } from "react";
import type { ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { IntakeStatus, Issue, IssueDetail } from "@nerve/api-client";
import { issueDetailOf, issueOf, issueStoreTab } from "@/store/issue/fake-issues";
import { IssueDetailsSidebar } from "./issue-detail/sidebar";
import { useIssueOperations } from "./issue-detail/use-issue-operations";
import { PeekOverviewProperties } from "./peek-overview/properties";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "state" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "state" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "ws" }) }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => tab.app.state }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: () => undefined }) }));
// The properties' controls have their own tests; here each is absent, its row's label alone drawn.
vi.mock("@/components/dropdowns/state/dropdown", () => ({ StateDropdown: () => null }));
vi.mock("@/components/dropdowns/member/dropdown", () => ({ MemberDropdown: () => null }));
vi.mock("@/components/dropdowns/priority", () => ({ PriorityDropdown: () => null }));
vi.mock("@/components/dropdowns/date", () => ({ DateDropdown: () => null }));
vi.mock("@/components/issues/parent-select-root", () => ({ IssueParentSelectRoot: () => null }));
vi.mock("./issue-detail/label", () => ({ IssueLabel: () => null }));
vi.mock("./issue-detail/planning-properties", () => ({ IssuePlanningProperties: () => null }));

let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let renderer: ReturnType<typeof createRoot>;

beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await issueStoreTab();
  host = document.createElement("div");
  document.body.append(host);
  renderer = createRoot(host);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  host.remove();
  tab.app.dispose();
  vi.useRealTimers();
});

function Preview() {
  const issueOperations = useIssueOperations("ws", "A", "x");
  return (
    <PeekOverviewProperties workspaceSlug="ws" projectId="A" issueId="x" issueOperations={issueOperations} disabled />
  );
}
function Sidebar() {
  const issueOperations = useIssueOperations("ws", "A", "x");
  return (
    <IssueDetailsSidebar
      workspaceSlug="ws"
      projectId="A"
      issueId="x"
      issueOperations={issueOperations}
      isEditable={false}
    />
  );
}

/** The work item x as its detail answers it, its intake row of status, or none (null). */
const detailIn = (status: IntakeStatus | null): IssueDetail => ({
  ...issueDetailOf("x"),
  intake: status === null ? null : { status, snoozed_till: null, duplicate_to_id: null },
});
/** Holds x as nerve answered it, then draws view. */
async function show(answer: Issue | IssueDetail, view: ReactNode) {
  tab.app.issue.issues.acceptRead(answer, tab.app.issue.captureRead());
  await act(async () => renderer.render(view));
}
/** The labels of the properties' rows, in order. */
const labels = () => [...host.querySelectorAll("span")].map((label) => label.textContent);

describe.each([
  { where: "its preview's properties", view: <Preview /> },
  { where: "its detail's sidebar", view: <Sidebar /> },
])("$where", ({ view }) => {
  it.each(["pending", "snoozed", "declined", "duplicate"] as const)(
    "show no State for a work item %s in its project's intake, the other properties as they are",
    async (status) => {
      await show(detailIn(status), view);
      expect(labels()).toEqual([
        "common.assignees",
        "common.priority",
        "common.order_by.start_date",
        "common.order_by.due_date",
        "common.parent",
        "common.labels",
      ]);
    }
  );

  it.each([
    { what: "accepted from its project's intake", answer: detailIn("accepted") },
    { what: "never submitted to an intake", answer: detailIn(null) },
    { what: "held as a list's row alone", answer: issueOf("x") },
  ])("show the State of a work item $what", async ({ answer }) => {
    await show(answer, view);
    expect(labels()).toEqual([
      "common.state",
      "common.assignees",
      "common.priority",
      "common.order_by.start_date",
      "common.order_by.due_date",
      "common.parent",
      "common.labels",
    ]);
  });
});
