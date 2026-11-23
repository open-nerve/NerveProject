// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { IssueIntake, Project } from "@nerve/api-client";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { acme, detailOf, intakeTab } from "@/store/issue/intake/fake-intake";
import { loadArchivedProjects, loadProjects, projectOf } from "@/store/project/fake-projects";
import type { DescriptionVersionsRoot } from "@/components/core/description-versions";
import type { IssueTitleInput } from "@/components/issues/title-input";
import { IntakeDetail } from "./detail";
import type { IntakeDuplicateOf } from "./duplicate-of";

// The properties of the selected work item of a project's intake (P5a spec 2.7, 2.9, R15, R27): its assignees,
// priority, due date and labels, changed by the project's admin or by a member who submitted it, while it is in the
// intake (pending or snoozed) and the project is not archived; a guest who submitted it changes its name and
// description alone; triaged, it is read-only; its state is shown only once it is accepted, read-only (a work item
// in the intake is in a state no catalog lists); the work item it duplicates is drawn below them. The detail and its
// property controls are real, as are the stores; the name, the description, the reactions, the versions, the activity
// and the duplicate's row are stand-ins.

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "acme" }),
}));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/use-profile-image", () => import("@/lib/fake-profile-image"));
vi.mock("@/hooks/store/use-issue-detail", async (original) => ({
  ...(await original<typeof import("@/hooks/store/use-issue-detail")>()),
  useIssueDetail: () => tab.app.issue.issueDetail,
}));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => tab.app.state }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => tab.app.memberRoot }));
vi.mock("@/hooks/store/use-label", () => ({ useLabel: () => tab.app.label }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => tab.app.workspaceRoot }));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => tab.app.user,
  useUserPermissions: () => tab.app.user.permission,
  useUserProfile: () => tab.app.user.userProfile,
}));
vi.mock("@/hooks/editor/use-issue-images", () => ({ useIssueImages: () => ({ images: undefined }) }));
vi.mock("@/components/issues/title-input", () => ({
  IssueTitleInput: (props: ComponentProps<typeof IssueTitleInput>) => (
    <input aria-label="title" disabled={props.disabled} />
  ),
}));
vi.mock("@/components/editor/rich-text/description-input", () => ({
  DescriptionInput: () => <div data-description />,
}));
vi.mock("@/components/issues/issue-detail/reactions", () => ({ IssueReaction: () => <div data-reactions /> }));
vi.mock("@/components/issues/issue-detail/issue-activity", () => ({ IssueActivity: () => <div data-activity /> }));
vi.mock("@/components/core/description-versions", () => ({
  DescriptionVersionsRoot: (props: ComponentProps<typeof DescriptionVersionsRoot>) => (
    <div data-versions={String(props.entityInformation.isRestoreDisabled)} />
  ),
}));
vi.mock("./duplicate-of", () => ({
  IntakeDuplicateOf: (props: ComponentProps<typeof IntakeDuplicateOf>) => (
    <p data-duplicate={String(props.duplicateToId)} data-where={`${props.workspaceSlug} ${props.projectId}`} />
  ),
}));

let tab: Awaited<ReturnType<typeof intakeTab>>;
let host: HTMLDivElement, root: Root;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers();
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  tab = await intakeTab();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  tab.app.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

const ADMIN = 20 as const;
const MEMBER = 15 as const;
const GUEST = 5 as const;
type Status = IssueIntake["status"];
const intakeOf = (status: Status, duplicateToId: string | null = null): IssueIntake => ({
  status,
  snoozed_till: status === "snoozed" ? "2099-01-01T00:00:00Z" : null,
  duplicate_to_id: duplicateToId,
});

/** The caller's role in WEB, the project's own fields, and the work item i-x of its intake, submitted by creator. */
async function shown(
  role: Project["member_role"],
  intake: IssueIntake,
  creator = "u-other",
  project: Partial<Project> = {}
) {
  const web = projectOf("WEB", acme.id, { intake_view: true, member_role: role, ...project });
  const store = tab.app.projectRoot.project;
  if (web.archived_at) await loadArchivedProjects(tab.nerve, store, acme, [web]);
  else await loadProjects(tab.nerve, store, acme, [web]);
  tab.nerve.calls.length = 0;
  tab.app.issue.issues.acceptRead(detailOf("x", { intake, created_by_id: creator }), tab.app.issue.captureRead());
  await act(async () =>
    root.render(
      <IntakeDetail
        workspaceSlug="acme"
        projectId="p-web"
        issueId="i-x"
        isSubmitting="saved"
        setTitleStatus={() => {}}
        setDescriptionStatus={() => {}}
      />
    )
  );
}

/** The properties, each by its label's key, as the detail draws them. */
const PROPERTIES = {
  state: "common.state",
  assignees: "common.assignees",
  priority: "common.priority",
  due_date: "common.order_by.due_date",
  labels: "common.labels",
} as const;
type Property = keyof typeof PROPERTIES;
/** The row of a property: its label, then its control. */
const row = (property: Property) =>
  [...host.querySelectorAll("span")].find((span) => span.textContent === PROPERTIES[property])?.parentElement
    ?.parentElement ?? undefined;
/** The properties drawn, in their order. */
const drawn = () => (Object.keys(PROPERTIES) as Property[]).filter((property) => row(property) !== undefined);
/** The properties the caller may change: those with a control he may press. */
const changeable = () =>
  drawn().filter((property) => [...row(property)!.querySelectorAll("button")].some((button) => !button.disabled));

const ALL: Property[] = ["assignees", "priority", "due_date", "labels"];
const WHO = [
  { who: "an admin", role: ADMIN, mine: false, changes: true },
  { who: "a member who submitted it", role: MEMBER, mine: true, changes: true },
  // a guest who submitted it changes its name and description alone (R15)
  { who: "a guest who submitted it", role: GUEST, mine: true, changes: false },
  { who: "a member", role: MEMBER, mine: false, changes: false },
] as const;
const STATUSES = ["pending", "snoozed", "accepted", "declined", "duplicate"] as const;
const MATRIX = WHO.flatMap(({ who, role, mine, changes }) =>
  STATUSES.map((status) => ({
    who,
    role,
    mine,
    status,
    // in the intake alone; triaged, it is read-only
    changes: changes && (status === "pending" || status === "snoozed") ? ALL : [],
    // the state once it is accepted alone
    drawn: status === "accepted" ? ["state", ...ALL] : ALL,
  }))
);

describe("InboxIssueContentProperties, by the caller's role and the work item's status (spec 2.7, 2.9)", () => {
  it.each(MATRIX)("lets $who, of a work item $status, change $changes", async (cell) => {
    await shown(cell.role, intakeOf(cell.status), cell.mine ? "u-me" : "u-other");
    expect(drawn()).toEqual(cell.drawn);
    expect(changeable()).toEqual(cell.changes);
  });

  // an archived project's intake is read-only (R27)
  it.each([
    { who: "an admin", role: ADMIN, mine: false },
    { who: "a member who submitted it", role: MEMBER, mine: true },
  ])("lets $who of an archived project change none, of a pending work item", async ({ role, mine }) => {
    await shown(role, intakeOf("pending"), mine ? "u-me" : "u-other", { archived_at: "2026-11-01T00:00:00Z" });
    expect(drawn()).toEqual(ALL);
    expect(changeable()).toEqual([]);
  });

  it("opens the assignees' list from its button by the keyboard, and reads the project's members", async () => {
    await shown(ADMIN, intakeOf("pending"));
    const button = row("assignees")!.querySelector<HTMLButtonElement>("button")!;
    await act(async () => {
      button.focus();
      button.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true }));
      await until(() => tab.nerve.calls.length === 1, "the members' read");
    });
    expect(tab.nerve.calls.map((call) => [call.method, call.path])).toEqual([
      ["GET", "/api/v0/projects/p-web/members"],
    ]);
    await act(async () => {
      tab.nerve.calls[0]!.answer(json(200, { data: [] }));
      await vi.advanceTimersByTimeAsync(20);
    });
    const search = document.querySelector<HTMLInputElement>('input[role="combobox"]');
    expect(search).not.toBeNull();
    expect(document.activeElement).toBe(search);
  });
});

describe("InboxIssueContentProperties, the work item it duplicates (spec 2.9)", () => {
  it.each([
    { what: "one marked a duplicate of i-t", intake: intakeOf("duplicate", "i-t"), id: "i-t" },
    { what: "one marked a duplicate of a work item since gone", intake: intakeOf("duplicate"), id: "null" },
    { what: "one pending", intake: intakeOf("pending"), id: "null" },
  ])("draws the row of the work item duplicated, by its id, for $what", async ({ intake, id }) => {
    await shown(ADMIN, intake);
    const duplicate = host.querySelector<HTMLElement>("[data-duplicate]");
    expect(duplicate?.dataset.duplicate).toBe(id);
    expect(duplicate?.dataset.where).toBe("acme p-web");
  });
});
