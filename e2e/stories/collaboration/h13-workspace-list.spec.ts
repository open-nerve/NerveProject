import {
  addProjectMembers,
  archiveProject,
  changeProject,
  changeRole,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMembershipOf,
  projectMemberWrites,
  slugFor,
} from "../../fixtures/api";
import { expectIssuePage } from "../../fixtures/assert/issue";
import { login, newAccount } from "../../fixtures/auth";
import type { Database } from "../../fixtures/db";
import { submitIntakeIssue } from "../../fixtures/intake";
import { validationFailed } from "../../fixtures/issue-planning";
import {
  createIssue,
  issueArchived,
  issueSubscribed,
  listIssues,
  listWorkspaceIssues,
  workspaceIssuesListed,
  type Issue,
  type IssueCreate,
  type WorkspaceIssueQuery,
} from "../../fixtures/issues";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

// H13, the workspace's list of work items across its projects (M7 design 2, 3.7; P3a spec 2.2, 2.3), its list half:
// the API version. A member of a workspace lists the work items of each of its projects he is an active member of,
// unarchived, each as that project's own list shows them to him: where he is a guest who sees only his own, his own. A
// project he is no member of, public or not, even as the workspace's admin, an archived one and one he left add none;
// nor do the work items in an intake's triage, nor archived ones. The four default views are the list with no filter,
// with assignee_id=me, created_by_id=me and subscriber_id=me. A cursor pages across the projects, each page with the
// list's total. The list is flat and has no cycles: a grouping or a cycle filter is refused. A personal access token
// and a browser session answer alike. Each filter, a cursor made at a row he does not see, the refusals' order and the
// read snapshot are the bootstrap tests' (TestWorkspaceIssueList) and the permission matrix's; the page is P6b's.

/**
 * A work item of a workspace as its list judges it for one account (workItemsFor): whether its project is in his
 * scope, one he is an active member of, neither archived nor deleted; whether he sees only his own there, its guest
 * not shown every work item; whether it may be listed at all, neither archived nor deleted nor in its intake's triage;
 * whether he made it, is assigned to it and subscribes to it.
 */
interface Judged {
  id: string;
  name: string;
  scope: boolean;
  own_only: boolean;
  listed: boolean;
  his: boolean;
  assigned: boolean;
  subscribed: boolean;
}

/**
 * Every work item of the workspace of slug, the archived and the deleted ones too, judged for the account of
 * accountId, in the list's order (-created_at, ties by id).
 */
async function workItemsFor(db: Database, slug: string, accountId: string): Promise<Judged[]> {
  return db.query<Judged>(
    `SELECT i.id::text AS id, i.name,
            pm.id IS NOT NULL AND p.archived_at IS NULL AND p.deleted_at IS NULL AS scope,
            coalesce(pm.role = 5 AND NOT p.guest_view_all_features, false) AS own_only,
            i.archived_at IS NULL AND i.deleted_at IS NULL AND s."group" <> 'triage' AS listed,
            i.created_by_id IS NOT DISTINCT FROM $2::uuid AS his,
            EXISTS (SELECT 1 FROM issue_assignees a
                     WHERE a.issue_id = i.id AND a.assignee_id = $2 AND a.deleted_at IS NULL) AS assigned,
            EXISTS (SELECT 1 FROM issue_subscribers f
                     WHERE f.issue_id = i.id AND f.subscriber_id = $2 AND f.deleted_at IS NULL) AS subscribed
       FROM issues i
       JOIN workspaces w ON w.id = i.workspace_id
       JOIN projects p ON p.id = i.project_id
       JOIN states s ON s.id = i.state_id
       LEFT JOIN project_members pm
         ON pm.project_id = p.id AND pm.member_id = $2 AND pm.is_active AND pm.deleted_at IS NULL
      WHERE w.slug = $1
      ORDER BY i.created_at DESC, i.id DESC`,
    [slug, accountId]
  );
}

/** Whether the workspace's list shows the work item judged to its account (M7 P3a spec 2.2). */
function shown(item: Judged): boolean {
  return item.scope && item.listed && (!item.own_only || item.his);
}

/** The names of items, in their order. */
function names(items: { name: string }[]): string[] {
  return items.map((item) => item.name);
}

/** The memberships of a project that make each of accounts its member (15). */
function members(...accounts: { id: string }[]) {
  return accounts.map(({ id }) => ({ member_id: id, role: 15 as const }));
}

test("H13 (API): walt, Alpha's member and Web's guest who sees only his own, lists the work items of both, across them, Web's his own alone; Open, public, Shelf, archived, and Former, which he left, add none, nor do Alpha's triage and archived work items; the four default views, each as the database says it should be, a cursor across the projects with the total, the list of each project as its own; made the workspace's admin, he lists the same; a grouping and a cycle refused; a personal access token and a browser session alike", async ({
  api,
  db,
}, testInfo) => {
  const [alice, walt, bob] = await Promise.all([
    newAccount(api, testInfo, "alice"),
    newAccount(api, testInfo, "walt"),
    newAccount(api, testInfo, "bob"),
  ]);
  const slug = slugFor(testInfo);
  await createWorkspace(api, alice.token, { name: "Acme", slug });
  await inviteAndAccept(api, alice.token, slug, walt, 15);
  await inviteAndAccept(api, alice.token, slug, bob, 15);
  // alice's projects: Alpha, private, with its intake on, and Web, public, walt's and bob's; Open, public, of neither;
  // Shelf and Former, walt's.
  const [alpha, web, open, shelf, former] = await Promise.all(
    (
      [
        ["Alpha", "ALP", 0],
        ["Web", "WEB", 2],
        ["Open", "OPN", 2],
        ["Shelf", "SHF", 0],
        ["Former", "FOR", 0],
      ] as const
    ).map(([name, identifier, network]) => createProject(api, alice.token, slug, { name, identifier, network }))
  );
  if (!alpha || !web || !open || !shelf || !former) throw new Error("missing projects");
  await Promise.all([
    addProjectMembers(api, alice.token, alpha.id, members(walt, bob)),
    addProjectMembers(api, alice.token, web.id, members(walt, bob)),
    addProjectMembers(api, alice.token, shelf.id, members(walt)),
    addProjectMembers(api, alice.token, former.id, members(walt)),
  ]);
  expect(await changeProject(api, alice.token, alpha.id, { intake_view: true }), "Alpha's intake on").toMatchObject({
    status: 200,
  });
  const { Done: done } = await stateIds(api, alice.token, alpha.id);

  // One after another, so that walt's rows alternate between Alpha and Web by -created_at and Web's rows not his fall
  // between them. Who makes a work item or is assigned to it subscribes to it.
  const make = (who: { token: string }, project: { id: string }, body: IssueCreate) =>
    createIssue(api, who.token, project.id, body);
  const a1 = await make(alice, alpha, { name: "Alpha plain" });
  const w1 = await make(alice, web, { name: "Web hidden", assignee_ids: [walt.id] });
  const w2 = await make(walt, web, { name: "Web own", assignee_ids: [walt.id] });
  const a2 = await make(bob, alpha, { name: "Alpha assigned", assignee_ids: [walt.id] });
  const w4 = await make(bob, web, { name: "Web bob" });
  const w3 = await make(walt, web, { name: "Web child", parent_id: w1.id });
  const a3 = await make(walt, alpha, { name: "Alpha own" });
  const shelved = await make(walt, shelf, { name: "Shelved", assignee_ids: [walt.id] });
  const left = await make(walt, former, { name: "Left", assignee_ids: [walt.id] });
  const outside = await make(alice, open, { name: "Open" });
  const a4 = await make(walt, alpha, { name: "Alpha archived", state_id: done, assignee_ids: [walt.id] });
  expect(await issueArchived(api, walt.token, a4.id), "Alpha archived, archived").toMatchObject({ status: 200 });
  const a6 = await submitIntakeIssue(api, walt.token, alpha.id, { name: "Alpha triage" });
  const a7 = await make(bob, alpha, { name: "Alpha bob", assignee_ids: [bob.id] });
  expect(await issueSubscribed(api, walt.token, a1.id), "walt subscribes to Alpha plain").toEqual({ status: 204 });
  // walt becomes Web's guest, who sees only his own; Shelf is archived; walt leaves Former.
  const webs = projectMemberWrites(api, web.id);
  expect(await webs.change(alice.token, await projectMembershipOf(api, alice.token, web.id, walt.id), 5)).toEqual({
    status: 200,
  });
  await archiveProject(api, alice.token, shelf.id);
  expect(await projectMemberWrites(api, former.id).leave(walt.token), "walt leaves Former").toEqual({ status: 204 });
  // newAccount's token is walt's personal access token; his browser session besides.
  const session = (await login(api, walt.email)).access_token;

  // The four default views. The database says which work items of the workspace each should show him, and which of
  // those it would take in it should not: not his in Web, of Open, Shelf or Former, archived, or in Alpha's triage.
  // Each view answers exactly the first, in its order, alike with either credential.
  const judged = await workItemsFor(db, slug, walt.id);
  const views = [
    ["all", {}, () => true, [a7, a3, w3, a2, w2, a1], [a6, a4, outside, left, shelved, w4, w1]],
    ["assigned to him", { assignee_id: ["me"] }, (i) => i.assigned, [a2, w2], [a4, left, shelved, w1]],
    ["made by him", { created_by_id: ["me"] }, (i) => i.his, [a3, w3, w2], [a6, a4, left, shelved]],
    [
      "subscribed to by him",
      { subscriber_id: ["me"] },
      (i) => i.subscribed,
      [a3, w3, a2, w2, a1],
      [a6, a4, left, shelved, w1],
    ],
  ] satisfies [string, WorkspaceIssueQuery, (item: Judged) => boolean, Issue[], Issue[]][];
  await views.reduce(async (before, [view, query, of, want, unseen]) => {
    await before;
    await test.step(`the view ${view}`, async () => {
      const should = judged.filter((i) => of(i) && shown(i));
      expect(names(should), `${view}: what the database says he sees`).toEqual(names(want));
      expect(names(judged.filter((i) => of(i) && !shown(i))), `${view}: what it says he does not`).toEqual(
        names(unseen)
      );
      const page = await listWorkspaceIssues(api, walt.token, slug, query);
      expectIssuePage(
        page,
        should.map((i) => i.id)
      );
      expect(await listWorkspaceIssues(api, session, slug, query), `${view}, with his browser session`).toEqual(page);
    });
  }, Promise.resolve());

  // By project: Alpha's, of which he is a member, bob's and alice's too; Web's, whose guest he is, his own, Web child's
  // parent, Web hidden, not named. Each is the project's own list as he reads it, row for row. Open, Shelf and Former
  // add none.
  const alphas = await listWorkspaceIssues(api, walt.token, slug, { project_id: [alpha.id] });
  expectIssuePage(alphas, [a7.id, a3.id, a2.id, a1.id]);
  expect(alphas, "Alpha's part, as Alpha's list").toEqual(await listIssues(api, walt.token, alpha.id));
  const webRows = await listWorkspaceIssues(api, session, slug, { project_id: [web.id] });
  expectIssuePage(webRows, [w3.id, w2.id]);
  expect(webRows.data?.[0]?.parent_id, "Web child's parent").toBeNull();
  expect(webRows, "Web's part, as Web's list").toEqual(await listIssues(api, session, web.id));
  expectIssuePage(await listWorkspaceIssues(api, walt.token, slug, { project_id: [open.id, shelf.id, former.id] }), []);

  // Two a page, with his browser session: three pages across Alpha and Web, each with the list's six.
  const first = await listWorkspaceIssues(api, session, slug, { limit: 2 });
  expectIssuePage(first, [a7.id, a3.id], 6, true);
  const second = await listWorkspaceIssues(api, session, slug, { limit: 2, cursor: first.next_cursor ?? undefined });
  expectIssuePage(second, [w3.id, a2.id], 6, true);
  const last = await listWorkspaceIssues(api, session, slug, { limit: 2, cursor: second.next_cursor ?? undefined });
  expectIssuePage(last, [w2.id, a1.id], 6);

  // Made the workspace's admin, he lists the same: a role in the workspace adds no project he is no member of.
  const everything = await listWorkspaceIssues(api, walt.token, slug);
  await changeRole(api, alice.token, slug, walt.id, 20);
  expect(await listWorkspaceIssues(api, session, slug), "walt's list as the workspace's admin").toEqual(everything);
  expectIssuePage(await listWorkspaceIssues(api, walt.token, slug, { project_id: [open.id] }), []);

  // Flat, without a project's cycles and modules: a grouping and a cycle filter are refused, with either credential.
  expect(
    [
      // @ts-expect-error -- the workspace's list declares no grouping; the real endpoint refuses it
      await workspaceIssuesListed(api, walt.token, slug, { group_by: "project_id" }),
      // @ts-expect-error -- nor a cycle filter
      await workspaceIssuesListed(api, session, slug, { cycle_id: ["none"] }),
    ],
    "walt's list grouped by project and filtered by no cycle"
  ).toEqual([validationFailed(["group_by", "not_allowed"]), validationFailed(["cycle_id", "not_allowed"])]);
});
