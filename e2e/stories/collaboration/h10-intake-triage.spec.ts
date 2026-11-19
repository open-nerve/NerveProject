import {
  addProjectMembers,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { expectIssueCollection, expectIssuePage } from "../../fixtures/assert/issue";
import { login, newAccount } from "../../fixtures/auth";
import { createIssueComment } from "../../fixtures/comments";
import { intakeIssueTriaged, submitIntakeIssue, triageIntakeIssue, type IntakeTriage } from "../../fixtures/intake";
import {
  activityPage,
  createIssue,
  issueRead,
  issueUpdated,
  listIssues,
  searchIssues,
  type Issue,
} from "../../fixtures/issues";
import { notificationsSent } from "../../fixtures/notifications";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

// H10, triaging the work items of a project's intake (M7 design 2, 3.5; P2a spec 2.5, 2.6): the API version. The
// project's admin snoozes a work item submitted to its intake till a time to come, ends the snooze, and declines it,
// or marks it a duplicate of a work item she sees, alike with a personal access token and her browser session: each
// triage answers the work item, in Triage still, with its intake so; a final status moves no more
// (issue.intake_transition_not_allowed). A member may not triage (forbidden). Pending, a work item takes a comment but
// no change (issue.in_triage). Accepted, it moves to the project's default state, into the project's list and
// searches, its activity saying its intake moved, then its state, at one moment; its submitter is sent its new state.
// A snooze's end by itself, the other transitions and refusals, their order, the locks and the interleavings are the
// bootstrap tests' and the permission matrix's.

/** A work item's intake as its answer gives it, its snooze's time as an instant in the form toISOString writes. */
function intakeOf(issue: Issue) {
  const till = issue.intake?.snoozed_till;
  return issue.intake && { ...issue.intake, snoozed_till: till ? new Date(till).toISOString() : null };
}

const pending = { status: "pending", snoozed_till: null, duplicate_to_id: null };
const notAllowed = { status: 409, code: "issue.intake_transition_not_allowed" };

test("H10 (API): Web's admin snoozes a work item of its intake, ends the snooze and declines it, and another the same but marks it a duplicate, one with her personal access token, the other with her browser session; a final status moves no more; a member may not triage; pending, a work item takes a comment but no change; accepted, it is in Web's default state, its list and searches, its activity says its intake moved before its state, and its submitter is sent its new state", async ({
  api,
}, testInfo) => {
  const [alice, bob] = await Promise.all([newAccount(api, testInfo, "alice"), newAccount(api, testInfo, "bob")]);
  const slug = slugFor(testInfo);
  const acme = await createWorkspace(api, alice.token, { name: "Acme", slug });
  await inviteAndAccept(api, alice.token, slug, bob, 15);
  const web = await createProject(api, alice.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, alice.token, web.id, [{ member_id: bob.id, role: 15 }]);
  const intakeOn = await changeProject(api, alice.token, web.id, { intake_view: true });
  expect(intakeOn, "Web's intake turned on by alice").toMatchObject({ status: 200, project: { intake_view: true } });
  // alice's Fix the login, which a duplicate names; bob's three submissions, each in Triage, pending.
  const fix = await createIssue(api, alice.token, web.id, { name: "Fix the login" });
  const spam = await submitIntakeIssue(api, bob.token, web.id, { name: "Buy now" });
  const again = await submitIntakeIssue(api, bob.token, web.id, { name: "The login fails" });
  const dark = await submitIntakeIssue(api, bob.token, web.id, { name: "Dark mode" });
  const { Backlog: backlog } = await stateIds(api, alice.token, web.id);
  const pat = { what: "her personal access token", token: alice.token };
  const session = { what: "her browser session", token: (await login(api, alice.email)).access_token };

  // With the credential given, alice triages item as body says: it answers in Triage still, its intake as intake.
  const moves = async ({ what, token }: typeof pat, item: Issue, body: IntakeTriage, intake: object) => {
    const triaged = await triageIntakeIssue(api, token, item.id, body);
    expect(triaged, `${item.name} triaged ${body.status} with ${what}`).toMatchObject({
      id: item.id,
      state_id: item.state_id,
    });
    expect(intakeOf(triaged), `${item.name}'s intake triaged ${body.status} with ${what}`).toEqual(intake);
  };
  // With the credential given, alice snoozes item till an hour on, which no request takes to reach nerve, ends the
  // snooze, then triages it last, a final status; from it, an acceptance moves it no more.
  const round = async (credential: typeof pat, item: Issue, last: IntakeTriage, final: object) => {
    const till = new Date(Date.now() + 3_600_000).toISOString();
    await moves(
      credential,
      item,
      { status: "snoozed", snoozed_till: till },
      { ...pending, status: "snoozed", snoozed_till: till }
    );
    await moves(credential, item, { status: "pending" }, pending);
    await moves(credential, item, last, final);
    expect(
      await intakeIssueTriaged(api, credential.token, item.id, { status: "accepted" }),
      `${item.name} accepted once ${last.status}, with ${credential.what}`
    ).toEqual(notAllowed);
  };
  await round(pat, spam, { status: "declined" }, { ...pending, status: "declined" });
  await round(
    session,
    again,
    { status: "duplicate", duplicate_to_id: fix.id },
    {
      ...pending,
      status: "duplicate",
      duplicate_to_id: fix.id,
    }
  );

  // bob, a member, may not triage Dark mode; pending, it takes his comment, but alice may not change it.
  expect(await intakeIssueTriaged(api, bob.token, dark.id, { status: "accepted" }), "bob's acceptance").toEqual({
    status: 403,
    code: "forbidden",
  });
  await createIssueComment(api, bob.token, dark.id, "<p>On every page</p>");
  expect(await issueUpdated(api, alice.token, dark.id, { name: "Dark theme" }), "alice's change, pending").toEqual({
    status: 409,
    code: "issue.in_triage",
  });
  expect(await issueRead(api, alice.token, dark.id), "Dark mode, pending").toMatchObject({
    status: 200,
    issue: { name: "Dark mode", state_id: dark.state_id, intake: pending },
  });

  // Accepted, Dark mode is in Backlog, Web's default state: in Web's list, after Fix, and first in a search of Web,
  // the latest changed; the declined and the duplicate are in neither, in Triage still.
  const accepted = await triageIntakeIssue(api, pat.token, dark.id, { status: "accepted" });
  expect(accepted, "Dark mode accepted").toMatchObject({ id: dark.id, state_id: backlog, name: "Dark mode" });
  expect(intakeOf(accepted), "Dark mode's intake accepted").toEqual({ ...pending, status: "accepted" });
  const listed = await listIssues(api, alice.token, web.id, { order_by: "created_at" });
  expectIssuePage(listed, [fix.id, dark.id]);
  expect(listed.data?.[1]?.state_id, "Dark mode's state in Web's list").toBe(backlog);
  expectIssueCollection(await searchIssues(api, alice.token, slug, { project_id: web.id }), [dark.id, fix.id]);
  expect(
    await intakeIssueTriaged(api, session.token, dark.id, { status: "accepted" }),
    `Dark mode accepted again, with ${session.what}`
  ).toEqual(notAllowed);

  // Its activity: its submission, then alice's acceptance, its intake from pending to accepted naming nothing, then its
  // state from Triage to Backlog, both at one moment.
  const activity = (await activityPage(api, alice.token, dark.id, undefined, 100)).data;
  expect(
    activity.map(({ field, verb, old_value, new_value, old_identifier, new_identifier, actor_id }) => ({
      field,
      verb,
      old_value,
      new_value,
      old_identifier,
      new_identifier,
      actor_id,
    })),
    "Dark mode's activity"
  ).toEqual([
    {
      field: null,
      verb: "created",
      old_value: null,
      new_value: null,
      old_identifier: null,
      new_identifier: null,
      actor_id: bob.id,
    },
    {
      field: "intake",
      verb: "accepted",
      old_value: "pending",
      new_value: "accepted",
      old_identifier: null,
      new_identifier: null,
      actor_id: alice.id,
    },
    {
      field: "state",
      verb: "updated",
      old_value: "Triage",
      new_value: "Backlog",
      old_identifier: dark.state_id,
      new_identifier: backlog,
      actor_id: alice.id,
    },
  ]);
  expect(activity[2]?.created_at, "its state moved at its acceptance's moment").toBe(activity[1]?.created_at);

  // bob, who submitted it, is sent its new state by alice, as its creator; no triage before sent him anything.
  expect(await notificationsSent(api, bob.token, slug, 1), "bob's all tab").toEqual([
    {
      id: expect.any(String),
      workspace_id: acme.id,
      project_id: web.id,
      issue_id: dark.id,
      sender: "created",
      triggered_by_id: alice.id,
      data: {
        issue: {
          id: dark.id,
          name: "Dark mode",
          sequence_id: dark.sequence_id,
          identifier: "WEB",
          state_name: "Backlog",
          state_group: "backlog",
        },
        change: {
          field: "state",
          verb: "updated",
          old_value: "Triage",
          new_value: "Backlog",
          old_id: dark.state_id,
          new_id: backlog,
          old_issue: null,
          new_issue: null,
          comment_id: null,
          excerpt: null,
        },
      },
      read_at: null,
      archived_at: null,
      snoozed_till: null,
      created_at: expect.any(String),
    },
  ]);
});
