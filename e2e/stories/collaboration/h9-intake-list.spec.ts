import {
  addProjectMembers,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { expectIssuePage } from "../../fixtures/assert/issue";
import { login, newAccount } from "../../fixtures/auth";
import {
  intakeCounts,
  intakeIssueDeleted,
  intakeIssueUpdated,
  listIntakeIssues,
  submitIntakeIssue,
  triageIntakeIssue,
} from "../../fixtures/intake";
import { issueRead, listIssues } from "../../fixtures/issues";
import { expect, test } from "../../fixtures/test";

// H9, a project's intake listed and the pending work items of a workspace's intakes counted, with H11's change and
// deletion of a work item of an intake (M7 design 2, 3.5, 3.10; P2b spec 2.3–2.6): the API version. The project's
// admin, members and guests list its intake, every status, accepted work items too, filtered, in the order asked, a
// page at a time with the number the filters select; a guest who sees only his own work items lists his own. Each
// counts the pending work items of each intake of his, which a submission and a triage change. A work item's submitter
// changes it while it is pending, a guest its name and description alone; closed, it changes no more
// (issue.intake_closed). Deleted, a pending work item is gone; an accepted one stays, its intake row alone deleted,
// listed no more. A personal access token and a browser session answer alike. Each filter, order and date boundary,
// a snooze whose time has come, the rows' trimming, the other roles and refusals and their order, the locks and the
// interleavings are the bootstrap tests' and the permission matrix's.

const pending = { status: "pending", snoozed_till: null, duplicate_to_id: null };

test("H9 (API): Web's admin, a member and a guest list its intake, every status, an accepted work item too, filtered by status and by creator, ordered by number and paged with its total, the guest his own alone; each counts the pending work items of his intakes, which a submission and a triage change; a submitter changes his work item, a guest its name and description alone, a declined one no more; deleted, a pending work item is gone, an accepted one stays, listed no more; a personal access token and a browser session alike", async ({
  api,
}, testInfo) => {
  const [alice, bob, gus] = await Promise.all([
    newAccount(api, testInfo, "alice"),
    newAccount(api, testInfo, "bob"),
    newAccount(api, testInfo, "gus"),
  ]);
  const slug = slugFor(testInfo);
  await createWorkspace(api, alice.token, { name: "Acme", slug });
  await inviteAndAccept(api, alice.token, slug, bob, 15);
  await inviteAndAccept(api, alice.token, slug, gus, 5);
  // Web, of which bob is a member and gus a guest, shows its guests only their own work items, as a project does
  // unless its admin says otherwise; Ops is alice's alone. Each has its intake turned on.
  const web = await createProject(api, alice.token, slug, { name: "Web", identifier: "WEB" });
  const ops = await createProject(api, alice.token, slug, { name: "Ops", identifier: "OPS" });
  await addProjectMembers(api, alice.token, web.id, [
    { member_id: bob.id, role: 15 },
    { member_id: gus.id, role: 5 },
  ]);
  await Promise.all(
    [web, ops].map(async (project) =>
      expect(
        await changeProject(api, alice.token, project.id, { intake_view: true }),
        `${project.name}'s intake turned on by alice`
      ).toMatchObject({ status: 200, project: { intake_view: true } })
    )
  );
  // newAccount's token is its personal access token; each one's browser session besides.
  const session = async ({ email }: { email: string }) => (await login(api, email)).access_token;
  const [aliceSession, bobSession, gusSession] = await Promise.all([session(alice), session(bob), session(gus)]);

  // What each counts while Web holds webs pending work items, his of them gus's: alice, Web's and Ops' none, in id
  // order, with her personal access token and her browser session alike; bob, a member of Web alone, Web's; gus, his
  // own of Web's.
  const countsAre = async (when: string, webs: number, his: number) => {
    const [alices, alicesBySession, bobs, guss] = await Promise.all([
      intakeCounts(api, alice.token, slug),
      intakeCounts(api, aliceSession, slug),
      intakeCounts(api, bob.token, slug),
      intakeCounts(api, gusSession, slug),
    ]);
    const inIdOrder = [
      { project_id: web.id, pending: webs },
      { project_id: ops.id, pending: 0 },
    ].toSorted((a, b) => (a.project_id < b.project_id ? -1 : 1));
    expect(alices, `alice's counts ${when}, with her personal access token`).toEqual({ data: inIdOrder });
    expect(alicesBySession, `alice's counts ${when}, with her browser session`).toEqual(alices);
    expect(bobs, `bob's counts ${when}`).toEqual({ data: [{ project_id: web.id, pending: webs }] });
    expect(guss, `gus's counts ${when}`).toEqual({ data: [{ project_id: web.id, pending: his }] });
  };
  await countsAre("before a submission", 0, 0);

  // bob and gus submit five work items to Web's intake, with either credential, two of them gus's: five pending.
  const breaks = await submitIntakeIssue(api, bob.token, web.id, { name: "Login breaks" });
  const slow = await submitIntakeIssue(api, gusSession, web.id, { name: "Login is slow" });
  const dark = await submitIntakeIssue(api, bobSession, web.id, { name: "Dark mode" });
  const spam = await submitIntakeIssue(api, bob.token, web.id, { name: "Buy now" });
  const signup = await submitIntakeIssue(api, gus.token, web.id, { name: "Sign up fails" });
  await countsAre("after five submissions", 5, 2);

  // alice snoozes Dark mode till an hour on, which no request takes to reach nerve, declines Buy now and accepts Sign up
  // fails, gus's: two pending are left, one of them gus's.
  const till = new Date(Date.now() + 3_600_000).toISOString();
  await triageIntakeIssue(api, alice.token, dark.id, { status: "snoozed", snoozed_till: till });
  await triageIntakeIssue(api, alice.token, spam.id, { status: "declined" });
  const accepted = await triageIntakeIssue(api, alice.token, signup.id, { status: "accepted" });
  await countsAre("after alice's triage", 2, 1);

  // alice lists Web's intake, the latest submission first, the accepted one too, alike with either credential, and bob
  // the same; gus lists his own two, as alice's rows show them.
  const listed = await listIntakeIssues(api, alice.token, web.id);
  expectIssuePage(listed, [signup.id, spam.id, dark.id, slow.id, breaks.id]);
  expect(
    listed.data.map(({ name, intake }) => [name, intake.status]),
    "Web's intake, as alice lists it"
  ).toEqual([
    ["Sign up fails", "accepted"],
    ["Buy now", "declined"],
    ["Dark mode", "snoozed"],
    ["Login is slow", "pending"],
    ["Login breaks", "pending"],
  ]);
  expect(await listIntakeIssues(api, aliceSession, web.id), "Web's intake, with alice's browser session").toEqual(
    listed
  );
  expect(await listIntakeIssues(api, bob.token, web.id), "Web's intake, as bob lists it").toEqual(listed);
  expect(await listIntakeIssues(api, gusSession, web.id), "Web's intake, as gus lists it").toEqual({
    total: 2,
    data: listed.data.filter((row) => row.created_by_id === gus.id),
    next_cursor: null,
  });

  // Pending or snoozed, CSV alternatives: the three in the intake still; gus's own one of them; bob's own two, his
  // creation a filter besides.
  const open = { status: ["pending", "snoozed"] };
  expectIssuePage(await listIntakeIssues(api, alice.token, web.id, open), [dark.id, slow.id, breaks.id]);
  expectIssuePage(await listIntakeIssues(api, gusSession, web.id, open), [slow.id]);
  expectIssuePage(await listIntakeIssues(api, bobSession, web.id, { ...open, created_by_id: ["me"] }), [
    dark.id,
    breaks.id,
  ]);

  // By number, two a page, with alice's browser session: three pages, each with the five's total.
  const byNumber = { order_by: "sequence_id", limit: 2 } as const;
  const first = await listIntakeIssues(api, aliceSession, web.id, byNumber);
  expectIssuePage(first, [breaks.id, slow.id], 5, true);
  const second = await listIntakeIssues(api, aliceSession, web.id, {
    ...byNumber,
    cursor: first.next_cursor ?? undefined,
  });
  expectIssuePage(second, [dark.id, spam.id], 5, true);
  const last = await listIntakeIssues(api, aliceSession, web.id, {
    ...byNumber,
    cursor: second.next_cursor ?? undefined,
  });
  expectIssuePage(last, [signup.id], 5);

  // bob changes his Login breaks with his browser session, gus his Login is slow's name and description with his
  // personal access token: each answered as changed, pending still. gus may change no other field; Buy now, declined,
  // changes no more.
  expect(
    await intakeIssueUpdated(api, bobSession, breaks.id, { name: "Login breaks on Safari", priority: "high" }),
    "bob's change of Login breaks"
  ).toMatchObject({
    status: 200,
    issue: { id: breaks.id, name: "Login breaks on Safari", priority: "high", intake: pending },
  });
  expect(
    await intakeIssueUpdated(api, gus.token, slow.id, {
      name: "Login is slow on mobile",
      description_html: "<p>Since Monday</p>",
    }),
    "gus's change of Login is slow"
  ).toMatchObject({
    status: 200,
    issue: { id: slow.id, name: "Login is slow on mobile", description_html: "<p>Since Monday</p>", intake: pending },
  });
  expect(
    await intakeIssueUpdated(api, gusSession, slow.id, { priority: "urgent" }),
    "gus's change of Login is slow's priority"
  ).toEqual({ status: 403, code: "forbidden" });
  expect(
    await intakeIssueUpdated(api, bob.token, spam.id, { name: "Buy now cheap" }),
    "bob's change of Buy now"
  ).toEqual({ status: 409, code: "issue.intake_closed" });

  // bob deletes Login breaks, pending, with his browser session: the work item is gone. alice deletes Sign up fails,
  // accepted, with her personal access token: its intake row alone; the work item reads as it was, but for its intake,
  // and is in Web's list.
  expect(await intakeIssueDeleted(api, bobSession, breaks.id), "bob's deletion of Login breaks").toEqual({
    status: 204,
  });
  expect(await issueRead(api, alice.token, breaks.id), "Login breaks, deleted").toEqual({
    status: 404,
    code: "issue.not_found",
  });
  expect(await intakeIssueDeleted(api, alice.token, signup.id), "alice's deletion of Sign up fails").toEqual({
    status: 204,
  });
  expect(await issueRead(api, aliceSession, signup.id), "Sign up fails, its intake row deleted").toEqual({
    status: 200,
    issue: { ...accepted, intake: null },
  });
  expectIssuePage(await listIssues(api, alice.token, web.id), [signup.id]);

  // Web's intake lists the three left, Login is slow as gus changed it, its priority as it was; one pending is left,
  // gus's.
  const left = await listIntakeIssues(api, alice.token, web.id);
  expectIssuePage(left, [spam.id, dark.id, slow.id]);
  expect(left.data[2], "Login is slow, listed").toMatchObject({ name: "Login is slow on mobile", priority: "none" });
  await countsAre("after the deletions", 1, 1);
});
