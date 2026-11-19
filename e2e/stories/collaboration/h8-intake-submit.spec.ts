import {
  addProjectMembers,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { expectIssueCollection, expectIssuePage } from "../../fixtures/assert/issue";
import { login, named, newAccount } from "../../fixtures/auth";
import type { Database } from "../../fixtures/db";
import { intakeIssueSubmitted, submitIntakeIssue } from "../../fixtures/intake";
import { createIssue, issueRead, listIssues, listMemberIssues, searchIssues, type Issue } from "../../fixtures/issues";
import { notificationsSent, type Notification } from "../../fixtures/notifications";
import { expect, test } from "../../fixtures/test";

// H8, submitting a work item to a project's intake (M7 design 2, 3.5; P2a spec 2.4, 2.6): the API version. While its
// admin has the project's intake turned on, its members and guests submit work items to it, each answered 201 with the
// work item in the project's triage state, its intake pending, an intake row of its submitter's. Such a work item is
// out of the project's list, the workspace's lists by member and the searches, and reads by its id as any other; its
// assignees and those it mentions are sent its creation, as of any other, showing it in Triage. Turned off, the intake
// refuses a submission (issue.intake_disabled), and the work items submitted read as they were. The other roles, the
// bodies refused, an archived project, the locks and the interleavings are the bootstrap tests' and the permission
// matrix's.

/**
 * The intake rows of the project of projectId, the deleted ones too, in the order they were made: its work item's name
 * and state, by name and group; whether it is live; its status, snooze and duplicate; who made it, by id, and whether
 * he wrote it last; and whether it is of its work item's workspace and project, made and last written at its work
 * item's creation.
 */
async function intakeRows(db: Database, projectId: string) {
  return db.query<{
    name: string;
    state: string;
    state_group: string;
    live: boolean;
    status: string;
    snoozed_till: string | null;
    duplicate_to_id: string | null;
    submitter: string;
    stamped: boolean;
  }>(
    `SELECT i.name, s.name AS state, s."group" AS state_group, ii.deleted_at IS NULL AS live, ii.status,
            ii.snoozed_till, ii.duplicate_to_id::text AS duplicate_to_id, ii.created_by_id::text AS submitter,
            ii.updated_by_id = ii.created_by_id
              AND ii.workspace_id = i.workspace_id AND ii.project_id = i.project_id
              AND ii.created_at = i.created_at AND ii.updated_at = i.created_at AS stamped
       FROM intake_issues ii JOIN issues i ON i.id = ii.issue_id JOIN states s ON s.id = i.state_id
      WHERE ii.project_id = $1 ORDER BY ii.created_at, ii.id`,
    [projectId]
  );
}

/** The key of a notification among one receiver's in this story: its work item and the field it changed. */
function subject(note: { issue_id: string; data: { change: { field: string } } }): string {
  return `${note.issue_id} ${note.data.change.field}`;
}

/** notes in the order of their subjects, to compare a receiver's notifications whatever their order. */
function bySubject<T extends Parameters<typeof subject>[0]>(notes: T[]): T[] {
  return notes.toSorted((a, b) => subject(a).localeCompare(subject(b)));
}

const pending = { status: "pending", snoozed_till: null, duplicate_to_id: null };

test("H8 (API): with Web's intake turned on by its admin, a member submits a work item with his personal access token and a guest with her browser session, each answered 201 in Web's triage state, its intake pending, an intake row of the submitter's; neither is in Web's list, the assignee's list or a search, each reads by its id; the assignee is sent each one's creation and his assignment, whom it mentions her mention, each showing it in Triage; turned off, the intake refuses a submission, and the two read as they were", async ({
  api,
  db,
}, testInfo) => {
  const [alice, bob, gus, walt] = await Promise.all([
    newAccount(api, testInfo, "alice"),
    newAccount(api, testInfo, "bob"),
    newAccount(api, testInfo, "gus"),
    newAccount(api, testInfo, "walt"),
  ]);
  await named(api, walt.token, { display_name: "Walt" });
  const slug = slugFor(testInfo);
  const acme = await createWorkspace(api, alice.token, { name: "Acme", slug });
  await inviteAndAccept(api, alice.token, slug, bob, 15);
  await inviteAndAccept(api, alice.token, slug, gus, 5);
  await inviteAndAccept(api, alice.token, slug, walt, 15);
  const web = await createProject(api, alice.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, alice.token, web.id, [
    { member_id: bob.id, role: 15 },
    { member_id: gus.id, role: 5 },
    { member_id: walt.id, role: 15 },
  ]);
  // alice's Fix the login, assigned to walt, is in Web's list, walt's assigned work items and a search for login.
  const fix = await createIssue(api, alice.token, web.id, { name: "Fix the login", assignee_ids: [walt.id] });
  const intakeOn = await changeProject(api, alice.token, web.id, { intake_view: true });
  expect(intakeOn, "Web's intake turned on by alice").toMatchObject({ status: 200, project: { intake_view: true } });
  const pat = { what: "his personal access token, a member's", who: bob, token: bob.token, name: "Login breaks" };
  const session = {
    what: "her browser session, a guest's",
    who: gus,
    token: (await login(api, gus.email)).access_token,
    name: "Login is slow",
  };
  const mention = `<p>See</p><mention-component entity_name="user_mention" entity_identifier="${alice.id}"></mention-component>`;

  // With the credential given, its holder submits a work item for walt that mentions alice: hers, in Triage, pending.
  const submits = async ({ what, who, token, name }: typeof pat) => {
    const issue = await submitIntakeIssue(api, token, web.id, {
      name,
      assignee_ids: [walt.id],
      description_html: mention,
    });
    expect(issue, `${name} submitted with ${what}`).toMatchObject({
      name,
      project_id: web.id,
      parent_id: null,
      created_by_id: who.id,
      assignee_ids: [walt.id],
      intake: pending,
    });
    expect(await issueRead(api, token, issue.id), `${name} read with ${what}`).toMatchObject({
      status: 200,
      issue: { id: issue.id, state_id: issue.state_id, intake: pending },
    });
    return issue;
  };
  const breaks = await submits(pat);
  const slow = await submits(session);
  expect(slow.state_id, "both in one state, Web's triage state").toBe(breaks.state_id);
  expect(slow.state_id, "not Fix's, Web's default").not.toBe(fix.state_id);

  // Neither is in Web's list, in walt's assigned work items, nor in a search, of Web or for login.
  expectIssuePage(await listIssues(api, alice.token, web.id), [fix.id]);
  expectIssuePage(await listMemberIssues(api, alice.token, slug, walt.id, { relation: "assigned" }), [fix.id]);
  expectIssueCollection(await searchIssues(api, alice.token, slug, { project_id: web.id }), [fix.id]);
  expectIssueCollection(await searchIssues(api, alice.token, slug, { q: "login" }), [fix.id]);

  // walt is sent each one's creation and his assignment, alice her mention, in her mentions tab: each by its submitter,
  // showing the work item in Triage, triage.
  const sent = (
    sender: Notification["sender"],
    item: Issue,
    change: { field: string; verb: string; [key: string]: unknown }
  ) => ({
    id: expect.any(String),
    workspace_id: acme.id,
    project_id: web.id,
    issue_id: item.id,
    sender,
    triggered_by_id: item.created_by_id,
    data: {
      issue: {
        id: item.id,
        name: item.name,
        sequence_id: item.sequence_id,
        identifier: "WEB",
        state_name: "Triage",
        state_group: "triage",
      },
      change: {
        old_value: null,
        new_value: null,
        old_id: null,
        new_id: null,
        old_issue: null,
        new_issue: null,
        comment_id: null,
        excerpt: null,
        ...change,
      },
    },
    read_at: null,
    archived_at: null,
    snoozed_till: null,
    created_at: expect.any(String),
  });
  const created = { field: "issue", verb: "created" };
  const assignment = { field: "assignees", verb: "created", new_value: "Walt", new_id: walt.id };
  const mentioned = { field: "description", verb: "mentioned" };
  // Fix's creation and walt's assignment to it, two of walt's, came before.
  const [walts, alices] = await Promise.all([
    notificationsSent(api, walt.token, slug, 6),
    notificationsSent(api, alice.token, slug, 2, { tab: "mentions" }),
  ]);
  expect(bySubject(walts.filter((n) => n.issue_id !== fix.id)), "walt's all tab but Fix").toEqual(
    bySubject(
      [breaks, slow].flatMap((issue) => [sent("assigned", issue, created), sent("assigned", issue, assignment)])
    )
  );
  expect(bySubject(alices), "alice's mentions tab").toEqual(
    bySubject([breaks, slow].map((issue) => sent("mentioned", issue, mentioned)))
  );

  // Turned off, the intake refuses a submission with either credential, and each work item submitted reads as it was.
  const intakeOff = await changeProject(api, alice.token, web.id, { intake_view: false });
  expect(intakeOff, "Web's intake turned off by alice").toMatchObject({ status: 200, project: { intake_view: false } });
  const [refusedPAT, refusedSession] = await Promise.all(
    [pat, session].map(({ token }) => intakeIssueSubmitted(api, token, web.id, { name: "Too late" }))
  );
  const disabled = { status: 409, code: "issue.intake_disabled" };
  expect(refusedPAT, `a submission with ${pat.what}, the intake off`).toEqual(disabled);
  expect(refusedSession, `a submission with ${session.what}, the intake off`).toEqual(disabled);
  const [breaksRead, slowRead] = await Promise.all([
    issueRead(api, pat.token, breaks.id),
    issueRead(api, session.token, slow.id),
  ]);
  expect(breaksRead, `${breaks.name} read with ${pat.what}, the intake off`).toEqual({ status: 200, issue: breaks });
  expect(slowRead, `${slow.name} read with ${session.what}, the intake off`).toEqual({ status: 200, issue: slow });

  // One intake row each, live, pending, its submitter's, at its creation; its work item in Triage. The refused wrote none.
  const row = { state: "Triage", state_group: "triage", live: true, status: "pending", stamped: true };
  expect(await intakeRows(db, web.id), "Web's intake rows").toEqual([
    { ...row, name: breaks.name, snoozed_till: null, duplicate_to_id: null, submitter: bob.id },
    { ...row, name: slow.name, snoozed_till: null, duplicate_to_id: null, submitter: gus.id },
  ]);
});
