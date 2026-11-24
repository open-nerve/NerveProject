// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// P5b spec 2.6, R20 (Q34): each dropdown that changes the assignees of an existing work item hands its write a change
// of additions and removals against the set it showed as it was picked (the value it drew), never the whole set the
// dropdown made of it; the write makes the whole set in its turn, from the set then held (issue-write-assignees.test).
// The seven: the intake detail's properties, the preview's, the detail's sidebar, a list row's, the spreadsheet's
// column, a sub-work item's and a relation's. The create and edit forms and the drafts keep their whole sets.
import { act } from "react";
import type { PropsWithChildren, ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import type { Issue } from "@nerve/api-client";
import type { IssueChange } from "@nerve/types";
import { InboxIssueContentProperties } from "@/components/inbox/content/issue-properties";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { IssueDetailsSidebar } from "./issue-detail/sidebar";
import { SubIssuesListItemProperties } from "./issue-detail-widgets/sub-issues/issues-list/properties";
import { IssueProperties } from "./issue-layouts/properties/all-properties";
import { SpreadsheetAssigneeColumn } from "./issue-layouts/spreadsheet/columns/assignee-column";
import { PeekOverviewProperties } from "./peek-overview/properties";
import { RelationIssueProperty } from "./relations/properties";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "q34" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "q34" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("react-router", () => ({
  useParams: () => ({ workspaceSlug: "ws" }),
  Link: ({ children }: PropsWithChildren) => children,
}));
vi.mock("@makeplane/propel/components/tooltip", () => ({ Tooltip: ({ children }: PropsWithChildren) => children }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => tab.app.state }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => ({ getProjectById: () => undefined }) }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: () => undefined }) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
// A list row's change runs as the detail's command does, its refusal its own.
vi.mock("./issue-detail/use-issue-operations", () => ({
  useDetailCommand: () => (command: () => Promise<void>) => command(),
}));
// The assignees' dropdown as each caller draws it: the value it shows and what it does with the set picked.
vi.mock("@/components/dropdowns/member/dropdown", () => ({
  MemberDropdown: (props: { multiple?: boolean; value: string[]; onChange: (ids: string[]) => void }) => {
    if (props.multiple) drawn = props;
    return null;
  },
}));
// The other properties have their own tests; here each is absent.
vi.mock("@/components/dropdowns/state/dropdown", () => ({ StateDropdown: () => null }));
vi.mock("@/components/dropdowns/priority", () => ({ PriorityDropdown: () => null }));
vi.mock("@/components/dropdowns/date", () => ({ DateDropdown: () => null }));
vi.mock("@/components/dropdowns/date-range", () => ({ DateRangeDropdown: () => null }));
vi.mock("@/components/issues/parent-select-root", () => ({ IssueParentSelectRoot: () => null }));
vi.mock("./issue-detail/label", () => ({ IssueLabel: () => null }));
vi.mock("./issue-detail/planning-properties", () => ({ IssuePlanningProperties: () => null }));
vi.mock("./issue-layouts/properties/labels", () => ({ IssuePropertyLabels: () => null }));
vi.mock("./issue-layouts/properties/row-planning-properties", () => ({ IssueRowPlanningProperties: () => null }));
vi.mock("@/components/inbox/content/duplicate-of", () => ({ IntakeDuplicateOf: () => null }));

let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let drawn: { value: string[]; onChange: (ids: string[]) => void } | undefined;
let host: HTMLDivElement;
let renderer: ReturnType<typeof createRoot>;

beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  drawn = undefined;
  tab = await issueStoreTab();
  // x, in its project's intake, assigned to A and Z (a member the page does not know).
  const detail = { ...issueDetailOf("x"), assignee_ids: ["A", "Z"] };
  tab.app.issue.issues.acceptRead(
    { ...detail, intake: { status: "pending", snoozed_till: null, duplicate_to_id: null } },
    tab.app.issue.captureRead()
  );
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

/** The work item x as held: the snapshot each caller draws. */
const held = (): Issue => tab.app.issue.issues.getIssueById("x")!;
/** The detail's operations, their update handing its change to handed. */
const operations = (handed: Mock) => ({
  fetch: async () => {},
  update: async (_ws: string, _project: string, _id: string, change: IssueChange) => void handed(change),
});

const callers: { name: string; draw: (handed: Mock) => ReactNode }[] = [
  {
    name: "the intake detail's properties",
    draw: (handed) => (
      <InboxIssueContentProperties
        workspaceSlug="ws"
        projectId="A"
        issueId="x"
        issueOperations={operations(handed)}
        isEditable
      />
    ),
  },
  {
    name: "the preview's properties",
    draw: (handed) => (
      <PeekOverviewProperties
        workspaceSlug="ws"
        projectId="A"
        issueId="x"
        issueOperations={operations(handed)}
        disabled={false}
      />
    ),
  },
  {
    name: "the detail's sidebar",
    draw: (handed) => (
      <IssueDetailsSidebar
        workspaceSlug="ws"
        projectId="A"
        issueId="x"
        issueOperations={operations(handed)}
        isEditable
      />
    ),
  },
  {
    name: "a list row's properties",
    draw: (handed) => (
      <IssueProperties
        issue={held()}
        updateIssue={async (_project, _id, change) => void handed(change)}
        displayProperties={{ assignee: true }}
        isReadOnly={false}
        className=""
        activeLayout="list"
      />
    ),
  },
  {
    name: "the spreadsheet's assignee column",
    draw: (handed) => (
      <SpreadsheetAssigneeColumn
        issue={held()}
        onChange={(_, change) => handed(change)}
        disabled={false}
        onClose={() => {}}
      />
    ),
  },
  {
    name: "a sub-work item's properties",
    draw: (handed) => (
      <SubIssuesListItemProperties
        workspaceSlug="ws"
        parentIssueId="p"
        issueId="x"
        canEdit
        updateSubIssue={async (_ws, _project, _parent, _id, change) => void handed(change)}
        displayProperties={{ assignee: true }}
        issue={held()}
      />
    ),
  },
  {
    name: "a relation's properties",
    draw: (handed) => (
      <RelationIssueProperty
        workspaceSlug="ws"
        issueId="x"
        disabled={false}
        issueOperations={{ ...operations(handed), copyLink: async () => {}, remove: async () => {} }}
      />
    ),
  },
];

describe.each(callers)("$name", ({ draw }) => {
  /** Draws the caller, and checks that its dropdown shows the set held. */
  async function shown(handed: Mock) {
    await act(async () => renderer.render(draw(handed)));
    expect(drawn?.value, "the set the dropdown shows").toEqual(["A", "Z"]);
  }

  it("hands its write the member picked as an addition to the set it showed, not the whole set", async () => {
    const handed = vi.fn();
    await shown(handed);
    await act(async () => drawn!.onChange(["A", "Z", "B"]));
    expect(handed.mock.calls).toEqual([[{ assignees: { add: ["B"], remove: [] } }]]);
  });

  it("hands its write the member taken out as a removal from the set it showed, not the whole set", async () => {
    const handed = vi.fn();
    await shown(handed);
    await act(async () => drawn!.onChange(["Z"]));
    expect(handed.mock.calls).toEqual([[{ assignees: { add: [], remove: ["A"] } }]]);
  });
});
