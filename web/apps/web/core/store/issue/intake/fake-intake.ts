/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The intake of the tests of the intake store and its readers (P5a spec 2.3), as nerve gives it, against a fake nerve:
// the counts, rows, pages and details nerve answers, the requests of the intake, the filters the intake page opens
// with, a tab whose caller (u-me) nerve listed acme and beta (and, for the writes, p-web), and the fetches that load a
// workspace's counts or a project's list into the store. Each export comes with the first test that reads it (plan,
// "knip 与第一个读者").

import type {
  IntakeCounts,
  IntakeIssue,
  IntakeIssuePage,
  IssueDetail,
  IssueListPage,
  Workspace,
} from "@nerve/api-client";
import { FakeNerve, answered, json } from "@/lib/auth/fake-nerve";
import type { ProjectIssueQuery } from "@/services/issue/issue-list.service";
import { issueOf, requestedIssue } from "@/store/issue/fake-issues";
import type { IssueListBinding } from "@/store/issue/helpers/issue-list-binding";
import type { IntakeFilters } from "@/store/issue/intake/filters";
import type { IIntakeStore } from "@/store/issue/intake/intake.store";
import { accountOf } from "@/store/member/fake-users";
import { seedAccount } from "@/store/user/fake-account";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

export const acme = workspaceOf("acme");
export const beta = workspaceOf("beta");

/**
 * The pending work items of a workspace's intakes as nerve counts them: one count for each project pending names, by
 * its id, in id order.
 */
export const countsOf = (pending: Record<string, number>): IntakeCounts => ({
  data: Object.entries(pending)
    .toSorted(([a], [b]) => a.localeCompare(b))
    .map(([projectId, count]) => ({ project_id: projectId, pending: count })),
});

/**
 * A work item of p-web's intake in acme (id-acme) as nerve lists it, its id and its name named by name: pending,
 * unless fields say so.
 */
export const intakeIssueOf = (name: string, fields: Partial<IntakeIssue> = {}): IntakeIssue => ({
  ...issueOf(`i-${name}`, "p-web", "id-acme"),
  name,
  intake: { status: "pending", snoozed_till: null, duplicate_to_id: null },
  ...fields,
});

/**
 * A work item of p-web's intake in acme as its detail answers it: its row (intakeIssueOf), with its HTML and the
 * caller's subscription, which a list's row does not carry, unless fields say otherwise.
 */
export const detailOf = (name: string, fields: Partial<IssueDetail> = {}): IssueDetail => ({
  ...intakeIssueOf(name),
  description_html: `<p>${name}</p>`,
  is_subscribed: false,
  ...fields,
});

/**
 * A page of a project's intake as nerve lists it: the last unless a cursor follows it; its total, the number its
 * filters select, the number of its rows unless given.
 */
export const pageOf = (
  data: IntakeIssue[],
  nextCursor: string | null = null,
  total = data.length
): IntakeIssuePage => ({
  total,
  data,
  next_cursor: nextCursor,
});

/** The request of a workspace's pending counts, by its slug. */
export const countsPath = (workspace: Pick<Workspace, "slug">) => `/api/v0/workspaces/${workspace.slug}/intake-counts`;
/** The request of a project's intake list, by its id. */
export const listPath = (projectId: string) => `/api/v0/projects/${projectId}/intake-issues`;

/** The open tab as the intake page opens it: the pending, newest first. */
export const open: IntakeFilters = {
  tab: "open",
  statuses: ["pending"],
  priorities: [],
  assignees: [],
  creators: [],
  labels: [],
  order: "-created_at",
};
/** The open tab with the snoozed too. */
export const snoozedToo: IntakeFilters = { ...open, statuses: ["pending", "snoozed"] };

/**
 * The intake store of a session whose caller, u-me, nerve listed acme and beta as his workspaces; the requests so far
 * forgotten. Its test mocks the tab's session (@/lib/auth/api-client, @/lib/store-context) before it calls this.
 */
export async function intakeTab() {
  const { RootStore } = await import("@/store/root.store");
  const nerve = new FakeNerve();
  const api = nerve.client();
  const app = new RootStore(api);
  await loadWorkspaces(nerve, app.workspaceRoot, [acme, beta]);
  await seedAccount(nerve, app.user, accountOf("me"));
  nerve.calls.length = 0;
  return { nerve, api, app, store: app.issue.intake };
}

/**
 * The intake store's tab (intakeTab) whose caller nerve also listed as an admin of p-web, in acme: the issue root's
 * writes reach the work items of p-web's intake, by its scope; the requests so far forgotten.
 */
export async function writesTab() {
  const { loadProjects, projectOf } = await import("@/store/project/fake-projects");
  const tab = await intakeTab();
  const web = projectOf("WEB", acme.id);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, acme, [web]);
  tab.nerve.calls.length = 0;
  return { ...tab, web, root: tab.app.issue, scope: tab.app.issue.scope(acme.slug, web.id) };
}

/** The tab of the writes of p-web's intake (writesTab). */
type WritesTab = Awaited<ReturnType<typeof writesTab>>;

/** The tab's issue root holds i-x as its detail (detailOf): in p-web's intake, pending, unless fields say otherwise. */
export function holdDetail(tab: WritesTab, fields: Partial<IssueDetail> = {}): IssueDetail {
  const detail = detailOf("x", fields);
  tab.root.issues.acceptRead(detail, tab.root.captureRead());
  return detail;
}

/** The tab's list of p-web's work items the query selects, as nerve gave page; the issue root holds it as each list. */
export async function loadProjectList(
  tab: WritesTab,
  query: ProjectIssueQuery,
  page: IssueListPage
): Promise<IssueListBinding> {
  const { IssueListBinding } = await import("@/store/issue/helpers/issue-list-binding");
  const list = new IssueListBinding(tab.root, {
    scope: "project",
    workspaceSlug: acme.slug,
    workspaceID: acme.id,
    projectID: tab.web.id,
    query: { order_by: "sort_order", ...query },
  });
  const read = await requestedIssue(tab.nerve, () => list.fetchFirst(), [
    "GET",
    `/api/v0/projects/${tab.web.id}/issues`,
  ]);
  read.call.answer(json(200, page));
  await read.promise;
  return list;
}

/** What a list of work items shows: its total and, by group, the group's total and its rows. */
export const shownList = (list: IssueListBinding) => ({
  total: list.value?.total,
  groups: [...(list.value?.groups ?? [])].map(([key, group]) => [key, group.total, group.rows.map((row) => row.id)]),
});

/** The store fetches the workspace's counts, and nerve gives counts. */
export function loadCounts(nerve: FakeNerve, store: IIntakeStore, workspace: Workspace, counts: IntakeCounts) {
  return answered(nerve, () => store.fetchCounts(workspace), ["GET", countsPath(workspace)], counts, "the counts");
}

/** The store fetches the first page of the project's intake the filters select, and nerve gives page. */
export function loadList(
  nerve: FakeNerve,
  store: IIntakeStore,
  projectId: string,
  filters: IntakeFilters,
  page: IntakeIssuePage
) {
  const fetch = () => store.fetchList(projectId, filters);
  return answered(nerve, fetch, ["GET", listPath(projectId)], page, "the list");
}
