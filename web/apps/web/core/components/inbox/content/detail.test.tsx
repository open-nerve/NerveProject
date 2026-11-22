// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { IssueIntake, Project } from "@nerve/api-client";
import { problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { acme, detailOf, intakeTab } from "@/store/issue/intake/fake-intake";
import { loadArchivedProjects, loadProjects, projectOf } from "@/store/project/fake-projects";
import type { DescriptionVersionsRoot } from "@/components/core/description-versions";
import type { DescriptionInput } from "@/components/editor/rich-text/description-input";
import type { IssueActivity } from "@/components/issues/issue-detail/issue-activity";
import type { IssueTitleInput } from "@/components/issues/title-input";
import { IntakeDetail } from "./detail";
import type { InboxIssueContentProperties } from "./issue-properties";

// The selected work item of a project's intake, inside the detail host (P5a spec 2.7, R15, R16, R27): who may change
// it (the project's admin or its submitter, while it is in the intake, pending or snoozed, the project not archived
// and its intake on; a guest submitter its name and description alone), and who reads its description versions
// (admins and members, IssueDescriptionVersions' own rule). Its controls are stand-ins that say whether they are
// disabled; the stores, the caller's role (the permission store's) and the versions' component are real.

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
vi.mock("@/hooks/store/use-issue-detail", async (original) => ({
  ...(await original<typeof import("@/hooks/store/use-issue-detail")>()),
  useIssueDetail: () => tab.app.issue.issueDetail,
}));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: () => undefined }) }));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => tab.app.user,
  useUserPermissions: () => tab.app.user.permission,
}));
vi.mock("@/hooks/editor/use-issue-images", () => ({ useIssueImages: () => ({ images: undefined }) }));
vi.mock("@/components/issues/title-input", () => ({
  IssueTitleInput: (props: ComponentProps<typeof IssueTitleInput>) => (
    <input aria-label="title" disabled={props.disabled} />
  ),
}));
vi.mock("@/components/editor/rich-text/description-input", () => ({
  DescriptionInput: (props: ComponentProps<typeof DescriptionInput>) => (
    <div data-description={String(!!props.disabled)} />
  ),
}));
vi.mock("@/components/issues/issue-detail/reactions", () => ({ IssueReaction: () => <div data-reactions /> }));
vi.mock("@/components/issues/issue-detail/issue-activity", () => ({
  IssueActivity: (props: ComponentProps<typeof IssueActivity>) => <div data-activity={String(!!props.isIntakeIssue)} />,
}));
vi.mock("./issue-properties", () => ({
  InboxIssueContentProperties: (props: ComponentProps<typeof InboxIssueContentProperties>) => (
    <div data-properties={String(props.isEditable)} />
  ),
}));
vi.mock("@/components/core/description-versions", () => ({
  DescriptionVersionsRoot: (props: ComponentProps<typeof DescriptionVersionsRoot>) => (
    <div data-versions={String(props.entityInformation.isRestoreDisabled)} />
  ),
}));

let tab: Awaited<ReturnType<typeof intakeTab>>;
let host: HTMLDivElement, root: Root;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await intakeTab();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  tab.app.dispose();
  vi.useRealTimers();
});

const ADMIN = 20 as const;
const MEMBER = 15 as const;
const GUEST = 5 as const;
const intakeOf = (status: IssueIntake["status"]): IssueIntake => ({
  status,
  snoozed_till: status === "snoozed" ? "2099-01-01T00:00:00Z" : null,
  duplicate_to_id: null,
});

/** The caller's role in WEB, the project's own fields, and the work item i-x of its intake, submitted by creator. */
async function shown(
  role: Project["member_role"],
  status: IssueIntake["status"],
  creator = "u-other",
  project: Partial<Project> = {}
) {
  const web = projectOf("WEB", acme.id, { intake_view: true, member_role: role, ...project });
  const store = tab.app.projectRoot.project;
  if (web.archived_at) await loadArchivedProjects(tab.nerve, store, acme, [web]);
  else await loadProjects(tab.nerve, store, acme, [web]);
  tab.nerve.calls.length = 0;
  tab.app.issue.issues.acceptRead(
    detailOf("x", { intake: intakeOf(status), created_by_id: creator }),
    tab.app.issue.captureRead()
  );
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
/** What the caller may change: its name, its description, its properties, and restore a version (if he reads them). */
const changes = () => ({
  name: !host.querySelector<HTMLInputElement>('input[aria-label="title"]')!.disabled,
  description: host.querySelector<HTMLElement>("[data-description]")!.dataset.description === "false",
  properties: host.querySelector<HTMLElement>("[data-properties]")!.dataset.properties === "true",
  restore: host.querySelector<HTMLElement>("[data-versions]")?.dataset.versions === "false",
});
const versions = () => host.querySelector("[data-versions]") !== null;
const ALL = { name: true, description: true, properties: true, restore: true };
const NONE = { name: false, description: false, properties: false, restore: false };

describe("IntakeDetail, who changes the work item (spec 2.7)", () => {
  it.each([
    { who: "an admin", role: ADMIN, status: "pending", changes: ALL },
    { who: "an admin", role: ADMIN, status: "snoozed", changes: ALL },
    { who: "a member who submitted it", role: MEMBER, mine: true, status: "pending", changes: ALL },
    { who: "a member who submitted it", role: MEMBER, mine: true, status: "snoozed", changes: ALL },
    // a guest submitter changes its name and description alone, and reads no versions
    {
      who: "a guest who submitted it",
      role: GUEST,
      mine: true,
      status: "pending",
      changes: { name: true, description: true, properties: false, restore: false },
    },
    { who: "a member", role: MEMBER, status: "pending", changes: NONE },
    { who: "an admin", role: ADMIN, status: "accepted", changes: NONE },
    { who: "an admin", role: ADMIN, status: "declined", changes: NONE },
    { who: "an admin", role: ADMIN, status: "duplicate", changes: NONE },
    { who: "a member who submitted it", role: MEMBER, mine: true, status: "declined", changes: NONE },
    { who: "an admin of an archived project", role: ADMIN, archived: true, status: "pending", changes: NONE },
    { who: "an admin of a project whose intake is off", role: ADMIN, off: true, status: "pending", changes: NONE },
  ] as const)("lets $who, of a work item $status, change $changes", async (cell) => {
    const project = {
      ...("archived" in cell ? { archived_at: "2026-11-01T00:00:00Z" } : {}),
      ...("off" in cell ? { intake_view: false } : {}),
    };
    await shown(cell.role, cell.status, "mine" in cell ? "u-me" : "u-other", project);
    expect(changes()).toEqual(cell.changes);
    expect(host.querySelector<HTMLElement>("[data-activity]")?.dataset.activity).toBe("true");
  });

  it.each([
    { who: "an admin", role: ADMIN, reads: true },
    { who: "a member", role: MEMBER, reads: true },
    { who: "a guest", role: GUEST, reads: false },
  ])("shows $who its description versions: $reads", async ({ role, reads }) => {
    await shown(role, "pending", "u-me");
    expect(versions()).toBe(reads);
  });

  it("says it is loading while the work item is read again, and why that read failed, with its retry", async () => {
    await shown(ADMIN, "pending");
    const again = tab.app.issue.issues.refreshAuthoritative("i-x").catch(() => undefined);
    await act(async () => until(() => tab.nerve.calls.length === 1, "the read again"));
    expect(host.querySelector('[role="status"]')?.textContent).toBe("common.loading");
    expect(host.querySelector('input[aria-label="title"]')).toBeNull();
    tab.nerve.calls[0]!.answer(problem(503, "service.unavailable"));
    await act(async () => void (await again));
    expect(host.querySelector('[role="alert"]')?.textContent).toContain("common.retry");
  });
});
